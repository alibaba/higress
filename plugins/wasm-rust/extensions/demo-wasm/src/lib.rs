use higress_wasm_rust::cluster_wrapper::DnsCluster;
use higress_wasm_rust::log::Log;
use higress_wasm_rust::plugin_wrapper::{HttpContextWrapper, RootContextWrapper};
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use http::Method;
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Bytes, ContextType, DataAction, HeaderAction, LogLevel};

use serde::Deserialize;
use std::cell::RefCell;
use std::ops::DerefMut;
use std::rc::{Rc, Weak};
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

struct DemoWasm {
    // 每个请求对应的插件实例
    log: Log,
    config: Option<Rc<DemoWasmConfig>>,
    weak: Weak<RefCell<Box<dyn HttpContextWrapper<DemoWasmConfig>>>>,
}

impl Context for DemoWasm {}
impl HttpContext for DemoWasm {}
impl HttpContextWrapper<DemoWasmConfig> for DemoWasm {
    fn init_self_weak(
        &mut self,
        self_weak: Weak<RefCell<Box<dyn HttpContextWrapper<DemoWasmConfig>>>>,
    ) {
        self.weak = self_weak;
        self.log.info("init_self_rc");
    }
    fn log(&self) -> &Log {
        &self.log
    }
    fn on_config(&mut self, config: Rc<DemoWasmConfig>) {
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
    fn on_http_request_complete_body(&mut self, req_body: &Bytes) -> DataAction {
        // 请求body获取完成回调
        self.log.info(&format!(
            "on_http_request_complete_body {}",
            String::from_utf8(req_body.clone()).unwrap_or("".to_string())
        ));
        let cluster = DnsCluster::new("httpbin", "httpbin.org", 80);

        let self_rc = match self.weak.upgrade() {
            Some(rc) => rc.clone(),
            None => {
                self.log.error("self_weak upgrade error");
                return DataAction::Continue;
            }
        };
        let http_call_res = self.http_call(
            &cluster,
            &Method::POST,
            "http://httpbin.org/post",
            MultiMap::new(),
            Some("test_body".as_bytes()),
            Box::new(move |status_code, headers, body| {
                if let Some(this) = self_rc.borrow().downcast_ref::<DemoWasm>() {
                    this.log.info(&format!(
                        "test_callback status_code:{}, headers: {:?}, body: {}",
                        status_code,
                        headers,
                        format_body(body)
                    ));
                    this.resume_http_request();
                } else {
                    self_rc.borrow().resume_http_request();
                }
            }),
            Duration::from_secs(5),
        );
        match http_call_res {
            Ok(_) => DataAction::StopIterationAndBuffer,
            Err(e) => {
                self.log.info(&format!("http_call fail {:?}", e));
                DataAction::Continue
            }
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

impl RootContextWrapper<DemoWasmConfig> for DemoWasmRoot {
    fn rule_matcher(&self) -> &SharedRuleMatcher<DemoWasmConfig> {
        &self.rule_matcher
    }

    fn create_http_context_wrapper(
        &self,
        _context_id: u32,
    ) -> Option<Box<dyn HttpContextWrapper<DemoWasmConfig>>> {
        Some(Box::new(DemoWasm {
            config: None,
            log: Log::new(PLUGIN_NAME.to_string()),
            weak: Weak::default(),
        }))
    }
}
