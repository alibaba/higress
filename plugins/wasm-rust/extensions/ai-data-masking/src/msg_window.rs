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

#[derive(Default)]
pub(crate) struct MessageWindow {
    message: Vec<u8>,
}

impl MessageWindow {
    pub(crate) fn update(&mut self, data: &[u8]) {
        self.message.extend(data);
    }

    pub(crate) fn pop(&mut self, char_window_size: usize, byte_window_size: usize) -> Vec<u8> {
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
            ret.as_bytes().to_vec()
        } else {
            let ret = self.message[..self.message.len() - byte_window_size].to_vec();
            self.message = self.message[self.message.len() - byte_window_size..].to_vec();
            ret
        }
    }
    pub(crate) fn finish(&mut self) -> Vec<u8> {
        core::mem::take(&mut self.message)
    }
    pub(crate) fn iter_mut(&mut self) -> impl Iterator<Item = &mut Vec<u8>> {
        std::iter::once(&mut self.message)
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_msg_window() {
        let mut msg_window = super::MessageWindow::default();
        msg_window.update(b"hello world");
        assert_eq!(msg_window.pop(5, 5), b"hello ");
        assert_eq!(msg_window.pop(5, 5), b"");
        msg_window.update(b"hello world");
        assert_eq!(msg_window.pop(5, 5), b"worldhello ");
        assert_eq!(msg_window.finish(), b"world");
    }
}
