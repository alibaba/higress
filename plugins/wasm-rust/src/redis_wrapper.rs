use std::{collections::HashMap, time::Duration};

use proxy_wasm::{hostcalls::RedisCallbackFn, types::Status};
use redis::{Cmd, ToRedisArgs, Value};

use crate::{cluster_wrapper::Cluster, internal};

pub type RedisValueCallbackFn = dyn FnOnce(&Result<Value, String>, usize, u32);

fn gen_callback(call_fn: Box<RedisValueCallbackFn>) -> Box<RedisCallbackFn> {
    Box::new(move |token_id, status, response_size| {
        let res = match internal::get_redis_call_response(0, response_size) {
            Some(data) => match redis::parse_redis_value(&data) {
                Ok(v) => Ok(v),
                Err(e) => Err(e.to_string()),
            },
            None => Err("response data not found".to_string()),
        };
        call_fn(&res, status, token_id);
    })
}

pub struct RedisClientBuilder {
    upstream: String,
    username: Option<String>,
    password: Option<String>,
    timeout: Duration,
}

impl RedisClientBuilder {
    pub fn new(cluster: &dyn Cluster, timeout: Duration) -> Self {
        RedisClientBuilder {
            upstream: cluster.cluster_name(),
            username: None,
            password: None,
            timeout,
        }
    }

    pub fn username<T: AsRef<str>>(mut self, username: Option<T>) -> Self {
        self.username = username.map(|u| u.as_ref().to_string());
        self
    }

    pub fn password<T: AsRef<str>>(mut self, password: Option<T>) -> Self {
        self.password = password.map(|p| p.as_ref().to_string());
        self
    }

    pub fn build(self) -> RedisClient {
        RedisClient {
            upstream: self.upstream,
            username: self.username,
            password: self.password,
            timeout: self.timeout,
        }
    }
}

pub struct RedisClientConfig {
    upstream: String,
    username: Option<String>,
    password: Option<String>,
    timeout: Duration,
}

impl RedisClientConfig {
    pub fn new(cluster: &dyn Cluster, timeout: Duration) -> Self {
        RedisClientConfig {
            upstream: cluster.cluster_name(),
            username: None,
            password: None,
            timeout,
        }
    }

    pub fn username<T: AsRef<str>>(&mut self, username: Option<T>) -> &Self {
        self.username = username.map(|u| u.as_ref().to_string());
        self
    }

    pub fn password<T: AsRef<str>>(&mut self, password: Option<T>) -> &Self {
        self.password = password.map(|p| p.as_ref().to_string());
        self
    }
}

#[derive(Debug, Clone)]
pub struct RedisClient {
    upstream: String,
    username: Option<String>,
    password: Option<String>,
    timeout: Duration,
}

impl RedisClient {
    pub fn new(config: &RedisClientConfig) -> Self {
        RedisClient {
            upstream: config.upstream.clone(),
            username: config.username.clone(),
            password: config.password.clone(),
            timeout: config.timeout,
        }
    }

    pub fn init(&self) -> Result<(), Status> {
        internal::redis_init(
            &self.upstream,
            self.username.as_ref().map(|u| u.as_bytes()),
            self.password.as_ref().map(|p| p.as_bytes()),
            self.timeout,
        )
    }

    fn call(&self, query: &[u8], call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        internal::dispatch_redis_call(&self.upstream, query, gen_callback(call_fn))
    }

    pub fn command(&self, cmd: &Cmd, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        self.call(&cmd.get_packed_command(), call_fn)
    }

    pub fn eval<T: ToRedisArgs>(
        &self,
        script: &str,
        numkeys: i32,
        keys: Vec<&str>,
        args: Vec<T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("eval");
        cmd.arg(script).arg(numkeys);
        for key in keys {
            cmd.arg(key);
        }
        for arg in args {
            cmd.arg(arg);
        }
        self.command(&cmd, call_fn)
    }

    // Key
    pub fn del(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("del");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn exists(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("exists");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn expire(
        &self,
        key: &str,
        ttl: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("expire");
        cmd.arg(key).arg(ttl);
        self.command(&cmd, call_fn)
    }

    pub fn persist(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("persist");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    // String
    pub fn get(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("get");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn set<T: ToRedisArgs>(
        &self,
        key: &str,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("set");
        cmd.arg(key).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn setex<T: ToRedisArgs>(
        &self,
        key: &str,
        value: T,
        ttl: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("setex");
        cmd.arg(key).arg(ttl).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn mget(&self, keys: Vec<&str>, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("mget");
        for key in keys {
            cmd.arg(key);
        }
        self.command(&cmd, call_fn)
    }

    pub fn mset<T: ToRedisArgs>(
        &self,
        kv_map: HashMap<&str, T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("mset");
        for (k, v) in kv_map {
            cmd.arg(k).arg(v);
        }
        self.command(&cmd, call_fn)
    }

    pub fn incr(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("incr");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn decr(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("decr");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn incrby(
        &self,
        key: &str,
        delta: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("incrby");
        cmd.arg(key).arg(delta);
        self.command(&cmd, call_fn)
    }

    pub fn decrby(
        &self,
        key: &str,
        delta: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("decrby");
        cmd.arg(key).arg(delta);
        self.command(&cmd, call_fn)
    }

    // List
    pub fn llen(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("llen");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn rpush<T: ToRedisArgs>(
        &self,
        key: &str,
        vals: Vec<T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("rpush");
        cmd.arg(key);
        for val in vals {
            cmd.arg(val);
        }
        self.command(&cmd, call_fn)
    }

    pub fn rpop(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("rpop");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn lpush<T: ToRedisArgs>(
        &self,
        key: &str,
        vals: Vec<T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("lpush");
        cmd.arg(key);
        for val in vals {
            cmd.arg(val);
        }
        self.command(&cmd, call_fn)
    }

    pub fn lpop(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("lpop");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn lindex(
        &self,
        key: &str,
        index: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("lindex");
        cmd.arg(key).arg(index);
        self.command(&cmd, call_fn)
    }

    pub fn lrange(
        &self,
        key: &str,
        start: i32,
        stop: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("lrange");
        cmd.arg(key).arg(start).arg(stop);
        self.command(&cmd, call_fn)
    }

    pub fn lrem<T: ToRedisArgs>(
        &self,
        key: &str,
        count: i32,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("lrem");
        cmd.arg(key).arg(count).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn linsert_before<T: ToRedisArgs>(
        &self,
        key: &str,
        pivot: T,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("linsert");
        cmd.arg(key).arg("before").arg(pivot).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn linsert_after<T: ToRedisArgs>(
        &self,
        key: &str,
        pivot: T,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("linsert");
        cmd.arg(key).arg("after").arg(pivot).arg(value);

        self.command(&cmd, call_fn)
    }

    // Hash
    pub fn hexists(
        &self,
        key: &str,
        field: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hexists");
        cmd.arg(key).arg(field);
        self.command(&cmd, call_fn)
    }

    pub fn hdel(
        &self,
        key: &str,
        fields: Vec<&str>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hdel");
        cmd.arg(key);
        for field in fields {
            cmd.arg(field);
        }
        self.command(&cmd, call_fn)
    }

    pub fn hlen(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hlen");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn hget(
        &self,
        key: &str,
        field: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hget");
        cmd.arg(key).arg(field);
        self.command(&cmd, call_fn)
    }

    pub fn hset<T: ToRedisArgs>(
        &self,
        key: &str,
        field: &str,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hset");
        cmd.arg(key).arg(field).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn hmget(
        &self,
        key: &str,
        fields: Vec<&str>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hmget");
        cmd.arg(key);
        for field in fields {
            cmd.arg(field);
        }
        self.command(&cmd, call_fn)
    }

    pub fn hmset<T: ToRedisArgs>(
        &self,
        key: &str,
        kv_map: HashMap<&str, T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hmset");
        cmd.arg(key);
        for (k, v) in kv_map {
            cmd.arg(k).arg(v);
        }
        self.command(&cmd, call_fn)
    }

    pub fn hkeys(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hkeys");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn hvals(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hvals");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn hgetall(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hgetall");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn hincrby(
        &self,
        key: &str,
        field: &str,
        delta: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hincrby");
        cmd.arg(key).arg(field).arg(delta);
        self.command(&cmd, call_fn)
    }

    pub fn hincrbyfloat(
        &self,
        key: &str,
        field: &str,
        delta: f64,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("hincrbyfloat");
        cmd.arg(key).arg(field).arg(delta);
        self.command(&cmd, call_fn)
    }

    // Set
    pub fn scard(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("scard");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn sadd<T: ToRedisArgs>(
        &self,
        key: &str,
        vals: Vec<T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sadd");
        cmd.arg(key);
        for val in vals {
            cmd.arg(val);
        }
        self.command(&cmd, call_fn)
    }

    pub fn srem<T: ToRedisArgs>(
        &self,
        key: &str,
        vals: Vec<T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("srem");
        cmd.arg(key);
        for val in vals {
            cmd.arg(val);
        }
        self.command(&cmd, call_fn)
    }

    pub fn sismember<T: ToRedisArgs>(
        &self,
        key: &str,
        value: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sismember");
        cmd.arg(key).arg(value);
        self.command(&cmd, call_fn)
    }

    pub fn smembers(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("smembers");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn sdiff(
        &self,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sdiff");
        cmd.arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    pub fn sdiffstore(
        &self,
        destination: &str,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sdiffstore");
        cmd.arg(destination).arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    pub fn sinter(
        &self,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sinter");
        cmd.arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    pub fn sinterstore(
        &self,
        destination: &str,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sinterstore");
        cmd.arg(destination).arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    pub fn sunion(
        &self,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sunion");
        cmd.arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    pub fn sunion_store(
        &self,
        destination: &str,
        key1: &str,
        key2: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("sunionstore");
        cmd.arg(destination).arg(key1).arg(key2);
        self.command(&cmd, call_fn)
    }

    // Sorted Set
    pub fn zcard(&self, key: &str, call_fn: Box<RedisValueCallbackFn>) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zcard");
        cmd.arg(key);
        self.command(&cmd, call_fn)
    }

    pub fn zadd<T: ToRedisArgs>(
        &self,
        key: &str,
        ms_map: HashMap<&str, T>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zadd");
        cmd.arg(key);
        for (m, s) in ms_map {
            cmd.arg(s).arg(m);
        }
        self.command(&cmd, call_fn)
    }

    pub fn zcount<T: ToRedisArgs>(
        &self,
        key: &str,
        min: T,
        max: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zcount");
        cmd.arg(key).arg(min).arg(max);
        self.command(&cmd, call_fn)
    }

    pub fn zincrby<T: ToRedisArgs>(
        &self,
        key: &str,
        member: &str,
        delta: T,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zincrby");
        cmd.arg(key).arg(delta).arg(member);
        self.command(&cmd, call_fn)
    }

    pub fn zscore(
        &self,
        key: &str,
        member: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zscore");
        cmd.arg(key).arg(member);
        self.command(&cmd, call_fn)
    }

    pub fn zrank(
        &self,
        key: &str,
        member: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zrank");
        cmd.arg(key).arg(member);
        self.command(&cmd, call_fn)
    }

    pub fn zrev_rank(
        &self,
        key: &str,
        member: &str,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zrevrank");
        cmd.arg(key).arg(member);
        self.command(&cmd, call_fn)
    }

    pub fn zrem(
        &self,
        key: &str,
        members: Vec<&str>,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zrem");
        cmd.arg(key);
        for member in members {
            cmd.arg(member);
        }
        self.command(&cmd, call_fn)
    }

    pub fn zrange(
        &self,
        key: &str,
        start: i32,
        stop: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zrange");
        cmd.arg(key).arg(start).arg(stop);
        self.command(&cmd, call_fn)
    }

    pub fn zrevrange(
        &self,
        key: &str,
        start: i32,
        stop: i32,
        call_fn: Box<RedisValueCallbackFn>,
    ) -> Result<u32, Status> {
        let mut cmd = redis::cmd("zrevrange");
        cmd.arg(key).arg(start).arg(stop);
        self.command(&cmd, call_fn)
    }
}
