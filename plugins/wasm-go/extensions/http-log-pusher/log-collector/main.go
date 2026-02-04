package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 1. 定义与 Wasm 插件发送格式一致的结构体（完整 27 字段，对齐 log-format.json）
type LogEntry struct {
	// 基础请求信息
	StartTime     string `json:"start_time"`               // 请求开始时间 (RFC3339)
	Authority     string `json:"authority"`                // Host/Authority
	TraceID       string `json:"trace_id"`                 // X-B3-TraceID
	Method        string `json:"method"`                   // HTTP 方法
	Path          string `json:"path"`                     // 请求路径
	Protocol      string `json:"protocol"`                 // HTTP 协议版本
	RequestID     string `json:"request_id"`               // X-Request-ID
	UserAgent     string `json:"user_agent"`               // User-Agent
	XForwardedFor string `json:"x_forwarded_for"`          // X-Forwarded-For
	
	// 响应信息
	ResponseCode        int    `json:"response_code"`                  // 响应状态码
	ResponseFlags       string `json:"response_flags"`                 // Envoy 响应标志
	ResponseCodeDetails string `json:"response_code_details"`          // 响应码详情
	
	// 流量信息
	BytesReceived int64 `json:"bytes_received"` // 接收字节数
	BytesSent     int64 `json:"bytes_sent"`     // 发送字节数
	Duration      int64 `json:"duration"`       // 请求总耗时(ms)
	
	// 上游信息
	UpstreamCluster                  string `json:"upstream_cluster"`                        // 上游集群名
	UpstreamHost                     string `json:"upstream_host"`                           // 上游主机
	UpstreamServiceTime              string `json:"upstream_service_time"`                   // 上游服务耗时
	UpstreamTransportFailureReason   string `json:"upstream_transport_failure_reason"`       // 上游传输失败原因
	UpstreamLocalAddress             string `json:"upstream_local_address"`                  // 上游本地地址
	
	// 连接信息
	DownstreamLocalAddress  string `json:"downstream_local_address"`  // 下游本地地址
	DownstreamRemoteAddress string `json:"downstream_remote_address"` // 下游远程地址
	
	// 路由信息
	RouteName           string `json:"route_name"`            // 路由名称
	RequestedServerName string `json:"requested_server_name"` // SNI
	
	// Istio 相关
	IstioPolicyStatus string `json:"istio_policy_status"` // Istio 策略状态
	
	// AI 日志
	AILog string `json:"ai_log"` // WASM AI 日志 (JSON 字符串)
}

// 全局变量
var (
	db         *sql.DB
	logBuffer  []LogEntry
	bufferLock sync.Mutex
	flushSize  = 50 // 批量写入阈值
)

// 查询响应结构体
type QueryResponse struct {
	Total  int64       `json:"total"`
	Logs   []LogEntry  `json:"logs"`
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
}

func main() {
	// 2. 初始化数据库连接
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		// 默认值，方便本地测试
		dsn = "root:root@tcp(127.0.0.1:3306)/higress_poc?charset=utf8mb4&parseTime=True"
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	// 限制连接池，模拟资源受限
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		log.Printf("Error: Database connection failed: %v", err)
		log.Fatalf("Failed to connect to database: %v", err)
	} else {
		log.Println("Database connected successfully")
	}

	// 3. 启动后台 Flush 协程
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			flushLogs()
		}
	}()

	// 4. 启动 HTTP Server
	http.HandleFunc("/ingest", handleIngest)
	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	port := "8080"
	log.Printf("Tiny Log Collector listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// 接收 Wasm 发来的日志
func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var entry LogEntry
	// 简单粗暴的 JSON 解析
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		log.Printf("[Ingest] Error decoding JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	log.Printf("[Ingest] Received log: path=%s, method=%s, status=%d, trace_id=%s",
		entry.Path, entry.Method, entry.ResponseCode, entry.TraceID)

	// 加锁写入内存 Buffer
	bufferLock.Lock()
	logBuffer = append(logBuffer, entry)
	currentLen := len(logBuffer)
	bufferLock.Unlock()
	
	log.Printf("[Ingest] Buffer size: %d/%d", currentLen, flushSize)

	// 达到阈值主动触发 Flush (非阻塞)
	if currentLen >= flushSize {
		log.Printf("[Ingest] Buffer full, triggering flush")
		go flushLogs()
	}

	w.WriteHeader(http.StatusOK)
}

// 批量写入 MySQL
func flushLogs() {
	bufferLock.Lock()
	if len(logBuffer) == 0 {
		bufferLock.Unlock()
		return
	}
	// 交换 Buffer
	chunk := logBuffer
	logBuffer = make([]LogEntry, 0, flushSize)
	bufferLock.Unlock()

	// 拼凑 SQL 语句
	if len(chunk) == 0 {
		return
	}

	log.Printf("[FlushLogs] Preparing to flush %d log entries", len(chunk))

	// 警告:这里的代码是为了 POC 写的,简单粗暴。
	// 生产环境应该使用 sqlx 或者 GORM 的 Batch Insert。
	valueStrings := []string{}
	valueArgs := []interface{}{}

	for idx, entry := range chunk {
		// 26 个字段的占位符 (对齐 log-format.json)
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		// 转换 RFC3339 时间为 MySQL datetime 格式
		startTime := entry.StartTime
		if t, err := time.Parse(time.RFC3339, entry.StartTime); err == nil {
			startTime = t.Format("2006-01-02 15:04:05")
		} else {
			log.Printf("[FlushLogs] Entry %d: Failed to parse start_time '%s': %v", idx, entry.StartTime, err)
		}

		// 按表结构顺序:26 个字段完整映射 (对齐 log-format.json)
		valueArgs = append(valueArgs,
			// 基础请求信息 (9字段)
			startTime,                  // start_time
			entry.TraceID,              // trace_id
			entry.Authority,            // authority
			entry.Method,               // method
			entry.Path,                 // path
			entry.Protocol,             // protocol
			entry.RequestID,            // request_id
			entry.UserAgent,            // user_agent
			entry.XForwardedFor,        // x_forwarded_for
			// 响应信息 (3字段)
			entry.ResponseCode,         // response_code
			entry.ResponseFlags,        // response_flags
			entry.ResponseCodeDetails,  // response_code_details
			// 流量信息 (3字段)
			entry.BytesReceived,        // bytes_received
			entry.BytesSent,            // bytes_sent
			entry.Duration,             // duration
			// 上游信息 (5字段)
			entry.UpstreamCluster,                  // upstream_cluster
			entry.UpstreamHost,                     // upstream_host
			entry.UpstreamServiceTime,              // upstream_service_time
			entry.UpstreamTransportFailureReason,   // upstream_transport_failure_reason
			entry.UpstreamLocalAddress,             // upstream_local_address
			// 连接信息 (2字段)
			entry.DownstreamLocalAddress,   // downstream_local_address
			entry.DownstreamRemoteAddress,  // downstream_remote_address
			// 路由信息 (2字段)
			entry.RouteName,            // route_name
			entry.RequestedServerName,  // requested_server_name
			// Istio + AI (2字段)
			entry.IstioPolicyStatus,    // istio_policy_status
			entry.AILog,                // ai_log
		)
		// 总计: 9+3+3+5+2+2+2 = 26 字段
		
		log.Printf("[FlushLogs] Entry %d: path=%s, method=%s, status=%d, authority=%s", 
			idx, entry.Path, entry.Method, entry.ResponseCode, entry.Authority)
	}

	// 统计实际参数数量
	expectedParamsPerRow := 26  // 对齐 log-format.json 的 26 个字段
	actualParamsTotal := len(valueArgs)
	actualRowCount := len(chunk)
	expectedParamsTotal := expectedParamsPerRow * actualRowCount
	
	log.Printf("[FlushLogs] SQL Stats: rows=%d, expected_params_total=%d, actual_params_total=%d, params_per_row=%d",
		actualRowCount, expectedParamsTotal, actualParamsTotal, actualParamsTotal/actualRowCount)
	
	if actualParamsTotal != expectedParamsTotal {
		log.Printf("[FlushLogs] ERROR: Parameter count mismatch! Expected %d but got %d", 
			expectedParamsTotal, actualParamsTotal)
	}

	// 构建 INSERT 语句 (26个字段,对齐 log-format.json)
	stmt := fmt.Sprintf(`INSERT INTO access_logs (
		start_time, trace_id, authority, method, path, protocol, request_id, user_agent, x_forwarded_for,
		response_code, response_flags, response_code_details,
		bytes_received, bytes_sent, duration,
		upstream_cluster, upstream_host, upstream_service_time, upstream_transport_failure_reason, upstream_local_address,
		downstream_local_address, downstream_remote_address,
		route_name, requested_server_name,
		istio_policy_status,
		ai_log
	) VALUES %s`, strings.Join(valueStrings, ","))
	
	log.Printf("[FlushLogs] SQL Column Count: 26 (对齐 log-format.json)")
	log.Printf("[FlushLogs] SQL Preview (first 500 chars): %s", stmt[:min(500, len(stmt))])

	// 执行写入
	start := time.Now()
	_, err := db.Exec(stmt, valueArgs...)
	if err != nil {
		// 这里体现了 POC 方案的脆弱性:如果 DB 挂了,这一批日志就直接丢了
		log.Printf("[FlushLogs] Failed to insert batch logs: %v", err)
		log.Printf("[FlushLogs] Failed SQL: %s", stmt)
		log.Printf("[FlushLogs] First row args (%d params): %v", min(26, len(valueArgs)), valueArgs[:min(26, len(valueArgs))])
	} else {
		log.Printf("[FlushLogs] SUCCESS: Flushed %d logs to MySQL in %v", len(chunk), time.Since(start))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 处理日志查询请求
func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 解析查询参数
	params := r.URL.Query()
	
	// 构建查询条件
	whereClause := []string{}
	args := []interface{}{}
	
	// 时间范围查询
	if start := params.Get("start"); start != "" {
		whereClause = append(whereClause, "start_time >= ?")
		args = append(args, start)
	}
	if end := params.Get("end"); end != "" {
		whereClause = append(whereClause, "start_time <= ?")
		args = append(args, end)
	}
	
	// 服务名查询
	if service := params.Get("service"); service != "" {
		whereClause = append(whereClause, "authority = ?")
		args = append(args, service)
	}
	
	// HTTP 方法查询
	if method := params.Get("method"); method != "" {
		whereClause = append(whereClause, "method = ?")
		args = append(args, method)
	}
	
	// 路径查询
	if path := params.Get("path"); path != "" {
		whereClause = append(whereClause, "path LIKE ?")
		args = append(args, "%"+path+"%")
	}
	
	// 状态码查询
	if status := params.Get("status"); status != "" {
		whereClause = append(whereClause, "response_code = ?")
		args = append(args, status)
	}
	
	// TraceID 查询
	if traceID := params.Get("trace_id"); traceID != "" {
		whereClause = append(whereClause, "trace_id = ?")
		args = append(args, traceID)
	}
	
	// 构建完整的 WHERE 子句
	whereSQL := ""
	if len(whereClause) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClause, " AND ")
	}
	
	// 计算总记录数
	countSQL := "SELECT COUNT(*) FROM access_logs " + whereSQL
	var total int64
	err := db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		log.Printf("Error counting logs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to count logs",
		})
		return
	}
	
	// 分页参数
	page := 1
	pageSize := 10
	if p := params.Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	if ps := params.Get("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
		if pageSize < 1 {
			pageSize = 10
		} else if pageSize > 100 {
			pageSize = 100 // 限制最大页面大小
		}
	}
	offset := (page - 1) * pageSize
	
	// 排序参数（必须使用数据库真实字段名）
	sortBy := "start_time"
	sortOrder := "DESC"
	if sb := params.Get("sort_by"); sb != "" {
		// 允许的排序字段白名单
		allowedFields := map[string]bool{
			"start_time":          true,
			"response_code":       true,
			"duration":            true,
			"authority":           true,
			"method":              true,
			"path":                true,
			"bytes_received":      true,
			"bytes_sent":          true,
			"upstream_cluster":    true,
			"route_name":          true,
		}
		if allowedFields[sb] {
			sortBy = sb
		}
	}
	if so := params.Get("sort_order"); so != "" {
		if so == "ASC" || so == "asc" {
			sortOrder = "ASC"
		}
	}
	
	// 构建查询 SQL（查询所有 27 个字段）
	querySQL := fmt.Sprintf(`
		SELECT start_time, trace_id, authority, method, path, protocol, request_id, user_agent, x_forwarded_for,
		       response_code, response_flags, response_code_details,
		       bytes_received, bytes_sent, duration,
		       upstream_cluster, upstream_host, upstream_service_time, upstream_transport_failure_reason, upstream_local_address,
		       downstream_local_address, downstream_remote_address,
		       route_name, requested_server_name,
		       istio_policy_status,
		       ai_log
		FROM access_logs %s ORDER BY %s %s LIMIT ? OFFSET ?`,
		whereSQL, sortBy, sortOrder,
	)
	
	// 添加分页参数
	args = append(args, pageSize, offset)
	
	// 执行查询
	rows, err := db.Query(querySQL, args...)
	if err != nil {
		log.Printf("Error querying logs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to query logs",
		})
		return
	}
	defer rows.Close()
	
	// 解析查询结果（读取所有 27 个字段）
	logs := []LogEntry{}
	for rows.Next() {
		var entry LogEntry
		var startTime time.Time

		err := rows.Scan(
			// 基础请求信息
			&startTime, &entry.TraceID, &entry.Authority, &entry.Method, &entry.Path,
			&entry.Protocol, &entry.RequestID, &entry.UserAgent, &entry.XForwardedFor,
			// 响应信息
			&entry.ResponseCode, &entry.ResponseFlags, &entry.ResponseCodeDetails,
			// 流量信息
			&entry.BytesReceived, &entry.BytesSent, &entry.Duration,
			// 上游信息
			&entry.UpstreamCluster, &entry.UpstreamHost, &entry.UpstreamServiceTime,
			&entry.UpstreamTransportFailureReason, &entry.UpstreamLocalAddress,
			// 连接信息
			&entry.DownstreamLocalAddress, &entry.DownstreamRemoteAddress,
			// 路由信息
			&entry.RouteName, &entry.RequestedServerName,
			// Istio 相关
			&entry.IstioPolicyStatus,
			// AI 日志
			&entry.AILog,
		)
		if err != nil {
			log.Printf("Error scanning log entry: %v", err)
			continue
		}

		entry.StartTime = startTime.Format(time.RFC3339)
		logs = append(logs, entry)
	}
	
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to iterate log entries",
		})
		return
	}
	
	// 返回查询结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryResponse{
		Total:  total,
		Logs:   logs,
		Status: "success",
	})
}