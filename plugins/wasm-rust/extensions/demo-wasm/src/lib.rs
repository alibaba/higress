use higress_wasm_rust::cluster_wrapper::{DnsCluster, StaticIpCluster};
use higress_wasm_rust::log::Log;
use higress_wasm_rust::plugin_wrapper::{HttpContextWrapper, RootContextWrapper};
use higress_wasm_rust::redis_wrapper::{RedisClient, RedisClientBuilder, RedisClientConfig};
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use http::Method;
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Bytes, ContextType, DataAction, HeaderAction, LogLevel};

use redis::Value;
use serde::Deserialize;
use serde_json::json;
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
    #[serde(default)]
    password: Option<String>,
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
    redis_client: Option<RedisClient>,
    cid: i64,
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
        self.config = Some(config.clone());
    }
    fn on_http_request_complete_headers(
        &mut self,
        headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        // 请求header获取完成回调
        self.log
            .info(&format!("on_http_request_complete_headers {:?}", headers));
        if let Some(config) = &self.config {
            let _redis_client = RedisClientBuilder::new(
                &StaticIpCluster::new("redis", 80, ""),
                Duration::from_secs(5),
            )
            .password(config.password.as_ref())
            .build();

            let redis_client = RedisClient::new(
                RedisClientConfig::new(
                    &StaticIpCluster::new("redis", 80, ""),
                    Duration::from_secs(5),
                )
                .password(config.password.as_ref()),
            );

            if let Some(self_rc) = self.weak.upgrade() {
                let init_res = redis_client.init();
                self.log.info(&format!("redis init {:?}", init_res));
                if init_res.is_ok() {
                    let incr_res = redis_client.incr(
                        "connect",
                        Box::new(move |res, status, token_id| {
                            self_rc.borrow().log().info(&format!(
                                "redis incr finish value_res:{:?}, status: {}, token_id: {}",
                                res, status, token_id
                            ));
                            if let Some(this) = self_rc.borrow_mut().downcast_mut::<DemoWasm>() {
                                if let Ok(Value::Int(value)) = res {
                                    this.cid = *value;
                                }
                            }
                            self_rc.borrow().resume_http_request();
                        }),
                    );
                    match incr_res {
                        Ok(s) => {
                            self.log.info(&format!("redis incr ok {}", s));
                            return HeaderAction::StopAllIterationAndBuffer;
                        }
                        Err(e) => self.log.info(&format!("redis incr error {:?}", e)),
                    };
                }
                self.redis_client = Some(redis_client);
            } else {
                self.log.error("self_weak upgrade error");
            }
        }

        HeaderAction::Continue
    }
    fn on_http_response_complete_headers(
        &mut self,
        headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        // 返回header获取完成回调
        self.log
            .info(&format!("on_http_response_complete_headers {:?}", headers));
        self.set_http_response_header("Content-Length", None);
        let self_rc = match self.weak.upgrade() {
            Some(rc) => rc.clone(),
            None => {
                self.log.error("self_weak upgrade error");
                return HeaderAction::Continue;
            }
        };
        if let Some(redis_client) = &self.redis_client {
            match redis_client.get(
                "connect",
                Box::new(move |res, status, token_id| {
                    if let Some(this) = self_rc.borrow().downcast_ref::<DemoWasm>() {
                        this.log.info(&format!(
                            "redis get connect value_res:{:?}, status: {}, token_id: {}",
                            res, status, token_id
                        ));
                        this.resume_http_response();
                    } else {
                        self_rc.borrow().resume_http_response();
                    }
                }),
            ) {
                Ok(o) => {
                    self.log.info(&format!("redis get ok {}", o));
                    return HeaderAction::StopIteration;
                }
                Err(e) => {
                    self.log.info(&format!("redis get fail {:?}", e));
                }
            }
        }
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
        DataAction::Continue
    }
    fn on_http_response_complete_body(&mut self, res_body: &Bytes) -> DataAction {
        // 返回body获取完成回调
        let res_body_string = String::from_utf8(res_body.clone()).unwrap_or("".to_string());
        self.log.info(&format!(
            "on_http_response_complete_body {}",
            res_body_string
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
                if let Some(this) = self_rc.borrow_mut().downcast_mut::<DemoWasm>() {
                    let body_string = format_body(body);
                    this.log.info(&format!(
                        "test_callback status_code:{}, headers: {:?}, body: {}",
                        status_code,
                        headers,
                        body_string
                    ));
                    let data = json!({"redis_cid": this.cid, "http_call_body": body_string, "res_body": res_body_string});
                    this.replace_http_response_body(data.to_string().as_bytes());
                    this.resume_http_response();
                } else {
                    self_rc.borrow().resume_http_response();
                }
            }),
            Duration::from_secs(5),
        );
        match http_call_res {
            Ok(_) => return DataAction::StopIterationAndBuffer,
            Err(e) => {
                self.log.info(&format!("http_call fail {:?}", e));
            }
        }
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
            redis_client: None,
            cid: -1,
        }))
    }
}
