use higress_wasm_rust::event_stream::EventStream;
use serde_json::json;

use crate::{Res, Usage};

#[derive(Default)]
pub(crate) struct MsgWindow {
    stream_parser: EventStream,
    pub(crate) message: Vec<u8>,
    usage: Usage,
}

impl MsgWindow {
    pub fn new() -> Self {
        MsgWindow::default()
    }

    fn update_event(&mut self, event: Vec<u8>) -> Option<Vec<u8>> {
        if event.is_empty() || !event.starts_with(b"data:") {
            Some(event)
        } else if let Ok(res) = serde_json::from_slice::<Res>(&event[b"data:".len()..]) {
            for choice in &res.choices {
                if let Some(delta) = &choice.delta {
                    self.message.extend(delta.content.as_bytes());
                }
            }
            self.usage.add(&res.usage);
            None
        } else if event.starts_with(b"data: [DONE]") {
            None
        } else {
            Some(event)
        }
    }
    pub fn push(&mut self, data: &[u8], is_openai: bool) {
        if is_openai {
            self.stream_parser.update(data.to_vec());
            while let Some(event) = self.stream_parser.next() {
                if let Some(msg) = self.update_event(event) {
                    self.message.extend(msg);
                }
            }
        } else {
            self.message.extend(data);
        }
    }

    pub fn pop(
        &mut self,
        char_window_size: usize,
        byte_window_size: usize,
        is_openai: bool,
    ) -> Vec<u8> {
        if let Ok(message) = String::from_utf8(self.message.clone()) {
            let chars = message.chars().collect::<Vec<char>>();
            if chars.len() <= char_window_size {
                return Vec::new();
            }
            let ret = chars[..chars.len() - char_window_size]
                .iter()
                .collect::<String>();
            self.message = chars[chars.len() - char_window_size..]
                .iter()
                .collect::<String>()
                .as_bytes()
                .to_vec();

            if is_openai {
                let usage = self.usage.clone();
                self.usage.reset();
                format!(
                    "data: {}\n\n",
                    json!({"choices": [{"index": 0, "delta": {"role": "assistant", "content": ret}}], "usage": usage})
                ).as_bytes().to_vec()
            } else {
                ret.as_bytes().to_vec()
            }
        } else {
            let ret = self.message[..self.message.len() - byte_window_size].to_vec();
            self.message = self.message[self.message.len() - byte_window_size..].to_vec();
            ret
        }
    }

    pub fn finish(&mut self, is_openai: bool) -> Vec<u8> {
        if let Some(event) = self.stream_parser.flush() {
            self.update_event(event);
        }
        if self.message.is_empty() {
            Vec::new()
        } else if is_openai {
            format!(
                "data: {}\n\ndata: [DONE]\n\n",
                json!({"choices": [{"index": 0, "delta": {"role": "assistant", "content": String::from_utf8_lossy(&self.message)}}], "usage": self.usage})
            ).as_bytes().to_vec()
        } else {
            self.message.clone()
        }
    }
}

#[cfg(test)]
mod tests {

    use super::*;

    #[test]
    fn test_msg() {
        let mut msg_win = MsgWindow::default();
        let data = r#"data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872009,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"H"},"finish_reason":null}]}

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

data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":"。"},"finish_reason":null}]}

data: {"id":"chatcmpl-936","object":"chat.completion.chunk","created":1739872012,"model":"qwen2.5-coder:32b","system_fingerprint":"fp_ollama","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":"stop"}]}

data: [DONE]

"#;
        let mut buffer = Vec::new();
        for line in data.split("\n\n") {
            msg_win.push(line.as_bytes(), true);
            msg_win.push(b"\n\n", true);
            if let Ok(mut msg) = String::from_utf8(msg_win.message.clone()) {
                msg = msg.replace("Higress", "***higress***");
                msg_win.message = msg.as_bytes().to_vec();
            }
            buffer.extend(msg_win.pop(7, 7, true));
        }
        buffer.extend(msg_win.finish(true));
        let mut message = String::new();
        for line in buffer.split(|&x| x == b'\n') {
            if line.is_empty() {
                continue;
            }
            assert!(line.starts_with(b"data:"));
            if line.starts_with(b"data: [DONE]") {
                continue;
            }
            let des = serde_json::from_slice(&line[b"data:".len()..]);
            assert!(des.is_ok());
            let res: Res = des.unwrap();
            for choice in &res.choices {
                if let Some(delta) = &choice.delta {
                    message.push_str(&delta.content);
                }
            }
        }
        assert_eq!(message, "***higress*** 是一个基于 Istio 的高性能服务网格数据平面项目，旨在提供高吞吐量、低延迟和可扩展的服务通信管理。它为企业级应用提供了丰富的流量治理功能，如负载均衡、熔断、限流等，并支持多协议代理（包括 HTTP/1.1, HTTP/2, gRPC）。***higress*** 的设计目标是优化 Istio 在大规模集群中的性能表现，满足高并发场景下的需求。");
    }
}
