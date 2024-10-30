use std::collections::HashSet;

use cedarwood::Cedar;
use jieba_rs::Jieba;

use crate::Asset;

pub(crate) trait DenyWord {
    fn check(&self, message: &str) -> Option<String>;
}

#[derive(Debug, Clone)]
pub(crate) struct DenyWordAccurate {
    records: Vec<String>,
    cedar: Cedar,
}
#[warn(dead_code)]
impl DenyWordAccurate {
    pub(crate) fn empty() -> Self {
        DenyWordAccurate {
            records: Vec::new(),
            cedar: Cedar::new(),
        }
    }

    pub(crate) fn from_iter<T: IntoIterator<Item = impl Into<String>>>(words: T) -> Self {
        let mut deny_word = DenyWordAccurate::empty();

        for word in words {
            deny_word.add_word(word.into());
        }

        deny_word
    }

    fn add_word(&mut self, word: String) -> bool {
        if self.cedar.exact_match_search(&word).is_none() {
            self.cedar.update(&word, self.records.len() as i32);
            self.records.push(word);
            true
        } else {
            false
        }
    }
}

impl Default for DenyWordAccurate {
    fn default() -> Self {
        Self::empty()
    }
}

impl DenyWord for DenyWordAccurate {
    fn check(&self, message: &str) -> Option<String> {
        for (start_pos, _) in message.char_indices() {
            if let Some(res) = self.cedar.common_prefix_search(&message[start_pos..]) {
                if !res.is_empty() {
                    return Some(self.records[res[0].0 as usize].clone());
                }
            }
        }
        None
    }
}

#[derive(Default, Debug, Clone)]
pub(crate) struct DenyWordJieba {
    jieba: Jieba,
    words: HashSet<String>,
}

impl DenyWordJieba {
    pub(crate) fn from_iter<T: IntoIterator<Item = impl Into<String>>>(words: T) -> Self {
        let mut deny_word = DenyWordJieba::default();

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
        DenyWordJieba {
            jieba: Jieba::empty(),
            words: HashSet::new(),
        }
    }

    pub(crate) fn new() -> Self {
        if let Some(file) = Asset::get("sensitive_word_dict.txt") {
            if let Ok(data) = std::str::from_utf8(file.data.as_ref()) {
                return DenyWordJieba::from_iter(data.split('\n'));
            }
        }
        Self::empty()
    }
}

impl DenyWord for DenyWordJieba {
    fn check(&self, message: &str) -> Option<String> {
        for word in self.jieba.cut(message, true) {
            if self.words.contains(word) {
                return Some(word.to_string());
            }
        }
        None
    }
}
