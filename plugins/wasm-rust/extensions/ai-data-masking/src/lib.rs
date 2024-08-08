// Copyright (c) 2023 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use fancy_regex::Regex;
use grok::patterns;
use higress_wasm_rust::log::Log;
use higress_wasm_rust::plugin_wrapper::{HttpContextWrapper, RootContextWrapper};
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use jieba_rs::Jieba;
use lazy_static::lazy_static;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, Bytes, ContextType, LogLevel};
use rust_embed::Embed;
use serde::de::Error;
use serde::Deserialize;
use serde::Deserializer;
use serde_json::{json, Value};
use std::cell::RefCell;
use std::collections::{BTreeMap, HashMap, HashSet, VecDeque};
use std::ops::DerefMut;
use std::rc::Rc;
use std::{usize, vec};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(AiDataMaskingRoot::new()));
}}

const PLUGIN_NAME: &str = "ai-data-masking";
const GROK_PATTERN: &str = r"%\{(?<name>(?<pattern>[A-z0-9]+)(?::(?<alias>[A-z0-9_:;\/\s\.]+))?)\}";

#[derive(Embed)]
#[folder = "res/"]
struct Asset;

struct System {
    jieba: Jieba,
    words: HashSet<String>,
    grok_regex: Regex,
    grok_patterns: BTreeMap<String, String>,
}
lazy_static! {
    static ref SYSTEM: System = System::new();
}

struct AiDataMaskingRoot {
    log: Log,
    rule_matcher: SharedRuleMatcher<AiDataMaskingConfig>,
}
struct AiDataMasking {
    config: Option<AiDataMaskingConfig>,
    mask_map: HashMap<String, Option<String>>,
    stream: bool,
}
fn deserialize_regexp<'de, D>(deserializer: D) -> Result<Regex, D::Error>
where
    D: Deserializer<'de>,
{
    let value: Value = Deserialize::deserialize(deserializer)?;
    if let Some(pattern) = value.as_str() {
        let (p, _) = SYSTEM.grok_to_pattern(pattern);
        if let Ok(reg) = Regex::new(&p) {
            Ok(reg)
        } else if let Ok(reg) = Regex::new(pattern) {
            Ok(reg)
        } else {
            Err(Error::custom(format!("regexp error field {}", pattern)))
        }
    } else {
        Err(Error::custom("regexp error not string".to_string()))
    }
}

fn deserialize_type<'de, D>(deserializer: D) -> Result<Type, D::Error>
where
    D: Deserializer<'de>,
{
    let value: Value = Deserialize::deserialize(deserializer)?;
    if let Some(_type) = value.as_str() {
        if _type == "replace" {
            Ok(Type::Replace)
        } else if _type == "hash" {
            Ok(Type::Hash)
        } else {
            Err(Error::custom(format!("regexp error value {}", _type)))
        }
    } else {
        Err(Error::custom("type error not string".to_string()))
    }
}
#[derive(Debug, Clone, PartialEq, Eq)]
enum Type {
    Replace,
    Hash,
}

#[derive(Debug, Deserialize, Clone)]
struct Rule {
    #[serde(deserialize_with = "deserialize_regexp")]
    regex: Regex,
    #[serde(deserialize_with = "deserialize_type", alias = "type")]
    type_: Type,
    #[serde(default)]
    restore: bool,
    #[serde(default)]
    value: String,
}
fn default_system_deny() -> bool {
    true
}
fn default_deny_code() -> u16 {
    200
}
fn default_deny_message() -> String {
    "提问或回答中包含敏感词，已被屏蔽".to_string()
}
#[derive(Default, Debug, Deserialize, Clone)]
pub struct AiDataMaskingConfig {
    #[serde(default = "default_system_deny")]
    system_deny: bool,
    #[serde(default = "default_deny_code")]
    deny_code: u16,
    #[serde(default = "default_deny_message")]
    deny_message: String,
    #[serde(default)]
    replace_roles: Vec<Rule>,
    #[serde(default)]
    deny_words: Vec<String>,
}

#[derive(Debug, Deserialize, Clone)]
struct Message {
    #[serde(default)]
    content: String,
    #[serde(default)]
    role: String,
}
#[derive(Debug, Deserialize, Clone)]
struct Req {
    #[serde(default)]
    stream: bool,
    #[serde(default)]
    messages: Vec<Message>,
}

#[derive(Default, Debug, Deserialize)]
struct ResMessage {
    #[serde(default)]
    message: Option<Message>,
    #[serde(default)]
    delta: Option<Message>,
    #[serde(default)]
    finish_reason: String,
}
#[derive(Default, Debug, Deserialize)]
struct Res {
    #[serde(default)]
    choices: Vec<ResMessage>,
}

static SYSTEM_PATTERNS: &[(&str, &str)] = &[
    ("MOBILE", r#"\d{8,11}"#),
    ("IDCARD", r#"\d{17}[0-9xX]|\d{15}"#),
];

impl System {
    fn new() -> Self {
        let jieba = Jieba::empty();
        let words = HashSet::new();

        let grok_regex = Regex::new(GROK_PATTERN).unwrap();
        let grok_patterns = BTreeMap::new();
        let mut system = System {
            jieba,
            words,
            grok_regex,
            grok_patterns,
        };
        system.init();
        system
    }
    fn init(&mut self) {
        if let Some(file) = Asset::get("sensitive_word_dict.txt") {
            if let Ok(data) = std::str::from_utf8(file.data.as_ref()) {
                for word in data.split('\n') {
                    let w = word.trim();
                    if w.is_empty() {
                        continue;
                    }
                    self.jieba.add_word(w, None, None);
                    self.words.insert(w.to_string());
                }
            }
        }
        let mut grok_temp_patterns = VecDeque::new();
        for patterns in [patterns(), SYSTEM_PATTERNS] {
            for &(key, value) in patterns {
                if self.grok_regex.is_match(value).is_ok_and(|r| r) {
                    grok_temp_patterns.push_back((String::from(key), String::from(value)));
                } else {
                    self.grok_patterns
                        .insert(String::from(key), String::from(value));
                }
            }
        }
        let mut last_ok: Option<String> = None;

        while let Some((key, value)) = grok_temp_patterns.pop_front() {
            if let Some(k) = &last_ok {
                if k == &key {
                    break;
                }
            }
            let (v, ok) = self.grok_to_pattern(&value);
            if ok {
                self.grok_patterns.insert(key, v);
                last_ok = None;
            } else {
                if last_ok.is_none() {
                    last_ok = Some(key.clone());
                }
                grok_temp_patterns.push_back((key, v));
            }
        }
    }
    fn grok_to_pattern(&self, pattern: &str) -> (String, bool) {
        let mut ok = true;
        let mut ret = pattern.to_string();
        for _c in self.grok_regex.captures_iter(pattern) {
            if _c.is_err() {
                ok = false;
                continue;
            }
            let c = _c.unwrap();
            if let (Some(full), Some(name)) = (c.get(0), c.name("pattern")) {
                if let Some(p) = self.grok_patterns.get(name.as_str()) {
                    if let Some(alias) = c.name("alias") {
                        ret = ret.replace(full.as_str(), &format!("(?P<{}>{})", alias.as_str(), p));
                    } else {
                        ret = ret.replace(full.as_str(), p);
                    }
                } else {
                    ok = false;
                }
            }
        }
        (ret, ok)
    }
    fn check(&self, message: &str) -> bool {
        for word in self.jieba.cut(message, true) {
            if self.words.contains(word) {
                return true;
            }
        }
        false
    }
}
impl AiDataMaskingRoot {
    fn new() -> Self {
        AiDataMaskingRoot {
            log: Log::new(PLUGIN_NAME.to_string()),
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::default())),
        }
    }
}

impl Context for AiDataMaskingRoot {}

impl RootContext for AiDataMaskingRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        on_configure(
            self,
            _plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }
    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        self.create_http_context_use_wrapper(_context_id)
    }
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl RootContextWrapper<AiDataMaskingConfig> for AiDataMaskingRoot {
    fn rule_matcher(&self) -> &SharedRuleMatcher<AiDataMaskingConfig> {
        &self.rule_matcher
    }

    fn create_http_context_wrapper(
        &self,
        _context_id: u32,
    ) -> Option<Box<dyn HttpContextWrapper<AiDataMaskingConfig>>> {
        Some(Box::new(AiDataMasking {
            mask_map: HashMap::new(),
            config: None,
            stream: false,
        }))
    }
}
impl AiDataMasking {
    fn check_message(&self, message: &str) -> bool {
        if let Some(config) = &self.config {
            for deny_word in &config.deny_words {
                if message.contains(deny_word) {
                    return true;
                }
            }
            if config.system_deny {
                return SYSTEM.check(message);
            }
        }
        false
    }
    fn msg_to_response(&self, msg: &str) -> (String, String) {
        if self.stream {
            (
                format!(
                    "data:{}\n\n",
                    json!({"choices": [{"index": 0, "delta": {"role": "assistant", "content": msg}}], "usage": {}})
                ),
                "text/event-stream;charset=UTF-8".to_string(),
            )
        } else {
            (
                json!({"choices": [{"index": 0, "message": {"role": "assistant", "content": msg}}], "usage": {}}).to_string(),
                 "application/json".to_string()
            )
        }
    }
    fn deny(&self) -> Action {
        let (deny_code, (deny_message, content_type)) = if let Some(config) = &self.config {
            (config.deny_code, self.msg_to_response(&config.deny_message))
        } else {
            (
                default_deny_code(),
                self.msg_to_response(&default_deny_message()),
            )
        };
        self.send_http_response(
            deny_code as u32,
            vec![("Content-Type", &content_type)],
            Some(deny_message.as_bytes()),
        );
        Action::Pause
    }
    fn process_sse_message(&mut self, sse_message: &str) -> (String, String, String, String) {
        let mut message = String::new();
        let mut last_line = "";
        let mut role = String::new();
        let mut finish_reason = String::new();
        for msg in sse_message.split('\n') {
            if !msg.starts_with("data:") {
                continue;
            }
            let res: Res = if let Some(m) = msg.strip_prefix("data:") {
                last_line = m;

                match serde_json::from_str(m) {
                    Ok(r) => r,
                    Err(_) => return (String::new(), last_line.to_string(), role, finish_reason),
                }
            } else {
                continue;
            };

            if res.choices.is_empty() {
                continue;
            }
            for choice in &res.choices {
                if !choice.finish_reason.is_empty() {
                    finish_reason = choice.finish_reason.to_string();
                }
                if let Some(delta) = &choice.delta {
                    if !delta.role.is_empty() {
                        role.clone_from(&delta.role);
                    }
                    message.push_str(&delta.content)
                }
            }
        }

        (message, last_line.to_string(), role, finish_reason)
    }
}

impl Context for AiDataMasking {}
impl HttpContext for AiDataMasking {}
impl HttpContextWrapper<AiDataMaskingConfig> for AiDataMasking {
    fn on_config(&mut self, _config: &AiDataMaskingConfig) {
        self.config = Some(_config.clone());
    }
    fn cache_request_body(&self) -> bool {
        true
    }
    fn cache_response_body(&self) -> bool {
        true
    }
    fn on_http_request_body_ok(&mut self, req_body: &Bytes) -> Action {
        if self.config.is_none() {
            return Action::Continue;
        }
        let config = self.config.as_ref().unwrap();

        let mut req_body = match String::from_utf8(req_body.clone()) {
            Ok(r) => r,
            Err(_) => return Action::Continue,
        };
        let req: Req = match serde_json::from_str(req_body.as_str()) {
            Ok(r) => r,
            Err(_) => return Action::Continue,
        };
        self.stream = req.stream;
        for msg in req.messages {
            if self.check_message(&msg.content) {
                return self.deny();
            }
            if config.replace_roles.is_empty() {
                continue;
            }
            let mut to_content = msg.content.clone();
            for rule in &config.replace_roles {
                let mut replace_pair = Vec::new();
                if rule.type_ == Type::Replace && !rule.restore {
                    to_content = rule.regex.replace_all(&to_content, &rule.value).to_string();
                } else {
                    for _m in rule.regex.find_iter(&to_content) {
                        if _m.is_err() {
                            continue;
                        }
                        let m = _m.unwrap();
                        let from_word = m.as_str();

                        let to_word = match rule.type_ {
                            Type::Hash => {
                                let digest = md5::compute(from_word.as_bytes());
                                format!("{:x}", digest)
                            }
                            Type::Replace => rule.regex.replace(from_word, &rule.value).to_string(),
                        };
                        replace_pair.push((from_word.to_string(), to_word.clone()));

                        if rule.restore && !to_word.is_empty() {
                            match self.mask_map.entry(to_word) {
                                std::collections::hash_map::Entry::Occupied(mut e) => {
                                    e.insert(None);
                                }
                                std::collections::hash_map::Entry::Vacant(e) => {
                                    e.insert(Some(from_word.to_string()));
                                }
                            }
                        }
                    }
                    for (from_word, to_word) in replace_pair {
                        to_content = to_content.replace(&from_word, &to_word);
                    }
                }
            }

            if to_content != msg.content.as_str() {
                if let (Ok(from), Ok(to)) = (
                    serde_json::to_string(&msg.content),
                    serde_json::to_string(&to_content),
                ) {
                    req_body = req_body.replace(&from, &to);
                }
            }
        }
        self.replace_http_request_body(req_body.as_bytes());
        Action::Continue
    }
    fn on_http_response_body_ok(&mut self, res_body: &Bytes) -> Action {
        let mut res_body = match String::from_utf8(res_body.clone()) {
            Ok(r) => r,
            Err(_) => return Action::Continue,
        };

        let new_body = if self.stream {
            let (mut message, last_line, role, finish_reason) = self.process_sse_message(&res_body);

            if self.check_message(&message) {
                return self.deny();
            }
            if self.mask_map.is_empty() {
                return Action::Continue;
            }
            let mut value: Value = match serde_json::from_str(last_line.as_str()) {
                Ok(r) => r,
                Err(_) => return Action::Continue,
            };
            if let Some(obj) = value.as_object_mut() {
                for (from_word, to_word) in self.mask_map.iter() {
                    if let Some(to) = to_word {
                        message = message.replace(from_word, to);
                    }
                }
                let msg = json!([{"index": 0, "delta": {"content": message, "role": role}, "finish_reason": finish_reason}]);
                obj.insert("choices".to_string(), msg);
            }
            format!("data:{}\n\n", value)
        } else {
            let res: Res = match serde_json::from_str(res_body.as_str()) {
                Ok(r) => r,
                Err(_) => return Action::Continue,
            };
            for msg in res.choices {
                if let Some(meesage) = msg.message {
                    if self.check_message(&meesage.content) {
                        return self.deny();
                    }

                    if self.mask_map.is_empty() {
                        continue;
                    }
                    let mut m = meesage.content.clone();
                    for (from_word, to_word) in self.mask_map.iter() {
                        if let Some(to) = to_word {
                            m = m.replace(from_word, to);
                        }
                    }
                    if m != meesage.content {
                        if let (Ok(from), Ok(to)) = (
                            serde_json::to_string(&meesage.content),
                            serde_json::to_string(&m),
                        ) {
                            res_body = res_body.replace(&from, &to);
                        }
                    }
                }
            }
            res_body
        };
        self.replace_http_response_body(new_body.as_bytes());
        Action::Continue
    }
}
