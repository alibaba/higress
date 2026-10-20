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

use std::collections::BTreeMap;

use higress_wasm_rust::event_stream::EventStream;
#[cfg(test)]
use serde::Deserialize;
use serde_json::Value;

use crate::msg_window::MessageWindow;

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
#[cfg(test)]
#[derive(Deserialize)]
struct Delta {
    #[serde(default)]
    content: Option<String>,
    #[serde(default)]
    reasoning_content: Option<String>,
}
#[cfg(test)]
#[derive(Deserialize)]
struct Choices {
    #[serde(default)]
    delta: Option<Delta>,
}

const CHOICES_PATH: &str = "choices";

type MessageLine = Vec<(MsgFlag, Vec<u8>)>;

#[derive(Default)]
struct MessageWindowOpenAi {
    message_window: MessageWindow,
    ret_messages: MessageLine,
    flag: MsgFlag,
    last_value: Value,
}

impl MessageWindowOpenAi {
    fn update(&mut self, data: &[u8], flag: MsgFlag, value: &Value) {
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
    }

    fn gen_value(&self, flag: &MsgFlag, msg: &[u8]) -> Value {
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
        ret["finish_reason"] = Value::Null;
        ret
    }

    fn messages_to_value(&mut self) -> Vec<Value> {
        let mut ret = Vec::new();
        for (flag, msg) in core::mem::take(&mut self.ret_messages) {
            ret.push(self.gen_value(&flag, &msg));
        }
        ret
    }

    fn pop(&mut self, char_window_size: usize, byte_window_size: usize) -> Vec<Value> {
        let mut ret = self.messages_to_value();

        let msg = self.message_window.pop(char_window_size, byte_window_size);
        if !msg.is_empty() {
            ret.push(self.gen_value(&self.flag, &msg));
        }

        ret
    }
    fn finish(&mut self) -> Vec<Value> {
        let mut ret = self.messages_to_value();
        let msg = self.message_window.finish();
        let flag = core::mem::replace(&mut self.flag, MsgFlag::None);
        if !msg.is_empty() {
            ret.push(self.gen_value(&flag, &msg));
        }

        ret
    }
    fn iter_mut(&mut self) -> impl Iterator<Item = &mut Vec<u8>> {
        self.ret_messages
            .iter_mut()
            .map(|(_, msg)| msg)
            .chain(self.message_window.iter_mut())
    }
}

enum QueuedEvent {
    Text {
        envelope: Value,
        window: MessageWindowOpenAi,
    },
    Raw(Vec<u8>),
}

#[derive(Default)]
pub(crate) struct MsgWindow {
    stream_parser: EventStream,
    base_message_window: MessageWindow,
    message_windows: BTreeMap<i64, MessageWindowOpenAi>,
    last_value: Value,
    queued_events: Vec<QueuedEvent>,
    done_received: bool,
}

impl MsgWindow {
    fn queue_text(&mut self) {
        for (_, window) in core::mem::take(&mut self.message_windows) {
            self.queued_events.push(QueuedEvent::Text {
                envelope: self.last_value.clone(),
                window,
            });
        }
    }

    fn queue_raw(&mut self, mut event: Vec<u8>) {
        event.extend_from_slice(b"\n\n");
        self.queued_events.push(QueuedEvent::Raw(event));
    }

    fn update_event(&mut self, event: Vec<u8>) -> bool {
        if event.is_empty() {
            return false;
        }
        if event.strip_prefix(b"data:").is_some_and(|data| {
            data.iter()
                .copied()
                .filter(|b| !b.is_ascii_whitespace())
                .eq(b"[DONE]".iter().copied())
        }) {
            self.queue_text();
            if !self.done_received {
                self.queue_raw(b"data: [DONE]".to_vec());
                self.done_received = true;
            }
            return false;
        }
        let Ok(res) = event
            .strip_prefix(b"data:")
            .ok_or(())
            .and_then(|data| serde_json::from_slice::<Value>(data).map_err(|_| ()))
        else {
            self.queue_text();
            self.queue_raw(event);
            return false;
        };
        let Some(choices) = res[CHOICES_PATH].as_array().filter(|a| !a.is_empty()) else {
            self.queue_text();
            self.queue_raw(event);
            return false;
        };
        // Tool-only and terminal events do not need text rewriting.
        if choices.iter().all(|choice| {
            ["content", "reasoning_content"].iter().all(|field| {
                choice["delta"][field]
                    .as_str()
                    .is_none_or(|text| text.is_empty())
            })
        }) {
            self.queue_text();
            self.queue_raw(event);
            return false;
        }

        self.last_value = res.clone();
        let mut controls = Vec::new();
        for choice in choices {
            let mut control = choice.clone();
            let Some(delta) = choice["delta"].as_object() else {
                controls.push(control);
                continue;
            };
            let mut text_choice = choice.clone();
            if let Some(object) = text_choice.as_object_mut() {
                object.retain(|key, _| ["index", "delta", "finish_reason"].contains(&key.as_str()));
            }
            text_choice["delta"] = Value::Object(
                delta
                    .iter()
                    .filter(|(key, _)| {
                        ["role", "content", "reasoning_content"].contains(&key.as_str())
                    })
                    .map(|(key, value)| (key.clone(), value.clone()))
                    .collect(),
            );
            let index = choice["index"].as_i64().unwrap_or_default();
            let mut has_text = false;
            for (field, flag) in [
                ("reasoning_content", MsgFlag::ReasoningContent),
                ("content", MsgFlag::Content),
            ] {
                if let Some(text) = delta
                    .get(field)
                    .and_then(Value::as_str)
                    .filter(|s| !s.is_empty())
                {
                    has_text = true;
                    self.message_windows.entry(index).or_default().update(
                        text.as_bytes(),
                        flag,
                        &text_choice,
                    );
                }
                if let Some(object) = control["delta"].as_object_mut() {
                    if object
                        .get(field)
                        .is_some_and(|value| value.is_string() || value.is_null())
                    {
                        object.remove(field);
                    }
                }
            }
            if has_text {
                if let Some(object) = control["delta"].as_object_mut() {
                    object.remove("role");
                }
            }
            let has_control = control["delta"]
                .as_object()
                .is_some_and(|object| !object.is_empty())
                || !control["finish_reason"].is_null()
                || control.as_object().is_some_and(|object| {
                    object.iter().any(|(key, value)| {
                        !["index", "delta", "finish_reason"].contains(&key.as_str())
                            && !value.is_null()
                    })
                });
            if !has_text || has_control {
                controls.push(control);
            }
        }
        // Flush earlier text before forwarding tool calls or terminal metadata.
        // Keep those events out of the text matcher, including tool arguments.
        if !controls.is_empty() || !res["usage"].is_null() {
            self.queue_text();
            let mut control_event = res;
            control_event[CHOICES_PATH] = Value::Array(controls);
            self.queue_raw(format!("data: {}", control_event).into_bytes());
        }
        true
    }

    fn write_values(output: &mut Vec<u8>, envelope: &Value, values: Vec<Value>) {
        for value in values {
            let mut event = envelope.clone();
            event[CHOICES_PATH] = Value::Array(vec![value]);
            if let Some(object) = event.as_object_mut() {
                object.remove("usage");
            }
            output.extend(format!("data: {}\n\n", event).as_bytes());
        }
    }

    fn drain_queued(&mut self) -> Vec<u8> {
        let mut output = Vec::new();
        for event in core::mem::take(&mut self.queued_events) {
            match event {
                QueuedEvent::Raw(raw) => output.extend(raw),
                QueuedEvent::Text {
                    envelope,
                    mut window,
                } => {
                    Self::write_values(&mut output, &envelope, window.finish());
                }
            }
        }
        output
    }
    fn push_base(&mut self, data: &[u8]) {
        self.base_message_window.update(data);
    }
    pub(crate) fn push(&mut self, data: &[u8], is_openai: bool) {
        if is_openai {
            self.stream_parser.update(data.to_vec());
            while let Some(event) = self.stream_parser.next() {
                self.update_event(event);
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
        let mut ret = self.drain_queued();
        for mw in self.message_windows.values_mut() {
            Self::write_values(
                &mut ret,
                &self.last_value,
                mw.pop(char_window_size, byte_window_size),
            );
        }
        ret
    }
    pub(crate) fn finish(&mut self, is_openai: bool) -> Vec<u8> {
        if !is_openai {
            return self.base_message_window.finish();
        }
        self.complete(is_openai);
        self.queue_text();
        self.drain_queued()
    }
    // Return whether the trailing event added text that needs checking.
    pub(crate) fn complete(&mut self, is_openai: bool) -> bool {
        if is_openai {
            if let Some(event) = self.stream_parser.flush() {
                return self.update_event(event);
            }
        }
        false
    }
    pub(crate) fn messages_iter_mut(&mut self) -> impl Iterator<Item = &mut Vec<u8>> {
        self.base_message_window
            .iter_mut()
            .chain(
                self.queued_events
                    .iter_mut()
                    .filter_map(|event| match event {
                        QueuedEvent::Text { window, .. } => Some(window),
                        QueuedEvent::Raw(_) => None,
                    })
                    .flat_map(|window| window.iter_mut()),
            )
            .chain(
                self.message_windows
                    .values_mut()
                    .flat_map(|mw| mw.iter_mut()),
            )
    }
}

#[cfg(test)]
mod tests {

    use std::fmt::Write;

    use rust_embed::Embed;

    use super::*;
    #[derive(Embed)]
    #[folder = "test/"]
    struct Asset;
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

    #[test]
    fn test_msg() {
        let mut msg_win = MsgWindow::default();
        let data = raw_message("raw_message.txt");
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

    fn rewrite_stream(input: &str, fragment_size: usize) -> String {
        let mut window = MsgWindow::default();
        let mut output = Vec::new();
        for fragment in input.as_bytes().chunks(fragment_size) {
            window.push(fragment, true);
            for message in window.messages_iter_mut() {
                *message = String::from_utf8(message.clone())
                    .unwrap()
                    .replace("MASKED", "original")
                    .into_bytes();
            }
            output.extend(window.pop(48, 48, true));
        }
        output.extend(window.finish(true));
        String::from_utf8(output).unwrap()
    }

    fn stream_events(output: &str) -> Vec<Value> {
        output
            .lines()
            .filter_map(|line| line.strip_prefix("data:"))
            .filter(|data| data.trim() != "[DONE]")
            .map(|data| serde_json::from_str(data).unwrap())
            .collect()
    }

    fn encode_events(events: &[Value]) -> String {
        let mut input = String::new();
        for event in events {
            writeln!(&mut input, "data: {}\n", event).unwrap();
        }
        input.push_str("data: [DONE]\n\n");
        input
    }

    #[test]
    fn test_preserve_tool_calls_after_buffered_text() {
        let calls = vec![
            serde_json::json!([{"index":0,"id":"call_read","type":"function","function":{"name":"read","arguments":""}}]),
            serde_json::json!([{"index":1,"id":"call_search","type":"function","function":{"name":"search","arguments":""}}]),
            serde_json::json!([{"index":0,"function":{"arguments":"{\"path\":"}}]),
            serde_json::json!([{"index":1,"function":{"arguments":"{\"query\":"}}]),
            serde_json::json!([{"index":0,"function":{"arguments":"\"README.md\"}"}}]),
            serde_json::json!([{"index":1,"function":{"arguments":"\"demo\"}"}}]),
        ];
        let mut events = vec![
            serde_json::json!({"choices":[{"index":0,"delta":{"content":"Checking files."}}]}),
        ];
        for call in &calls {
            events.push(serde_json::json!({"choices":[{"index":0,"delta":{"tool_calls":call}}]}));
        }
        events.push(
            serde_json::json!({"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}),
        );
        let input = encode_events(&events);
        for size in [1, 19, input.len()] {
            let output = rewrite_stream(&input, size);
            let mut actual = Vec::new();
            let mut content = String::new();
            let mut finishes = Vec::new();
            for event in stream_events(&output) {
                for choice in event["choices"].as_array().unwrap() {
                    content.push_str(choice["delta"]["content"].as_str().unwrap_or_default());
                    if let Some(call) = choice["delta"].get("tool_calls") {
                        assert_eq!(content, "Checking files.");
                        actual.push(call.clone());
                    }
                    if let Some(reason) = choice["finish_reason"].as_str() {
                        finishes.push(reason.to_string());
                    }
                }
            }
            assert_eq!(actual, calls, "fragment size {size}");
            assert_eq!(content, "Checking files.");
            assert_eq!(finishes, vec!["tool_calls"]);
            assert_eq!(output.matches("data: [DONE]").count(), 1);
            assert!(output.ends_with("data: [DONE]\n\n"));
        }
    }

    #[test]
    fn test_preserve_empty_terminal_and_usage_events() {
        let usage = serde_json::json!({"prompt_tokens":10,"completion_tokens":5,"total_tokens":15});
        for terminal in [
            serde_json::json!({"index":0,"delta":{},"finish_reason":"stop"}),
            serde_json::json!({"index":0,"delta":null,"finish_reason":"stop"}),
            serde_json::json!({"index":0,"finish_reason":"stop"}),
        ] {
            let input = encode_events(&[
                serde_json::json!({"choices":[{"index":0,"delta":{"role":"assistant","content":""}}]}),
                serde_json::json!({"choices":[{"index":0,"delta":{"reasoning_content":"Thinking.","content":"Answer."}}]}),
                serde_json::json!({"choices":[terminal.clone()]}),
                serde_json::json!({"choices":[],"usage":usage.clone()}),
            ]);
            for size in [1, 19, input.len()] {
                let output = rewrite_stream(&input, size);
                let events = stream_events(&output);
                let mut content = String::new();
                let mut reasoning = String::new();
                let mut finishes = Vec::new();
                let mut usages = Vec::new();
                for event in &events {
                    if !event["usage"].is_null() {
                        usages.push(event["usage"].clone());
                    }
                    for choice in event["choices"].as_array().unwrap() {
                        content.push_str(choice["delta"]["content"].as_str().unwrap_or_default());
                        reasoning.push_str(
                            choice["delta"]["reasoning_content"]
                                .as_str()
                                .unwrap_or_default(),
                        );
                        if !choice["finish_reason"].is_null() {
                            finishes.push(choice.clone());
                        }
                    }
                }
                assert_eq!(content, "Answer.");
                assert_eq!(reasoning, "Thinking.");
                assert_eq!(finishes, vec![terminal.clone()]);
                assert_eq!(usages, vec![usage.clone()]);
                assert_eq!(output.matches("data: [DONE]").count(), 1);
                assert!(output.ends_with("data: [DONE]\n\n"));
            }
        }
    }

    #[test]
    fn test_restore_text_without_changing_tool_arguments() {
        let call = serde_json::json!([{"index":0,"id":"call_read","type":"function","function":{"name":"read","arguments":"{\"path\":\"MASKED\"}"}}]);
        let input = encode_events(&[
            serde_json::json!({"choices":[{"index":0,"delta":{"content":"MA"}}]}),
            serde_json::json!({"choices":[{"index":0,"delta":{"content":"SKED","tool_calls":call.clone()}}]}),
            serde_json::json!({"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}),
        ]);
        for size in [1, 19, input.len()] {
            let output = rewrite_stream(&input, size);
            let mut content = String::new();
            let mut calls = Vec::new();
            for event in stream_events(&output) {
                for choice in event["choices"].as_array().unwrap() {
                    content.push_str(choice["delta"]["content"].as_str().unwrap_or_default());
                    if let Some(value) = choice["delta"].get("tool_calls") {
                        assert_eq!(content, "original");
                        calls.push(value.clone());
                    }
                }
            }
            assert_eq!(content, "original");
            assert_eq!(calls, vec![call.clone()]);
        }
    }

    #[test]
    fn test_preserve_multiple_choices_and_long_arguments() {
        let arguments = serde_json::json!({"content":"report detail\n".repeat(1000)}).to_string();
        let mut events = vec![serde_json::json!({"choices":[
            {"index":1,"delta":{"tool_calls":[{"index":0,"id":"call_write","type":"function","function":{"name":"write","arguments":""}}]}},
            {"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_read","type":"function","function":{"name":"read","arguments":"{}"}}]}}
        ]})];
        for chunk in arguments.as_bytes().chunks(512) {
            events.push(serde_json::json!({"choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":std::str::from_utf8(chunk).unwrap()}}]}}]}));
        }
        events.push(serde_json::json!({"choices":[
            {"index":1,"delta":{},"finish_reason":"tool_calls"},
            {"index":0,"delta":{},"finish_reason":"tool_calls"}
        ]}));
        let input = encode_events(&events);
        for size in [1, 19, input.len()] {
            let output = rewrite_stream(&input, size);
            let actual: Vec<Value> = stream_events(&output)
                .into_iter()
                .flat_map(|event| event["choices"].as_array().unwrap().clone())
                .collect();
            let expected: Vec<Value> = events
                .iter()
                .flat_map(|event| event["choices"].as_array().unwrap().clone())
                .collect();
            assert_eq!(actual, expected, "fragment size {size}");
        }
    }

    #[test]
    fn test_preserve_error_without_synthesizing_completion() {
        let input =
            "data: {\"error\":{\"message\":\"upstream failure\",\"type\":\"server_error\"}}\n\n";
        for size in [1, 19, input.len()] {
            let output = rewrite_stream(input, size);
            assert_eq!(output, input);
            assert!(!output.contains("finish_reason"));
            assert!(!output.contains("[DONE]"));
        }
    }

    #[test]
    fn test_check_final_event_before_flushing_text() {
        let input = "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"MASKED\"}}]}";
        let mut window = MsgWindow::default();
        window.push(input.as_bytes(), true);
        assert!(window.complete(true));
        assert!(!window.complete(true));
        for message in window.messages_iter_mut() {
            *message = String::from_utf8(message.clone())
                .unwrap()
                .replace("MASKED", "original")
                .into_bytes();
        }
        let output = String::from_utf8(window.finish(true)).unwrap();
        assert_eq!(
            stream_events(&output)[0]["choices"][0]["delta"]["content"],
            "original"
        );
        assert!(!output.contains("[DONE]"));
    }

    fn raw_message(file_name: &str) -> String {
        if let Some(file) = Asset::get(file_name) {
            if let Ok(data) = std::str::from_utf8(file.data.as_ref()) {
                return data.to_string();
            }
        }
        String::new()
    }
}
