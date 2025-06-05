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

use serde_json::{json, Number, Value};

fn merge_number(target: &mut Value, add: &Value) {
    if target.is_null() {
        if add.is_object() {
            *target = json!({});
        } else if add.is_number() {
            *target = Value::from(0i64);
        } else {
            return;
        }
    }
    match (target, add) {
        (Value::Object(t), Value::Object(a)) => {
            for (key, value) in a.iter() {
                if let Some(v) = t.get_mut(key) {
                    merge_number(v, value);
                } else {
                    t.insert(key.clone(), value.clone());
                }
            }
        }
        (Value::Number(t), Value::Number(a)) => {
            *t = Number::from(t.as_i64().unwrap_or_default() + a.as_i64().unwrap_or_default());
        }
        _ => {}
    }
}
#[derive(Default, Clone)]
pub(crate) struct NumberMerge {
    value: Value,
}

impl NumberMerge {
    pub(crate) fn add(&mut self, number: &Value) {
        merge_number(&mut self.value, number);
    }
    pub(crate) fn finish(&mut self) -> Value {
        core::mem::replace(&mut self.value, Value::Null)
    }
}
