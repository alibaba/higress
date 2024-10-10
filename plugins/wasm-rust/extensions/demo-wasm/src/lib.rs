use higress_wasm_rust::cluster_wrapper::DnsCluster;
use higress_wasm_rust::log::Log;
use higress_wasm_rust::plugin_wrapper::{
    HttpCallArgStorage, HttpCallbackFn, HttpContextWrapper, RootContextWrapper,
};
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use http::Method;
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Bytes, ContextType, DataAction, HeaderAction, LogLevel};

use serde::Deserialize;
use std::cell::RefCell;
use std::ops::DerefMut;
use std::rc::Rc;
use std::time::Duration;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(DemoWasmRoot::new()));
}}

const PLUGIN_NAME: &str = "demo-wasm";

#[derive(Default, Debug, Deserialize, Clone)]
struct DemoWasmConfig {
    // 配置文件结构体
    test: String,
}

fn format_body(body: Option<Vec<u8>>) -> String {
    if let Some(bd) = &body {
        if let Ok(b) = std::str::from_utf8(bd) {
            return b.to_string();
        }
    }
    format!("{:?}", body)
}

fn test_callback(
    this: &mut DemoWasm,
    status_code: u16,
    headers: &MultiMap<String, String>,
    body: Option<Vec<u8>>,
) {
    this.log.info(&format!(
        "test_callback status_code:{}, headers: {:?}, body: {}",
        status_code,
        headers,
        format_body(body)
    ));
    this.reset_http_request();
}
struct DemoWasm {
    // 每个请求对应的插件实例
    log: Log,
    config: Option<DemoWasmConfig>,

    arg_storage: HttpCallArgStorage<Box<HttpCallbackFn<DemoWasm>>>,
}

impl Context for DemoWasm {}
impl HttpContext for DemoWasm {}
impl HttpContextWrapper<DemoWasmConfig, Box<HttpCallbackFn<DemoWasm>>> for DemoWasm {
    fn log(&self) -> &Log {
        &self.log
    }
    fn get_http_call_storage(
        &mut self,
    ) -> Option<&mut HttpCallArgStorage<Box<HttpCallbackFn<DemoWasm>>>> {
        Some(&mut self.arg_storage)
    }
    fn on_config(&mut self, config: &DemoWasmConfig) {
        // 获取config
        self.log.info(&format!("on_config {}", config.test));
        self.config = Some(config.clone())
    }
    fn on_http_request_complete_headers(
        &mut self,
        headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        // 请求header获取完成回调
        self.log
            .info(&format!("on_http_request_complete_headers {:?}", headers));
        HeaderAction::Continue
    }
    fn on_http_response_complete_headers(
        &mut self,
        headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        // 返回header获取完成回调
        self.log
            .info(&format!("on_http_response_complete_headers {:?}", headers));
        HeaderAction::Continue
    }
    fn cache_request_body(&self) -> bool {
        // 是否缓存请求body
        true
    }
    fn cache_response_body(&self) -> bool {
        // 是否缓存返回body
        true
    }
    fn on_http_call_response_detail(
        &mut self,
        _token_id: u32,
        arg: Box<HttpCallbackFn<DemoWasm>>,
        status_code: u16,
        headers: &MultiMap<String, String>,
        body: Option<Vec<u8>>,
    ) {
        arg(self, status_code, headers, body)
    }
    fn on_http_request_complete_body(&mut self, req_body: &Bytes) -> DataAction {
        // 请求body获取完成回调
        self.log.info(&format!(
            "on_http_request_complete_body {}",
            String::from_utf8(req_body.clone()).unwrap_or("".to_string())
        ));
        let cluster = DnsCluster::new("httpbin", "httpbin.org", 80);
        if self
            .http_call(
                &cluster,
                &Method::POST,
                "http://httpbin.org/post",
                MultiMap::new(),
                Some("test_body".as_bytes()),
                // Box::new(move |this, _status_code, _headers, _body|  this.resume_http_request()),
                Box::new(test_callback),
                Duration::from_secs(5),
            )
            .is_ok()
        {
            DataAction::StopIterationAndBuffer
        } else {
            self.log.info("http_call fail");
            DataAction::Continue
        }
    }
    fn on_http_response_complete_body(&mut self, res_body: &Bytes) -> DataAction {
        // 返回body获取完成回调
        self.log.info(&format!(
            "on_http_response_complete_body {}",
            String::from_utf8(res_body.clone()).unwrap_or("".to_string())
        ));
        DataAction::Continue
    }
}
struct DemoWasmRoot {
    log: Log,
    rule_matcher: SharedRuleMatcher<DemoWasmConfig>,
}
impl DemoWasmRoot {
    fn new() -> Self {
        let log = Log::new(PLUGIN_NAME.to_string());
        log.info("DemoWasmRoot::new");
        DemoWasmRoot {
            log,
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::default())),
        }
    }
}

impl Context for DemoWasmRoot {}

impl RootContext for DemoWasmRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        self.log.info("DemoWasmRoot::on_configure");
        on_configure(
            self,
            _plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }
    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        self.log.info(&format!(
            "DemoWasmRoot::create_http_context({})",
            context_id
        ));
        self.create_http_context_use_wrapper(context_id)
    }
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl RootContextWrapper<DemoWasmConfig, Box<HttpCallbackFn<DemoWasm>>> for DemoWasmRoot {
    fn rule_matcher(&self) -> &SharedRuleMatcher<DemoWasmConfig> {
        &self.rule_matcher
    }

    fn create_http_context_wrapper(
        &self,
        _context_id: u32,
    ) -> Option<Box<dyn HttpContextWrapper<DemoWasmConfig, Box<HttpCallbackFn<DemoWasm>>>>> {
        Some(Box::new(DemoWasm {
            config: None,
            log: Log::new(PLUGIN_NAME.to_string()),
            arg_storage: HttpCallArgStorage::new(),
        }))
    }
}
