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

use std::cell::RefCell;
use std::rc::Rc;

enum PromiseState<T> {
    Pending,
    Fulfilled(T),
    Rejected(String),
}

pub struct Promise<T> {
    state: RefCell<PromiseState<T>>,
    then_callback: RefCell<Option<Box<dyn FnOnce(T)>>>,
    catch_callback: RefCell<Option<Box<dyn FnOnce(String)>>>,
}

impl<T> Promise<T>
where
    T: 'static + Clone,
{
    pub fn new() -> Rc<Self> {
        Rc::new(Self {
            state: RefCell::new(PromiseState::Pending),
            then_callback: RefCell::new(None),
            catch_callback: RefCell::new(None),
        })
    }

    pub fn fulfill(self: &Rc<Self>, value: T) {
        *self.state.borrow_mut() = PromiseState::Fulfilled(value.clone());
        if let Some(callback) = self.then_callback.borrow_mut().take() {
            callback(value);
        }
    }

    pub fn reject(self: &Rc<Self>, reason: String) {
        *self.state.borrow_mut() = PromiseState::Rejected(reason.clone());
        if let Some(callback) = self.catch_callback.borrow_mut().take() {
            callback(reason);
        }
    }

    pub fn then<F, R>(self: &Rc<Self>, f: F) -> Rc<Promise<R>>
    where
        F: FnOnce(T) -> R + 'static,
        R: 'static + Clone,
    {
        let new_promise = Promise::new();
        let new_promise_clone = new_promise.clone();
        match &*self.state.borrow() {
            PromiseState::Pending => {
                *self.then_callback.borrow_mut() = Some(Box::new(move |value| {
                    let result = f(value.clone());
                    new_promise_clone.fulfill(result);
                }));
                let new_promise_for_catch = new_promise.clone();
                *self.catch_callback.borrow_mut() = Some(Box::new(move |reason| {
                    new_promise_for_catch.reject(reason);
                }));
            }
            PromiseState::Fulfilled(value) => {
                let result = f(value.clone());
                new_promise.fulfill(result);
            }
            PromiseState::Rejected(reason) => new_promise.reject(reason.clone()),
        }
        new_promise
    }

    pub fn catch<F>(self: &Rc<Self>, f: F) -> Rc<Self>
    where
        F: FnOnce(String) + 'static,
    {
        match &*self.state.borrow() {
            PromiseState::Pending => *self.catch_callback.borrow_mut() = Some(Box::new(f)),
            PromiseState::Fulfilled(_) => {}
            PromiseState::Rejected(reason) => f(reason.clone()),
        }
        self.clone()
    }
}

#[cfg(test)]
mod test {
    use crate::promise::Promise;
    use std::cell::RefCell;
    use std::rc::Rc;

    #[test]
    fn test_fulfill_and_then() {
        let cell = Rc::new(RefCell::new(0));
        let cell_clone = cell.clone();

        let promise = Promise::new();
        promise
            .then(|x| x + x)
            .then(|x| x * x)
            .then(move |x| *cell_clone.borrow_mut() = x);

        promise.fulfill(1);
        assert_eq!(cell.take(), 4)
    }

    #[test]
    fn test_reject_and_catch() {
        let cell = Rc::new(RefCell::new("".to_owned()));
        let cell_clone = cell.clone();

        let promise = Promise::new();
        promise
            .then(|x: i32| x + x)
            .then(|x| x * x)
            .catch(move |reason| *cell_clone.borrow_mut() = reason);

        promise.reject("panic!".to_string());

        assert_eq!("panic!", cell.take())
    }
}
