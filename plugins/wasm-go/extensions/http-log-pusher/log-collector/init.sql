CREATE TABLE IF NOT EXISTS access_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    trace_id VARCHAR(64),
    service VARCHAR(128),
    method VARCHAR(16),
    path VARCHAR(1024),
    status INT,
    latency_ms INT,
    details JSON COMMENT '存储Header和其他元数据'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 只加一个时间索引，方便演示按时间排序，但不加全文索引
CREATE INDEX idx_ts ON access_logs(ts);