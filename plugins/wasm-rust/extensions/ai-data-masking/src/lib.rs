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
use jsonpath_rust::{JsonPath, JsonPathValue};
use lazy_static::lazy_static;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Bytes, ContextType, DataAction, HeaderAction, LogLevel};
use rust_embed::Embed;
use serde::de::Error;
use serde::Deserialize;
use serde::Deserializer;
use serde_json::{json, Value};
use std::cell::RefCell;
use std::collections::{BTreeMap, HashMap, HashSet, VecDeque};
use std::ops::DerefMut;
use std::rc::Rc;
use std::str::FromStr;
use std::vec;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(AiDataMaskingRoot::new()));
}}

const PLUGIN_NAME: &str = "ai-data-masking";
const GROK_PATTERN: &str = r"%\{(?<name>(?<pattern>[A-z0-9]+)(?::(?<alias>[A-z0-9_:;\/\s\.]+))?)\}";

#[derive(Embed)]
#[folder = "res/"]
struct Asset;

#[derive(Default, Debug, Clone)]
struct DenyWord {
    jieba: Jieba,
    words: HashSet<String>,
}
struct System {
    deny_word: DenyWord,
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
    is_openai: bool,
    stream: bool,
    res_body: Bytes,
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

fn deserialize_denyword<'de, D>(deserializer: D) -> Result<DenyWord, D::Error>
where
    D: Deserializer<'de>,
{
    let value: Vec<String> = Deserialize::deserialize(deserializer)?;
    Ok(DenyWord::from_iter(value))
}

fn deserialize_jsonpath<'de, D>(deserializer: D) -> Result<Vec<JsonPath>, D::Error>
where
    D: Deserializer<'de>,
{
    let value: Vec<String> = Deserialize::deserialize(deserializer)?;
    let mut ret = Vec::new();
    for v in value {
        if v.is_empty() {
            continue;
        }
        match JsonPath::from_str(&v) {
            Ok(jp) => ret.push(jp),
            Err(_) => return Err(Error::custom(format!("jsonpath error value {}", v))),
        }
    }
    Ok(ret)
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
fn default_deny_openai() -> bool {
    true
}
fn default_deny_raw() -> bool {
    false
}
fn default_system_deny() -> bool {
    true
}
fn default_deny_code() -> u16 {
    200
}
fn default_deny_content_type() -> String {
    "application/json".to_string()
}
fn default_deny_raw_message() -> String {
    "{\"errmsg\":\"提问或回答中包含敏感词，已被屏蔽\"}".to_string()
}
fn default_deny_message() -> String {
    "提问或回答中包含敏感词，已被屏蔽".to_string()
}
#[derive(Default, Debug, Deserialize, Clone)]
pub struct AiDataMaskingConfig {
    #[serde(default = "default_deny_openai")]
    deny_openai: bool,
    #[serde(default = "default_deny_raw")]
    deny_raw: bool,
    #[serde(default, deserialize_with = "deserialize_jsonpath")]
    deny_jsonpath: Vec<JsonPath>,
    #[serde(default = "default_system_deny")]
    system_deny: bool,
    #[serde(default = "default_deny_code")]
    deny_code: u16,
    #[serde(default = "default_deny_message")]
    deny_message: String,
    #[serde(default = "default_deny_raw_message")]
    deny_raw_message: String,
    #[serde(default = "default_deny_content_type")]
    deny_content_type: String,
    #[serde(default)]
    replace_roles: Vec<Rule>,
    #[serde(deserialize_with = "deserialize_denyword", default = "DenyWord::empty")]
    deny_words: DenyWord,
}

#[derive(Debug, Deserialize, Clone)]
struct Message {
    content: String,
}
#[derive(Debug, Deserialize, Clone)]
struct Req {
    #[serde(default)]
    stream: bool,
    messages: Vec<Message>,
}

#[derive(Default, Debug, Deserialize)]
struct ResMessage {
    #[serde(default)]
    message: Option<Message>,
    #[serde(default)]
    delta: Option<Message>,
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

impl DenyWord {
    fn empty() -> Self {
        DenyWord {
            jieba: Jieba::empty(),
            words: HashSet::new(),
        }
    }
    fn from_iter<T: IntoIterator<Item = impl Into<String>>>(words: T) -> Self {
        let mut deny_word = DenyWord::empty();

        for word in words {
            let _w = word.into();
            let w = _w.trim();
            if w.is_empty() {
                continue;
            }
            deny_word.jieba.add_word(w, None, None);
            deny_word.words.insert(w.to_string());
        }

        deny_word
    }
    fn default() -> Self {
        if let Some(file) = Asset::get("sensitive_word_dict.txt") {
            if let Ok(data) = std::str::from_utf8(file.data.as_ref()) {
                return DenyWord::from_iter(data.split('\n'));
            }
        }
        DenyWord::empty()
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
impl System {
    fn new() -> Self {
        let grok_regex = Regex::new(GROK_PATTERN).unwrap();
        let grok_patterns = BTreeMap::new();
        let mut system = System {
            deny_word: DenyWord::default(),
            grok_regex,
            grok_patterns,
        };
        system.init();
        system
    }
    fn init(&mut self) {
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
            is_openai: false,
            stream: false,
            res_body: Bytes::new(),
        }))
    }
}
impl AiDataMasking {
    fn check_message(&self, message: &str) -> bool {
        if let Some(config) = &self.config {
            config.deny_words.check(message)
                || (config.system_deny && SYSTEM.deny_word.check(message))
        } else {
            false
        }
    }
    fn msg_to_response(&self, msg: &str, raw_msg: &str, content_type: &str) -> (String, String) {
        if !self.is_openai {
            (raw_msg.to_string(), content_type.to_string())
        } else if self.stream {
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
    fn deny(&mut self, in_response: bool) -> DataAction {
        if in_response && self.stream {
            self.replace_http_response_body(&[]);
            return DataAction::Continue;
        }
        let (deny_code, (deny_message, content_type)) = if let Some(config) = &self.config {
            (
                config.deny_code,
                self.msg_to_response(
                    &config.deny_message,
                    &config.deny_raw_message,
                    &config.deny_content_type,
                ),
            )
        } else {
            (
                default_deny_code(),
                self.msg_to_response(
                    &default_deny_message(),
                    &default_deny_raw_message(),
                    &default_deny_content_type(),
                ),
            )
        };
        if in_response {
            self.replace_http_response_body(deny_message.as_bytes());
            return DataAction::Continue;
        }
        self.send_http_response(
            deny_code as u32,
            vec![("Content-Type", &content_type)],
            Some(deny_message.as_bytes()),
        );
        DataAction::StopIterationAndBuffer
    }

    fn process_sse_message(&mut self, sse_message: &str) -> Vec<String> {
        let mut messages = Vec::new();
        for msg in sse_message.split('\n') {
            if !msg.starts_with("data:") {
                continue;
            }
            let res: Res = if let Some(m) = msg.strip_prefix("data:") {
                match serde_json::from_str(m) {
                    Ok(r) => r,
                    Err(_) => continue,
                }
            } else {
                continue;
            };

            if res.choices.is_empty() {
                continue;
            }
            for choice in &res.choices {
                if let Some(delta) = &choice.delta {
                    messages.push(delta.content.clone());
                }
            }
        }
        messages
    }
    fn replace_request_msg(&mut self, message: &str) -> String {
        let config = self.config.as_ref().unwrap();
        let mut msg = message.to_string();
        for rule in &config.replace_roles {
            let mut replace_pair = Vec::new();
            if rule.type_ == Type::Replace && !rule.restore {
                msg = rule.regex.replace_all(&msg, &rule.value).to_string();
            } else {
                for _m in rule.regex.find_iter(&msg) {
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
                    msg = msg.replace(&from_word, &to_word);
                }
            }
        }
        msg
    }
}

impl Context for AiDataMasking {}
impl HttpContext for AiDataMasking {
    fn on_http_request_headers(
        &mut self,
        _num_headers: usize,
        _end_of_stream: bool,
    ) -> HeaderAction {
        HeaderAction::StopIteration
    }
    fn on_http_response_headers(
        &mut self,
        _num_headers: usize,
        _end_of_stream: bool,
    ) -> HeaderAction {
        self.set_http_response_header("Content-Length", None);
        HeaderAction::Continue
    }
    fn on_http_response_body(&mut self, body_size: usize, _end_of_stream: bool) -> DataAction {
        if !self.stream {
            return DataAction::Continue;
        }
        if let Some(body) = self.get_http_response_body(0, body_size) {
            self.res_body.extend(&body);

            if let Ok(body_str) = String::from_utf8(self.res_body.clone()) {
                if self.is_openai {
                    let messages = self.process_sse_message(&body_str);

                    if self.check_message(&messages.join("")) {
                        return self.deny(true);
                    }
                } else if self.check_message(&body_str) {
                    return self.deny(true);
                }
            }
            if self.mask_map.is_empty() {
                return DataAction::Continue;
            }
            if let Ok(body_str) = std::str::from_utf8(&body) {
                let mut new_str = body_str.to_string();
                if self.is_openai {
                    let messages = self.process_sse_message(body_str);

                    for message in messages {
                        let mut new_message = message.clone();
                        for (from_word, to_word) in self.mask_map.iter() {
                            if let Some(to) = to_word {
                                new_message = new_message.replace(from_word, to);
                            }
                        }
                        if new_message != message {
                            new_str = new_str.replace(
                                &json!(message).to_string(),
                                &json!(new_message).to_string(),
                            );
                        }
                    }
                } else {
                    for (from_word, to_word) in self.mask_map.iter() {
                        if let Some(to) = to_word {
                            new_str = new_str.replace(from_word, to);
                        }
                    }
                }
                if new_str != body_str {
                    self.replace_http_response_body(new_str.as_bytes());
                }
            }
        }
        DataAction::Continue
    }
}
impl HttpContextWrapper<AiDataMaskingConfig> for AiDataMasking {
    fn on_config(&mut self, config: &AiDataMaskingConfig) {
        self.config = Some(config.clone());
    }
    fn cache_request_body(&self) -> bool {
        true
    }
    fn cache_response_body(&self) -> bool {
        !self.stream
    }
    fn on_http_request_complete_body(&mut self, req_body: &Bytes) -> DataAction {
        if self.config.is_none() {
            return DataAction::Continue;
        }
        let config = self.config.as_ref().unwrap();

        let mut req_body = match String::from_utf8(req_body.clone()) {
            Ok(r) => r,
            Err(_) => return DataAction::Continue,
        };
        if config.deny_openai {
            if let Ok(r) = serde_json::from_str(req_body.as_str()) {
                let req: Req = r;
                self.is_openai = true;
                self.stream = req.stream;
                for msg in req.messages {
                    if self.check_message(&msg.content) {
                        return self.deny(false);
                    }
                    let new_content = self.replace_request_msg(&msg.content);
                    if new_content != msg.content {
                        if let (Ok(from), Ok(to)) = (
                            serde_json::to_string(&msg.content),
                            serde_json::to_string(&new_content),
                        ) {
                            req_body = req_body.replace(&from, &to);
                        }
                    }
                }
                self.replace_http_request_body(req_body.as_bytes());
                return DataAction::Continue;
            }
        }
        if !config.deny_jsonpath.is_empty() {
            if let Ok(r) = serde_json::from_str(req_body.as_str()) {
                let json: Value = r;
                for jsonpath in config.deny_jsonpath.clone() {
                    for v in jsonpath.find_slice(&json) {
                        if let JsonPathValue::Slice(d, _) = v {
                            if let Some(s) = d.as_str() {
                                if self.check_message(s) {
                                    return self.deny(false);
                                }
                                let content = s.to_string();
                                let new_content = self.replace_request_msg(&content);
                                if new_content != content {
                                    if let (Ok(from), Ok(to)) = (
                                        serde_json::to_string(&content),
                                        serde_json::to_string(&new_content),
                                    ) {
                                        req_body = req_body.replace(&from, &to);
                                    }
                                }
                            }
                        }
                    }
                }
                self.replace_http_request_body(req_body.as_bytes());
                return DataAction::Continue;
            }
        }
        if config.deny_raw {
            if self.check_message(&req_body) {
                return self.deny(false);
            }
            let new_body = self.replace_request_msg(&req_body);
            if new_body != req_body {
                self.replace_http_request_body(new_body.as_bytes())
            }
            return DataAction::Continue;
        }
        DataAction::Continue
    }
    fn on_http_response_complete_body(&mut self, res_body: &Bytes) -> DataAction {
        if self.config.is_none() {
            self.reset_http_response();
            return DataAction::Continue;
        }
        let config = self.config.as_ref().unwrap();
        let mut res_body = match String::from_utf8(res_body.clone()) {
            Ok(r) => r,
            Err(_) => {
                self.reset_http_response();
                return DataAction::Continue;
            }
        };
        if config.deny_openai && self.is_openai {
            if let Ok(r) = serde_json::from_str(res_body.as_str()) {
                let res: Res = r;
                for msg in res.choices {
                    if let Some(meesage) = msg.message {
                        if self.check_message(&meesage.content) {
                            return self.deny(true);
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
                self.replace_http_response_body(res_body.as_bytes());

                return DataAction::Continue;
            }
        }
        if config.deny_raw {
            if self.check_message(&res_body) {
                return self.deny(true);
            }
            if !self.mask_map.is_empty() {
                for (from_word, to_word) in self.mask_map.iter() {
                    if let Some(to) = to_word {
                        res_body = res_body.replace(from_word, to);
                    }
                }
            }
            self.replace_http_response_body(res_body.as_bytes());
            return DataAction::Continue;
        }
        DataAction::Continue
    }
}
