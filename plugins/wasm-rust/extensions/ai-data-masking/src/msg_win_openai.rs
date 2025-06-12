// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

use std::collections::HashMap;

use higress_wasm_rust::event_stream::EventStream;
use serde::Deserialize;
use serde_json::Value;

use crate::msg_window::MessageWindow;
use crate::number_merge::NumberMerge;

#[derive(PartialEq, Eq, Clone, Copy)]
enum MsgFlag {
    None,
    Content,
    ReasoningContent,
}
impl Default for MsgFlag {
    fn default() -> Self {
        Self::None
    }
}
#[derive(Deserialize)]
struct Delta {
    #[serde(default)]
    content: Option<String>,
    #[serde(default)]
    reasoning_content: Option<String>,
}
#[derive(Deserialize)]
struct Choices {
    #[serde(default)]
    index: i64,
    #[serde(default)]
    delta: Option<Delta>,
    #[serde(default)]
    finish_reason: Option<String>,
}

impl Delta {
    fn get_flag_msg(&self, default_flag: &MsgFlag) -> (MsgFlag, &[u8]) {
        if let Some(msg) = &self.content {
            if !msg.is_empty() {
                return (MsgFlag::Content, msg.as_bytes());
            }
        }
        if let Some(msg) = &self.reasoning_content {
            if !msg.is_empty() {
                return (MsgFlag::ReasoningContent, msg.as_bytes());
            }
        }
        (*default_flag, &[])
    }
}
const USAGE_PATH: &str = "usage";
const CHOICES_PATH: &str = "choices";

type MessageLine = Vec<(MsgFlag, Vec<u8>)>;

#[derive(Default)]
struct MessageWindowOpenAi {
    message_window: MessageWindow,
    ret_messages: MessageLine,
    flag: MsgFlag,
    last_value: Value,
    finish_reason: Option<String>,
}

impl MessageWindowOpenAi {
    fn update(
        &mut self,
        data: &[u8],
        flag: MsgFlag,
        value: &Value,
        finish_reason: &Option<String>,
    ) {
        self.last_value = value.clone();
        if data.is_empty() {
            return;
        }
        if self.flag == MsgFlag::None {
            self.flag = flag;
        }
        if self.flag != flag {
            let last_flag = core::mem::replace(&mut self.flag, flag);
            let msg = self.message_window.finish();
            self.ret_messages.push((last_flag, msg));
        }
        self.message_window.update(data);
        if let Some(fr) = finish_reason {
            self.finish_reason = Some(fr.clone());
        }
    }

    fn gen_value(&self, flag: &MsgFlag, msg: &[u8], finish: bool) -> Value {
        let mut ret = self.last_value.clone();
        match flag {
            MsgFlag::Content => {
                ret["delta"]["content"] = Value::String(String::from_utf8_lossy(msg).to_string());
                if let Some(m) = ret["delta"].as_object_mut() {
                    m.remove("reasoning_content");
                }
            }
            MsgFlag::ReasoningContent => {
                ret["delta"]["reasoning_content"] =
                    Value::String(String::from_utf8_lossy(msg).to_string());
                ret["delta"]["content"] = Value::String(String::new());
            }
            _ => {}
        }
        if finish {
            ret["finish_reason"] = self
                .finish_reason
                .as_ref()
                .map_or(Value::Null, |v| Value::String(v.to_string()));
        } else {
            ret["finish_reason"] = Value::Null;
        }
        ret
    }

    fn messages_to_value(&mut self) -> Vec<Value> {
        let mut ret = Vec::new();
        for (flag, msg) in core::mem::take(&mut self.ret_messages) {
            ret.push(self.gen_value(&flag, &msg, false));
        }
        ret
    }

    fn pop(&mut self, char_window_size: usize, byte_window_size: usize) -> Vec<Value> {
        let mut ret = self.messages_to_value();

        let msg = self.message_window.pop(char_window_size, byte_window_size);
        if !msg.is_empty() {
            ret.push(self.gen_value(&self.flag, &msg, false));
        }

        ret
    }
    fn finish(&mut self) -> Vec<Value> {
        let mut ret = self.messages_to_value();
        let msg = self.message_window.finish();
        let flag = core::mem::replace(&mut self.flag, MsgFlag::None);
        ret.push(self.gen_value(&flag, &msg, true));

        ret
    }
    fn iter_mut(&mut self) -> impl Iterator<Item = &mut Vec<u8>> {
        self.ret_messages
            .iter_mut()
            .map(|(_, msg)| msg)
            .chain(self.message_window.iter_mut())
    }
}

#[derive(Default)]
pub(crate) struct MsgWindow {
    stream_parser: EventStream,
    base_message_window: MessageWindow,
    message_windows: HashMap<i64, MessageWindowOpenAi>,
    last_value: Value,
    usage: NumberMerge,
}

impl MsgWindow {
    fn update_event(&mut self, event: Vec<u8>) -> Option<Vec<u8>> {
        if event.is_empty() || !event.starts_with(b"data:") {
            Some(event)
        } else if let Ok(res) = serde_json::from_slice::<Value>(&event[b"data:".len()..]) {
            self.last_value = res;
            if let Some(r) = self.last_value.as_object() {
                if let Some(v) = r.get(USAGE_PATH) {
                    self.usage.add(v);
                }
                if let Some(v) = r.get(CHOICES_PATH) {
                    if let Some(a) = v.as_array() {
                        for item in a {
                            if let Ok(c) = serde_json::from_value::<Choices>(item.clone()) {
                                if let Some(d) = &c.delta {
                                    let mw = self.message_windows.entry(c.index).or_default();
                                    let (flag, msg) = d.get_flag_msg(&mw.flag);
                                    mw.update(msg, flag, item, &c.finish_reason);
                                }
                            }
                        }
                    }
                }
            }
            None
        } else if event.starts_with(b"data: [DONE]") {
            None
        } else {
            Some(event)
        }
    }
    fn push_base(&mut self, data: &[u8]) {
        self.base_message_window.update(data);
    }
    pub(crate) fn push(&mut self, data: &[u8], is_openai: bool) {
        if is_openai {
            self.stream_parser.update(data.to_vec());
            while let Some(event) = self.stream_parser.next() {
                if let Some(msg) = self.update_event(event) {
                    self.push_base(&msg);
                }
            }
        } else {
            self.push_base(data);
        }
    }

    pub(crate) fn pop(
        &mut self,
        char_window_size: usize,
        byte_window_size: usize,
        is_openai: bool,
    ) -> Vec<u8> {
        if !is_openai {
            return self
                .base_message_window
                .pop(char_window_size, byte_window_size);
        }
        let mut ret = Vec::new();
        for mw in self.message_windows.values_mut() {
            for value in mw.pop(char_window_size, byte_window_size) {
                let usage = self.usage.finish();
                let mut ret_value = self.last_value.clone();
                ret_value[CHOICES_PATH] = Value::Array(vec![value]);
                ret_value[USAGE_PATH] = usage;
                ret.extend(format!("data: {}\n\n", ret_value).as_bytes())
            }
        }
        ret
    }
    pub(crate) fn finish(&mut self, is_openai: bool) -> Vec<u8> {
        if !is_openai {
            return self.base_message_window.finish();
        }
        if let Some(event) = self.stream_parser.flush() {
            self.update_event(event);
        }
        let mut ret = Vec::new();
        for mw in &mut self.message_windows.values_mut() {
            for value in mw.finish() {
                let usage = self.usage.finish();
                let mut ret_value = self.last_value.clone();
                ret_value[CHOICES_PATH] = Value::Array(vec![value]);
                ret_value[USAGE_PATH] = usage;
                ret.extend(format!("data: {}\n\n", ret_value).as_bytes())
            }
        }
        ret
    }
    pub(crate) fn messages_iter_mut(&mut self) -> impl Iterator<Item = &mut Vec<u8>> {
        self.base_message_window.iter_mut().chain(
            self.message_windows
                .values_mut()
                .flat_map(|mw| mw.iter_mut()),
        )
    }
}

#[cfg(test)]
mod tests {

    #[derive(Deserialize)]
    struct Res {
        choices: Vec<Choices>,
    }

    impl Res {
        fn get_text(&self) -> (String, String) {
            let mut content = String::new();
            let mut reasoning_content = String::new();
            for choice in self.choices.iter() {
                if let Some(delta) = &choice.delta {
                    if let Some(c) = &delta.content {
                        content += c;
                    }
                    if let Some(rc) = &delta.reasoning_content {
                        reasoning_content += rc;
                    }
                }
            }
            (content, reasoning_content)
        }
    }
    use super::*;

    #[test]
    fn test_msg() {
        let mut msg_win = MsgWindow::default();
        let data = raw_message();
        let mut buffer = Vec::new();
        for line in data.split("\n") {
            msg_win.push(line.as_bytes(), true);
            msg_win.push(b"\n\n", true);
            for message in msg_win.messages_iter_mut() {
                if let Ok(mut msg) = String::from_utf8(message.clone()) {
                    msg = msg.replace("Higress", "***higress***");
                    message.clear();
                    message.extend_from_slice(msg.as_bytes());
                }
            }

            buffer.extend(msg_win.pop(7, 7, true));
        }
        buffer.extend(msg_win.finish(true));
        let mut message = String::new();
        let mut reasoning_message = String::new();
        for line in buffer.split(|&x| x == b'\n') {
            if line.is_empty() {
                continue;
            }
            assert!(line.starts_with(b"data:"));
            if line.starts_with(b"data: [DONE]") {
                continue;
            }
            let des = serde_json::from_slice::<Res>(&line[b"data:".len()..]);
            assert!(des.is_ok());
            let res = des.unwrap();
            let (c, rc) = res.get_text();
            message.push_str(&c);
            reasoning_message.push_str(&rc);
        }
        let res = "***higress*** 是一个基于 Istio 的高性能服务网格数据平面项目，旨在提供高吞吐量、低延迟和可扩展的服务通信管理。它为企业级应用提供了丰富的流量治理功能，如负载均衡、熔断、限流等，并支持多协议代理（包括 HTTP/1.1, HTTP/2, gRPC）。***higress*** 的设计目标是优化 Istio 在大规模集群中的性能表现，满足高并发场景下的需求。";
        assert_eq!(message, res);
        assert_eq!(reasoning_message, res);
    }

    fn raw_message() -> String {
        let msg = r#"data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"H"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"ig"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"ress"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" 是"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"一个"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"基于"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" Ist"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"io"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" 的"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"高性能"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"服务"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"网格"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"数据"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"平面"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"项目"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"，"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"旨在"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"提供"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"高"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"吞"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"吐"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"量"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"、"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"低"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"延迟"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"和"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"可"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"扩展"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"的服务"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"通信"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"管理"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"。"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"它"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"为企业"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"级"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"应用"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"提供了"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"丰富的"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"流量"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"治理"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"功能"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"，"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"如"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"负载"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"均衡"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"、"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"熔"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"断"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"、"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"限"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"流"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872010,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"等"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"，并"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"支持"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"多"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"协议"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"代理"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"（"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"包括"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" HTTP"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"/"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"1"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"."},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"1"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":","},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" HTTP"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"/"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"2"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":","},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" g"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"RPC"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"）。"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"H"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"ig"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"ress"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" 的"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"设计"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"目标"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"是"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"优化"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" Ist"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"io"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":" 在"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"大规模"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"集群"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872011,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"中的"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"性能"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"表现"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"，"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"满足"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"高"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"并发"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"场景"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"下的"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"需求"},"finish_reason":null}]}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"。"},"finish_reason":null}]}"#;
        msg.replace("\"content\":", "\"reasoning_content\":")
            + "\n"
            + msg
            + r#"
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":null,"finish_reason":"stop"}],"usage":null}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":1,"delta":{},"finish_reason":"stop"}],"usage":{}}
data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{}}],"usage":{"prompt_tokens":372,"completion_tokens":9,"total_tokens":381}}
data: [DONE]"#
    }
}
