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

    pub fn push(&mut self, data: &[u8], is_openai: bool) {
        if is_openai {
            self.stream_parser.update(data.to_vec());
            for event in self.stream_parser.by_ref() {
                if event.is_empty() || !event.starts_with(b"data:") {
                    continue;
                }
                if let Ok(res) = serde_json::from_slice::<Res>(&event[b"data:".len()..]) {
                    for choice in &res.choices {
                        if let Some(delta) = &choice.delta {
                            self.message.extend(delta.content.as_bytes());
                        }
                    }
                    self.usage.add(&res.usage);
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
                    "data:{}\r\n",
                    json!({"choices": [{"index": 0, "message": {"role": "assistant", "content": ret}}], "usage": usage})
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
        if self.message.is_empty() {
            Vec::new()
        } else if is_openai {
            format!(
                "data:{}\n\n",
                json!({"choices": [{"index": 0, "delta": {"role": "assistant", "content": self.message}}], "usage": self.usage})
            ).as_bytes().to_vec()
        } else {
            self.message.clone()
        }
    }
}
