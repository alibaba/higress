// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

/// Parsing MIME type text/event-stream according to https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream
///
/// The event stream format is as described by the stream production of the following ABNF
///
/// | rule   | expression                |
/// |--------|---------------------------|
/// |stream  |= [ bom ] *event           |
/// |event   |= *( comment / field ) eol |
/// |comment |= colon *any-char eol      |
/// |field   |= 1*name-char [ colon [ space ] *any-char ] eol |
/// |eol     |= ( cr lf / cr / lf )      |
///
/// According to spec, we must judge EOL twice before we can identify a complete event.
/// However, in the rules of event and field, there is an ambiguous grammar in the judgment of eol,
/// and it will bring ambiguity (whether the field ends). In order to eliminate this ambiguity,
/// we believe that CRLF as CR+LF belongs to event and field respectively.
pub struct EventStream {
    buffer: Vec<u8>,
    processed_offset: usize,
}

impl EventStream {
    pub fn new() -> Self {
        EventStream {
            buffer: Vec::new(),
            processed_offset: 0,
        }
    }

    /// Update the event stream by adding new data to the buffer and resetting processed offset if needed.
    pub fn update(&mut self, data: Vec<u8>) {
        if self.processed_offset > 0 {
            self.buffer.drain(0..self.processed_offset);
            self.processed_offset = 0;
        }

        self.buffer.extend(data);
    }

    /// Get the next event from the event stream. Return the event data if available, otherwise return None.
    /// Next will consume all the data in the current buffer. However, if there is a valid event at the end of the buffer,
    /// it will return the event directly even if the data after the next `update` could be considered part of the same event
    /// (especially in cases where CRLF hits an ambiguous grammar).
    /// When this happens, the next call to next may return an empty event.
    ///
    /// ```
    /// let mut parser = EventStream::new();
    /// parser.update(...);
    /// loop {
    ///     match parser.next() {
    ///         None => {}
    ///         Some(event) => {
    ///             if !event.is_empty() {
    ///                 ...
    ///             }
    ///         }
    ///     }
    /// }
    /// ```
    pub fn next(&mut self) -> Option<Vec<u8>> {
        let mut i = self.processed_offset;

        while i < self.buffer.len() {
            if let Some(size) = self.is_2eol(i) {
                let event = self.buffer[self.processed_offset..i].to_vec();
                self.processed_offset = i + size;
                return Some(event);
            }

            i += 1;
        }

        None
    }

    /// Flush the event stream and return any remaining unprocessed event data. Return None if there is none.
    pub fn flush(&mut self) -> Option<Vec<u8>> {
        if self.processed_offset < self.buffer.len() {
            let remaining_event = self.buffer[self.processed_offset..].to_vec();
            self.processed_offset = self.buffer.len();
            Some(remaining_event)
        } else {
            None
        }
    }

    fn is_eol(&self, i: usize) -> Option<usize> {
        if i + 1 < self.buffer.len() && self.buffer[i] == b'\r' && self.buffer[i + 1] == b'\n' {
            Some(2)
        } else if self.buffer[i] == b'\r' || self.buffer[i] == b'\n' {
            Some(1)
        } else {
            None
        }
    }

    fn is_2eol(&self, i: usize) -> Option<usize> {
        let size1 = match self.is_eol(i) {
            None => return None,
            Some(size1) => size1,
        };
        if i + size1 < self.buffer.len() {
            match self.is_eol(i + size1) {
                None => {
                    if size1 == 2 {
                        Some(2)
                    } else {
                        None
                    }
                }
                Some(size2) => Some(size1 + size2),
            }
        } else if size1 == 2 {
            Some(2)
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_crlf_events() {
        let mut parser = EventStream::new();
        parser.update(b"event1\n\nevent2\n\n".to_vec());

        assert_eq!(parser.next(), Some(b"event1".to_vec()));
        assert_eq!(parser.next(), Some(b"event2".to_vec()));
    }

    #[test]
    fn test_lf_events() {
        let mut parser = EventStream::new();
        parser.update(b"event3\n\r\nevent4\r\n".to_vec());

        assert_eq!(parser.next(), Some(b"event3".to_vec()));
        assert_eq!(parser.next(), Some(b"event4".to_vec()));
    }

    #[test]
    fn test_partial_event() {
        let mut parser = EventStream::new();
        parser.update(b"partial_event1".to_vec());

        assert_eq!(parser.next(), None);

        parser.update(b"\n\n".to_vec());
        assert_eq!(parser.next(), Some(b"partial_event1".to_vec()));
    }

    #[test]
    fn test_mixed_eol_events() {
        let mut parser = EventStream::new();
        parser.update(b"event5\r\nevent6\r\n\r\nevent7\r\n".to_vec());

        assert_eq!(parser.next(), Some(b"event5".to_vec()));
        assert_eq!(parser.next(), Some(b"event6".to_vec()));
        assert_eq!(parser.next(), Some(b"event7".to_vec()));
    }

    #[test]
    fn test_mixed2_eol_events() {
        let mut parser = EventStream::new();
        parser.update(b"event5\r\nevent6\r\n".to_vec());
        assert_eq!(parser.next(), Some(b"event5".to_vec()));
        assert_eq!(parser.next(), Some(b"event6".to_vec()));
        parser.update(b"\r\nevent7\r\n".to_vec());
        assert_eq!(parser.next(), Some(b"".to_vec()));
        assert_eq!(parser.next(), Some(b"event7".to_vec()));
    }

    #[test]
    fn test_no_event() {
        let mut parser = EventStream::new();
        parser.update(b"no_eol_in_this_string".to_vec());

        assert_eq!(parser.next(), None);
        assert_eq!(parser.flush(), Some(b"no_eol_in_this_string".to_vec()));
    }
}
