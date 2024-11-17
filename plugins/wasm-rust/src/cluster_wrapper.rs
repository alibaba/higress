use crate::{internal::get_property, request_wrapper::get_request_host};

pub trait Cluster {
    fn cluster_name(&self) -> String;
    fn host_name(&self) -> String;
}
#[derive(Debug, Clone)]
pub struct RouteCluster {
    host: String,
}
impl RouteCluster {
    pub fn new(host: &str) -> Self {
        RouteCluster {
            host: host.to_string(),
        }
    }
}
impl Cluster for RouteCluster {
    fn cluster_name(&self) -> String {
        if let Some(res) = get_property(vec!["cluster_name"]) {
            if let Ok(r) = String::from_utf8(res) {
                return r;
            }
        }
        String::new()
    }

    fn host_name(&self) -> String {
        if !self.host.is_empty() {
            return self.host.clone();
        }

        get_request_host()
    }
}

#[derive(Debug, Clone)]
pub struct K8sCluster {
    service_name: String,
    namespace: String,
    port: String,
    version: String,
    host: String,
}

impl K8sCluster {
    pub fn new(service_name: &str, namespace: &str, port: &str, version: &str, host: &str) -> Self {
        K8sCluster {
            service_name: service_name.to_string(),
            namespace: namespace.to_string(),
            port: port.to_string(),
            version: version.to_string(),
            host: host.to_string(),
        }
    }
}

impl Cluster for K8sCluster {
    fn cluster_name(&self) -> String {
        format!(
            "outbound|{}|{}|{}.{}.svc.cluster.local",
            self.port,
            self.version,
            self.service_name,
            if self.namespace.is_empty() {
                "default"
            } else {
                &self.namespace
            }
        )
    }

    fn host_name(&self) -> String {
        if self.host.is_empty() {
            format!("{}.{}.svc.cluster.local", self.service_name, self.namespace)
        } else {
            self.host.clone()
        }
    }
}

#[derive(Debug, Clone)]
pub struct NacosCluster {
    service_name: String,
    group: String,
    namespace_id: String,
    port: u16,
    is_ext_registry: bool,
    version: String,
    host: String,
}

impl NacosCluster {
    pub fn new(
        service_name: &str,
        group: &str,
        namespace_id: &str,
        port: u16,
        is_ext_registry: bool,
        version: &str,
        host: &str,
    ) -> Self {
        NacosCluster {
            service_name: service_name.to_string(),
            group: group.to_string(),
            namespace_id: namespace_id.to_string(),
            port,
            is_ext_registry,
            version: version.to_string(),
            host: host.to_string(),
        }
    }
}
impl Cluster for NacosCluster {
    fn cluster_name(&self) -> String {
        let group = if self.group.is_empty() {
            "DEFAULT-GROUP".to_string()
        } else {
            self.group.replace('_', "-")
        };
        let tail = if self.is_ext_registry {
            "nacos-ext"
        } else {
            "nacos"
        };
        format!(
            "outbound|{}|{}|{}.{}.{}.{}",
            self.port, self.version, self.service_name, group, self.namespace_id, tail
        )
    }

    fn host_name(&self) -> String {
        if self.host.is_empty() {
            self.service_name.clone()
        } else {
            self.host.clone()
        }
    }
}

#[derive(Debug, Clone)]
pub struct StaticIpCluster {
    service_name: String,
    port: u16,
    host: String,
}

impl StaticIpCluster {
    pub fn new(service_name: &str, port: u16, host: &str) -> Self {
        StaticIpCluster {
            service_name: service_name.to_string(),
            port,
            host: host.to_string(),
        }
    }
}
impl Cluster for StaticIpCluster {
    fn cluster_name(&self) -> String {
        format!("outbound|{}||{}.static", self.port, self.service_name)
    }

    fn host_name(&self) -> String {
        if self.host.is_empty() {
            self.service_name.clone()
        } else {
            self.host.clone()
        }
    }
}

#[derive(Debug, Clone)]
pub struct DnsCluster {
    service_name: String,
    domain: String,
    port: u16,
}

impl DnsCluster {
    pub fn new(service_name: &str, domain: &str, port: u16) -> Self {
        DnsCluster {
            service_name: service_name.to_string(),
            domain: domain.to_string(),
            port,
        }
    }
}
impl Cluster for DnsCluster {
    fn cluster_name(&self) -> String {
        format!("outbound|{}||{}.dns", self.port, self.service_name)
    }

    fn host_name(&self) -> String {
        self.domain.clone()
    }
}

#[derive(Debug, Clone)]
pub struct ConsulCluster {
    service_name: String,
    datacenter: String,
    port: u16,
    host: String,
}

impl ConsulCluster {
    pub fn new(service_name: &str, datacenter: &str, port: u16, host: &str) -> Self {
        ConsulCluster {
            service_name: service_name.to_string(),
            datacenter: datacenter.to_string(),
            port,
            host: host.to_string(),
        }
    }
}
impl Cluster for ConsulCluster {
    fn cluster_name(&self) -> String {
        format!(
            "outbound|{}||{}.{}.consul",
            self.port, self.service_name, self.datacenter
        )
    }

    fn host_name(&self) -> String {
        if self.host.is_empty() {
            self.service_name.clone()
        } else {
            self.host.clone()
        }
    }
}

#[derive(Debug, Clone)]
pub struct FQDNCluster {
    fqdn: String,
    host: String,
    port: u16,
}

impl FQDNCluster {
    pub fn new(fqdn: &str, host: &str, port: u16) -> Self {
        FQDNCluster {
            fqdn: fqdn.to_string(),
            host: host.to_string(),
            port,
        }
    }
}
impl Cluster for FQDNCluster {
    fn cluster_name(&self) -> String {
        format!("outbound|{}||{}", self.port, self.fqdn)
    }
    fn host_name(&self) -> String {
        if self.host.is_empty() {
            self.fqdn.clone()
        } else {
            self.host.clone()
        }
    }
}
