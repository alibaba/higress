CREATE TABLE IF NOT EXISTS access_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    start_time TIMESTAMP NULL,
    trace_id VARCHAR(64),
    authority VARCHAR(128),
    method VARCHAR(16),
    path VARCHAR(1024),
    response_code INT,
    duration INT,
    ai_log TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 只加一个时间索引，方便演示按时间排序，但不加全文索引
CREATE INDEX idx_ts ON access_logs(ts);