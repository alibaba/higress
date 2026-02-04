-- ================================================================
-- Higress HTTP Log Collector - 数据库初始化脚本
-- ================================================================
-- 功能: 创建 access_logs 表并建立性能优化索引
-- 对齐: log-format.json 定义的 27 个字段
-- ================================================================

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS higress_poc DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE higress_poc;

-- 删除旧表（谨慎使用，生产环境需备份）
DROP TABLE IF EXISTS access_logs;

-- 创建 access_logs 表
CREATE TABLE `access_logs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  
  -- 基础请求信息（9字段）
  `start_time` timestamp NULL DEFAULT NULL COMMENT '请求开始时间',
  `trace_id` varchar(64) NULL DEFAULT NULL COMMENT 'X-B3-TraceID 分布式追踪ID',
  `authority` varchar(128) NULL DEFAULT NULL COMMENT 'Host/Authority 域名',
  `method` varchar(16) NULL DEFAULT NULL COMMENT 'HTTP 方法 (GET/POST等)',
  `path` varchar(1024) NULL DEFAULT NULL COMMENT '请求路径',
  `protocol` varchar(16) NULL DEFAULT NULL COMMENT 'HTTP 协议版本 (HTTP/1.1等)',
  `request_id` varchar(64) NULL DEFAULT NULL COMMENT 'X-Request-ID 请求唯一标识',
  `user_agent` varchar(512) NULL DEFAULT NULL COMMENT 'User-Agent 客户端信息',
  `x_forwarded_for` varchar(256) NULL DEFAULT NULL COMMENT 'X-Forwarded-For 客户端真实IP',
  
  -- 响应信息（3字段）
  `response_code` int NULL DEFAULT NULL COMMENT '响应状态码 (200/404/500等)',
  `response_flags` varchar(64) NULL DEFAULT NULL COMMENT 'Envoy 响应标志',
  `response_code_details` varchar(256) NULL DEFAULT NULL COMMENT '响应码详情',
  
  -- 流量信息（3字段）
  `bytes_received` bigint NULL DEFAULT NULL COMMENT '接收字节数',
  `bytes_sent` bigint NULL DEFAULT NULL COMMENT '发送字节数',
  `duration` int NULL DEFAULT NULL COMMENT '请求总耗时(ms)',
  
  -- 上游信息（5字段）
  `upstream_cluster` varchar(256) NULL DEFAULT NULL COMMENT '上游集群名',
  `upstream_host` varchar(256) NULL DEFAULT NULL COMMENT '上游主机地址',
  `upstream_service_time` varchar(32) NULL DEFAULT NULL COMMENT '上游服务耗时',
  `upstream_transport_failure_reason` varchar(256) NULL DEFAULT NULL COMMENT '上游传输失败原因',
  `upstream_local_address` varchar(64) NULL DEFAULT NULL COMMENT '上游本地地址',
  
  -- 连接信息（2字段）
  `downstream_local_address` varchar(64) NULL DEFAULT NULL COMMENT '下游本地地址',
  `downstream_remote_address` varchar(64) NULL DEFAULT NULL COMMENT '下游远程地址',
  
  -- 路由信息（2字段）
  `route_name` varchar(256) NULL DEFAULT NULL COMMENT '路由名称',
  `requested_server_name` varchar(256) NULL DEFAULT NULL COMMENT 'SNI 服务器名称',
  
  -- Istio + AI（2字段）
  `istio_policy_status` varchar(64) NULL DEFAULT NULL COMMENT 'Istio 策略状态',
  `ai_log` text NULL DEFAULT NULL COMMENT 'WASM AI 日志 (JSON序列化字符串)',
  
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='HTTP 访问日志表（对齐 log-format.json 27字段）';

-- ================================================================
-- 性能优化索引（根据查询场景设计）
-- ================================================================

-- 1. 时间范围查询索引（最常用：按时间范围查询日志）
CREATE INDEX `idx_start_time` ON `access_logs` (`start_time` DESC);

-- 2. 分布式追踪索引（根据 trace_id 查询完整调用链）
CREATE INDEX `idx_trace_id` ON `access_logs` (`trace_id`);

-- 3. 域名+时间复合索引（查询特定域名的访问日志）
CREATE INDEX `idx_authority_time` ON `access_logs` (`authority`, `start_time` DESC);

-- 4. 状态码+时间复合索引（查询错误日志：4xx/5xx）
CREATE INDEX `idx_response_code_time` ON `access_logs` (`response_code`, `start_time` DESC);

-- 5. 请求路径索引（路径模糊搜索，使用前缀索引优化）
CREATE INDEX `idx_path` ON `access_logs` (`path`(255));

-- 6. 方法+域名复合索引（按 HTTP 方法分析流量）
CREATE INDEX `idx_method_authority` ON `access_logs` (`method`, `authority`);

-- 7. 耗时索引（慢查询分析：duration > 1000ms）
CREATE INDEX `idx_duration` ON `access_logs` (`duration` DESC);

-- 8. 上游集群索引（服务级别监控与故障定位）
CREATE INDEX `idx_upstream_cluster` ON `access_logs` (`upstream_cluster`, `start_time` DESC);

-- 9. 路由名称索引（路由级别性能分析）
CREATE INDEX `idx_route_name` ON `access_logs` (`route_name`, `start_time` DESC);

-- ================================================================
-- 初始化完成提示
-- ================================================================
SELECT '✅ access_logs 表创建成功！包含 27 个字段 + 9 个性能索引' AS status;
