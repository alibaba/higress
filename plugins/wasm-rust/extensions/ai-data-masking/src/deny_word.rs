use std::collections::HashSet;

use jieba_rs::Jieba;

use crate::Asset;

#[derive(Default, Debug, Clone)]
pub(crate) struct DenyWord {
    jieba: Jieba,
    words: HashSet<String>,
}

impl DenyWord {
    pub(crate) fn from_iter<T: IntoIterator<Item = impl Into<String>>>(words: T) -> Self {
        let mut deny_word = DenyWord::default();

        for word in words {
            let word_s = word.into();
            let w = word_s.trim();
            if w.is_empty() {
                continue;
            }
            deny_word.jieba.add_word(w, None, None);
            deny_word.words.insert(w.to_string());
        }

        deny_word
    }

    pub(crate) fn empty() -> Self {
        DenyWord {
            jieba: Jieba::empty(),
            words: HashSet::new(),
        }
    }

    pub(crate) fn system() -> Self {
        if let Some(file) = Asset::get("sensitive_word_dict.txt") {
            if let Ok(data) = std::str::from_utf8(file.data.as_ref()) {
                return DenyWord::from_iter(data.split('\n'));
            }
        }
        Self::empty()
    }

    pub(crate) fn check(&self, message: &str) -> Option<String> {
        for word in self.jieba.cut(message, true) {
            if self.words.contains(word) {
                return Some(word.to_string());
            }
        }
        None
    }
}
