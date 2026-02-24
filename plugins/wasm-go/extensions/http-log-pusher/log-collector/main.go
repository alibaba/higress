package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 1. 定义与 Wasm 插件发送格式一致的结构体（完整 37 字段，对齐 log-format.json + 监控元数据 + token字段）
type LogEntry struct {
	// 基础请求信息
	StartTime     string `json:"start_time"`      // 请求开始时间 (RFC3339)
	Authority     string `json:"authority"`       // Host/Authority
	TraceID       string `json:"trace_id"`        // X-B3-TraceID
	Method        string `json:"method"`          // HTTP 方法
	Path          string `json:"path"`            // 请求路径
	Protocol      string `json:"protocol"`        // HTTP 协议版本
	RequestID     string `json:"request_id"`      // X-Request-ID
	UserAgent     string `json:"user_agent"`      // User-Agent
	XForwardedFor string `json:"x_forwarded_for"` // X-Forwarded-For

	// 响应信息
	ResponseCode        int    `json:"response_code"`         // 响应状态码
	ResponseFlags       string `json:"response_flags"`        // Envoy 响应标志
	ResponseCodeDetails string `json:"response_code_details"` // 响应码详情

	// 流量信息
	BytesReceived int64 `json:"bytes_received"` // 接收字节数
	BytesSent     int64 `json:"bytes_sent"`     // 发送字节数
	Duration      int64 `json:"duration"`       // 请求总耗时(ms)

	// 上游信息
	UpstreamCluster                string `json:"upstream_cluster"`                  // 上游集群名
	UpstreamHost                   string `json:"upstream_host"`                     // 上游主机
	UpstreamServiceTime            string `json:"upstream_service_time"`             // 上游服务耗时
	UpstreamTransportFailureReason string `json:"upstream_transport_failure_reason"` // 上游传输失败原因
	UpstreamLocalAddress           string `json:"upstream_local_address"`            // 上游本地地址

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

	// ===== 监控元数据字段 (8个) =====
	InstanceID string `json:"instance_id"` // 实例ID
	API        string `json:"api"`         // API名称
	Model      string `json:"model"`       // 模型名称
	Consumer   string `json:"consumer"`    // 消费者信息
	Route      string `json:"route"`       // 路由名称(冗余字段，便于查询)
	Service    string `json:"service"`     // 服务名称
	MCPServer  string `json:"mcp_server"`  // MCP服务器名称
	MCPTool    string `json:"mcp_tool"`    // MCP工具名称

	// ===== Token使用统计字段 (3个) =====
	InputTokens  int64 `json:"input_tokens"`  // 输入token数量
	OutputTokens int64 `json:"output_tokens"` // 输出token数量
	TotalTokens  int64 `json:"total_tokens"`  // 总token数量
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
	Total  int64      `json:"total"`
	Logs   []LogEntry `json:"logs"`
	Status string     `json:"status"`
	Error  string     `json:"error,omitempty"`
}

// 聚合查询响应结构体
type AggregationResponse struct {
	Status string                 `json:"status"`
	Error  string                 `json:"error,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// KPI数据结构体
type KpiData struct {
	PV            int64 `json:"pv"`
	UV            int64 `json:"uv"`
	BytesReceived int64 `json:"bytes_received"`
	BytesSent     int64 `json:"bytes_sent"`
	InputTokens   int64 `json:"input_tokens"`
	OutputTokens  int64 `json:"output_tokens"`
	TotalTokens   int64 `json:"total_tokens"`
	FallbackCount int64 `json:"fallback_count"`
}

// 时间序列数据结构体
type TimeSeriesData struct {
	Timestamp int64       `json:"timestamp"`
	Values    interface{} `json:"values"`
}

// 业务类型常量
const (
	BizTypeMCPServer = "MCP_SERVER"
	BizTypeModelAPI  = "MODEL_API"
)

// appendBizTypeWhereClause 根据 bizType 追加 WHERE 条件（表无 bizType 列，用 mcp_tool 推断）
// 约定（来自实际埋点约束）：
// - MCP_SERVER：mcp_tool 一定有值（真实 MCP 工具名）
// - MODEL_API：一定没有 mcp_tool（为空或 NULL）
func appendBizTypeWhereClause(whereClause *[]string, args *[]interface{}, bizType string) {
	if bizType == "" {
		return
	}
	switch bizType {
	case BizTypeModelAPI:
		// 模型 API 调用
		*whereClause = append(*whereClause, buildModelNotNullCondition())
	case BizTypeMCPServer:
		// 真实 MCP 调用
		*whereClause = append(*whereClause, buildMCPNotNullCondition())
	default:
		// 未知 bizType 不追加条件
	}
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

	// 3. 启动后台 Flush 协程（定时刷新）
	flushInterval := 1 * time.Second
	log.Printf("[Batch] Starting background flush goroutine, interval=%v, threshold=%d logs", flushInterval, flushSize)
	go func() {
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()
		for range ticker.C {
			bufferLock.Lock()
			bufferSize := len(logBuffer)
			bufferLock.Unlock()
			if bufferSize > 0 {
				log.Printf("[Batch] Trigger flush by timer: buffer=%d", bufferSize)
				flushLogs()
			}
		}
	}()

	// 4. 启动 HTTP Server
	http.HandleFunc("/ingest", handleIngest)
	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/batch/kpi", handleBatchKpi)
	http.HandleFunc("/batch/chart", handleBatchChart)
	http.HandleFunc("/batch/table", handleBatchTable)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // 默认端口
	}
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

	// 加锁写入内存 Buffer
	bufferLock.Lock()
	logBuffer = append(logBuffer, entry)
	currentLen := len(logBuffer)
	bufferLock.Unlock()

	// 达到阈值主动触发 Flush (非阻塞)
	if currentLen >= flushSize {
		log.Printf("[Batch] Trigger flush by count: buffer=%d/%d", currentLen, flushSize)
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

	log.Printf("[Batch] Start flushing %d logs to MySQL", len(chunk))

	// 警告:这里的代码是为了 POC 写的,简单粗暴。
	// 生产环境应该使用 sqlx 或者 GORM 的 Batch Insert。
	valueStrings := []string{}
	valueArgs := []interface{}{}

	for _, entry := range chunk {
		// 37 个字段的占位符 (对齐 log-format.json + 监控元数据 + token字段)
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		// 转换 RFC3339 时间为 MySQL datetime 格式
		startTime := entry.StartTime
		if t, err := time.Parse(time.RFC3339, entry.StartTime); err == nil {
			startTime = t.Format("2006-01-02 15:04:05")
		}

		// 按表结构顺序:37 个字段完整映射
		valueArgs = append(valueArgs,
			// 基础请求信息 (9字段)
			startTime,           // start_time
			entry.TraceID,       // trace_id
			entry.Authority,     // authority
			entry.Method,        // method
			entry.Path,          // path
			entry.Protocol,      // protocol
			entry.RequestID,     // request_id
			entry.UserAgent,     // user_agent
			entry.XForwardedFor, // x_forwarded_for
			// 响应信息 (3字段)
			entry.ResponseCode,        // response_code
			entry.ResponseFlags,       // response_flags
			entry.ResponseCodeDetails, // response_code_details
			// 流量信息 (3字段)
			entry.BytesReceived, // bytes_received
			entry.BytesSent,     // bytes_sent
			entry.Duration,      // duration
			// 上游信息 (5字段)
			entry.UpstreamCluster,                // upstream_cluster
			entry.UpstreamHost,                   // upstream_host
			entry.UpstreamServiceTime,            // upstream_service_time
			entry.UpstreamTransportFailureReason, // upstream_transport_failure_reason
			entry.UpstreamLocalAddress,           // upstream_local_address
			// 连接信息 (2字段)
			entry.DownstreamLocalAddress,  // downstream_local_address
			entry.DownstreamRemoteAddress, // downstream_remote_address
			// 路由信息 (2字段)
			entry.RouteName,           // route_name
			entry.RequestedServerName, // requested_server_name
			// Istio + AI (2字段)
			entry.IstioPolicyStatus, // istio_policy_status
			entry.AILog,             // ai_log
			// ===== 监控元数据 (8字段) =====
			entry.InstanceID, // instance_id
			entry.API,        // api
			entry.Model,      // model
			entry.Consumer,   // consumer
			entry.Route,      // route
			entry.Service,    // service
			entry.MCPServer,  // mcp_server
			entry.MCPTool,    // mcp_tool
			// ===== Token使用统计 (3字段) =====
			entry.InputTokens,  // input_tokens
			entry.OutputTokens, // output_tokens
			entry.TotalTokens,  // total_tokens
		)
		// 总计: 9+3+3+5+2+2+2+8+3 = 37 字段
	}

	// 构建 INSERT 语句 (37个字段,对齐 log-format.json + 监控元数据 + token字段)
	stmt := fmt.Sprintf(`INSERT INTO access_logs (
		start_time, trace_id, authority, method, path, protocol, request_id, user_agent, x_forwarded_for,
		response_code, response_flags, response_code_details,
		bytes_received, bytes_sent, duration,
		upstream_cluster, upstream_host, upstream_service_time, upstream_transport_failure_reason, upstream_local_address,
		downstream_local_address, downstream_remote_address,
		route_name, requested_server_name,
		istio_policy_status,
		ai_log,
		instance_id, api, model, consumer, route, service, mcp_server, mcp_tool,
		input_tokens, output_tokens, total_tokens
	) VALUES %s`, strings.Join(valueStrings, ","))

	// 执行写入
	start := time.Now()
	_, err := db.Exec(stmt, valueArgs...)
	duration := time.Since(start)
	if err != nil {
		// 这里体现了 POC 方案的脆弱性:如果 DB 挂了,这一批日志就直接丢了
		log.Printf("[Batch] ❌ FAILED to flush %d logs (duration=%v): %v", len(chunk), duration, err)
	} else {
		log.Printf("[Batch] ✓ SUCCESS flushed %d logs to MySQL (duration=%v, avg=%v/log)",
			len(chunk), duration, duration/time.Duration(len(chunk)))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 处理日志查询请求
func handleQuery(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 解析查询参数
	params := r.URL.Query()
	log.Printf("[Query] Request received: %s", r.URL.RawQuery)

	// 构建查询条件
	whereClause := []string{}
	args := []interface{}{}
	filters := []string{} // 记录使用的过滤条件

	// 时间范围查询 (支持 start_time 参数)
	if start := params.Get("start_time"); start != "" {
		whereClause = append(whereClause, "start_time >= ?")
		args = append(args, start)
		filters = append(filters, fmt.Sprintf("start_time>=%s", start))
	}
	// 兼容旧参数 start
	if start := params.Get("start"); start != "" {
		whereClause = append(whereClause, "start_time >= ?")
		args = append(args, start)
		filters = append(filters, fmt.Sprintf("start>=%s", start))
	}
	if end := params.Get("end"); end != "" {
		whereClause = append(whereClause, "start_time <= ?")
		args = append(args, end)
		filters = append(filters, fmt.Sprintf("end<=%s", end))
	}

	// authority 查询 (原始字段名)
	if authority := params.Get("authority"); authority != "" {
		whereClause = append(whereClause, "authority = ?")
		args = append(args, authority)
		filters = append(filters, fmt.Sprintf("authority=%s", authority))
	}
	// 兼容旧参数 service
	if service := params.Get("service"); service != "" {
		whereClause = append(whereClause, "authority = ?")
		args = append(args, service)
		filters = append(filters, fmt.Sprintf("service=%s", service))
	}

	// HTTP 方法查询
	if method := params.Get("method"); method != "" {
		whereClause = append(whereClause, "method = ?")
		args = append(args, method)
		filters = append(filters, fmt.Sprintf("method=%s", method))
	}

	// BizType 查询（区分 MCP Server 和 Model API）
	if bizType := params.Get("bizType"); bizType != "" {
		filters = append(filters, fmt.Sprintf("bizType=%s", bizType))
	}

	// 路径查询 (支持精确匹配和模糊匹配)
	if path := params.Get("path"); path != "" {
		if pathLike := params.Get("path_like"); pathLike == "true" {
			// 模糊查询
			whereClause = append(whereClause, "path LIKE ?")
			args = append(args, "%"+path+"%")
			filters = append(filters, fmt.Sprintf("path LIKE %%%s%%", path))
		} else {
			// 默认模糊查询 (兼容原有行为)
			whereClause = append(whereClause, "path LIKE ?")
			args = append(args, "%"+path+"%")
			filters = append(filters, fmt.Sprintf("path LIKE %%%s%%", path))
		}
	}

	// 状态码查询 (原始字段名 response_code)
	if responseCode := params.Get("response_code"); responseCode != "" {
		whereClause = append(whereClause, "response_code = ?")
		args = append(args, responseCode)
		filters = append(filters, fmt.Sprintf("response_code=%s", responseCode))
	}
	// 兼容旧参数 status
	if status := params.Get("status"); status != "" {
		whereClause = append(whereClause, "response_code = ?")
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status=%s", status))
	}

	// TraceID 查询
	if traceID := params.Get("trace_id"); traceID != "" {
		whereClause = append(whereClause, "trace_id = ?")
		args = append(args, traceID)
		filters = append(filters, fmt.Sprintf("trace_id=%s", traceID))
	}

	// ===== 新增监控元数据查询支持 =====
	// 实例ID查询
	if instanceID := params.Get("instance_id"); instanceID != "" {
		whereClause = append(whereClause, "instance_id = ?")
		args = append(args, instanceID)
		filters = append(filters, fmt.Sprintf("instance_id=%s", instanceID))
	}

	// API名称查询
	if api := params.Get("api"); api != "" {
		whereClause = append(whereClause, "api = ?")
		args = append(args, api)
		filters = append(filters, fmt.Sprintf("api=%s", api))
	}

	// 模型名称查询
	if model := params.Get("model"); model != "" {
		whereClause = append(whereClause, "model = ?")
		args = append(args, model)
		filters = append(filters, fmt.Sprintf("model=%s", model))
	}

	// 消费者查询
	if consumer := params.Get("consumer"); consumer != "" {
		whereClause = append(whereClause, "consumer = ?")
		args = append(args, consumer)
		filters = append(filters, fmt.Sprintf("consumer=%s", consumer))
	}

	// 路由查询
	if route := params.Get("route"); route != "" {
		whereClause = append(whereClause, "route = ?")
		args = append(args, route)
		filters = append(filters, fmt.Sprintf("route=%s", route))
	}

	// 服务查询
	if service := params.Get("service"); service != "" {
		whereClause = append(whereClause, "service = ?")
		args = append(args, service)
		filters = append(filters, fmt.Sprintf("service=%s", service))
	}

	// MCP Server查询
	if mcpServer := params.Get("mcp_server"); mcpServer != "" {
		whereClause = append(whereClause, "mcp_server = ?")
		args = append(args, mcpServer)
		filters = append(filters, fmt.Sprintf("mcp_server=%s", mcpServer))
	}

	// MCP Tool查询
	if mcpTool := params.Get("mcp_tool"); mcpTool != "" {
		whereClause = append(whereClause, "mcp_tool = ?")
		args = append(args, mcpTool)
		filters = append(filters, fmt.Sprintf("mcp_tool=%s", mcpTool))
	}

	// 构建完整的 WHERE 子句
	whereSQL := ""
	if len(whereClause) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClause, " AND ")
	}
	log.Printf("[Query] Filters applied: [%s]", strings.Join(filters, ", "))

	// 计算总记录数
	countStart := time.Now()
	countSQL := "SELECT COUNT(*) FROM access_logs " + whereSQL
	var total int64
	err := db.QueryRow(countSQL, args...).Scan(&total)
	countDuration := time.Since(countStart)
	if err != nil {
		log.Printf("[Query] ❌ COUNT failed (duration=%v): %v", countDuration, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to count logs",
		})
		return
	}
	log.Printf("[Query] COUNT result: total=%d (duration=%v)", total, countDuration)

	// 分页参数 (带错误处理)
	page := 1
	pageSize := 10
	if p := params.Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			page = n
		} else {
			log.Printf("[Query] Invalid page parameter: %s, using default: 1", p)
		}
		if page < 1 {
			log.Printf("[Query] Page < 1 (%d), corrected to 1", page)
			page = 1
		}
	}
	if ps := params.Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil {
			pageSize = n
		} else {
			log.Printf("[Query] Invalid page_size parameter: %s, using default: 10", ps)
		}
		if pageSize < 1 {
			log.Printf("[Query] Page_size < 1 (%d), corrected to 10", pageSize)
			pageSize = 10
		} else if pageSize > 100 {
			log.Printf("[Query] Page_size > 100 (%d), limited to 100", pageSize)
			pageSize = 100 // 限制最大页面大小
		}
	}
	offset := (page - 1) * pageSize
	log.Printf("[Query] Pagination: page=%d, page_size=%d, offset=%d", page, pageSize, offset)

	// 排序参数（必须使用数据库真实字段名）
	sortBy := "start_time"
	sortOrder := "DESC"
	if sb := params.Get("sort_by"); sb != "" {
		// 允许的排序字段白名单
		allowedFields := map[string]bool{
			"start_time":       true,
			"response_code":    true,
			"duration":         true,
			"authority":        true,
			"method":           true,
			"path":             true,
			"bytes_received":   true,
			"bytes_sent":       true,
			"upstream_cluster": true,
			"route_name":       true,
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
	log.Printf("[Query] Sorting: sort_by=%s, sort_order=%s", sortBy, sortOrder)

	// 构建查询 SQL（查询所有 37 个字段）
	querySQL := fmt.Sprintf(`
		SELECT start_time, trace_id, authority, method, path, protocol, request_id, user_agent, x_forwarded_for,
		       response_code, response_flags, response_code_details,
		       bytes_received, bytes_sent, duration,
		       upstream_cluster, upstream_host, upstream_service_time, upstream_transport_failure_reason, upstream_local_address,
		       downstream_local_address, downstream_remote_address,
		       route_name, requested_server_name,
		       istio_policy_status,
		       ai_log,
		       instance_id, api, model, consumer, route, service, mcp_server, mcp_tool,
		       input_tokens, output_tokens, total_tokens
		FROM access_logs %s ORDER BY %s %s LIMIT ? OFFSET ?`,
		whereSQL, sortBy, sortOrder,
	)

	// 添加分页参数
	args = append(args, pageSize, offset)

	// 执行查询
	queryExecStart := time.Now()
	rows, err := db.Query(querySQL, args...)
	queryExecDuration := time.Since(queryExecStart)
	if err != nil {
		log.Printf("[Query] ❌ SELECT failed (duration=%v): %v", queryExecDuration, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to query logs",
		})
		return
	}
	defer rows.Close()
	log.Printf("[Query] SELECT executed (duration=%v)", queryExecDuration)

	// 解析查询结果（读取所有 37 个字段）
	parseScanStart := time.Now()
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
			// ===== 监控元数据 (8字段) =====
			&entry.InstanceID, &entry.API, &entry.Model, &entry.Consumer,
			&entry.Route, &entry.Service, &entry.MCPServer, &entry.MCPTool,
			// ===== Token使用统计 (3字段) =====
			&entry.InputTokens, &entry.OutputTokens, &entry.TotalTokens,
		)
		if err != nil {
			log.Printf("[Query] Error scanning row: %v", err)
			continue
		}

		entry.StartTime = startTime.Format(time.RFC3339)
		logs = append(logs, entry)
	}
	parseScanDuration := time.Since(parseScanStart)
	log.Printf("[Query] Rows scanned: count=%d (duration=%v, avg=%v/row)",
		len(logs), parseScanDuration, parseScanDuration/time.Duration(max(1, len(logs))))

	if err = rows.Err(); err != nil {
		log.Printf("[Query] Error iterating rows: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Status: "error",
			Error:  "Failed to iterate log entries",
		})
		return
	}

	totalDuration := time.Since(queryStart)
	log.Printf("[Query] ✓ SUCCESS: returned=%d/%d logs (total_duration=%v, count=%v, query=%v, scan=%v)",
		len(logs), total, totalDuration, countDuration, queryExecDuration, parseScanDuration)

	// 返回查询结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryResponse{
		Total:  total,
		Logs:   logs,
		Status: "success",
	})
}

// 处理批量KPI查询请求
func handleBatchKpi(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[BatchKpi] Request received: %s", r.URL.RawQuery)

	var payloads []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		log.Printf("[BatchKpi] Error decoding payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AggregationResponse{
			Status: "error",
			Error:  "Invalid payload format",
		})
		return
	}

	results := make(map[string]interface{})

	for i, payload := range payloads {
		result, err := processKpiQuery(payload)
		if err != nil {
			log.Printf("[BatchKpi] Error processing payload %d: %v", i, err)
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "success",
				"data":   result,
			}
		}
	}

	totalDuration := time.Since(queryStart)
	log.Printf("[BatchKpi] ✓ SUCCESS: processed %d queries (duration=%v)", len(payloads), totalDuration)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AggregationResponse{
		Status: "success",
		Data:   results,
	})
}

// 处理批量图表查询请求
func handleBatchChart(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[BatchChart] Request received: %s", r.URL.RawQuery)

	// 读取请求体内容用于调试
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[BatchChart] Error reading request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AggregationResponse{
			Status: "error",
			Error:  "Failed to read request body",
		})
		return
	}
	log.Printf("[BatchChart] Request body: %s", string(body))

	// 重新包装body以供后续解码使用
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	var payloads []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		// 兼容单对象：若 body 是单个 query 对象则包装成单元素数组
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		var single map[string]interface{}
		if decErr := json.NewDecoder(r.Body).Decode(&single); decErr != nil {
			log.Printf("[BatchChart] Error decoding payload (array or object): %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(AggregationResponse{
				Status: "error",
				Error:  "Invalid payload format: expected array or single object",
			})
			return
		}
		payloads = []map[string]interface{}{single}
		log.Printf("[BatchChart] Parsed single object as 1 query")
	}

	log.Printf("[BatchChart] Parsed %d queries from payload", len(payloads))
	for i, payload := range payloads {
		log.Printf("[BatchChart] Query %d: %+v", i, payload)
	}

	results := make(map[string]interface{})

	for i, payload := range payloads {
		result, err := processChartQuery(payload)
		if err != nil {
			log.Printf("[BatchChart] Error processing payload %d: %v", i, err)
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "success",
				"data":   result,
			}
		}
	}

	totalDuration := time.Since(queryStart)
	log.Printf("[BatchChart] ✓ SUCCESS: processed %d queries (duration=%v)", len(payloads), totalDuration)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AggregationResponse{
		Status: "success",
		Data:   results,
	})
}

// 处理批量表格查询请求
func handleBatchTable(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[BatchTable] Request received: %s", r.URL.RawQuery)

	var payloads []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		log.Printf("[BatchTable] Error decoding payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AggregationResponse{
			Status: "error",
			Error:  "Invalid payload format",
		})
		return
	}

	results := make(map[string]interface{})

	for i, payload := range payloads {
		result, err := processTableQuery(payload)
		if err != nil {
			log.Printf("[BatchTable] Error processing payload %d: %v", i, err)
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results[fmt.Sprintf("query_%d", i)] = map[string]interface{}{
				"status": "success",
				"data":   result,
			}
		}
	}

	totalDuration := time.Since(queryStart)
	log.Printf("[BatchTable] ✓ SUCCESS: processed %d queries (duration=%v)", len(payloads), totalDuration)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AggregationResponse{
		Status: "success",
		Data:   results,
	})
}

// 处理KPI查询的核心逻辑
func processKpiQuery(payload map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	log.Printf("[KpiQuery] Processing KPI query with payload: %+v", payload)

	// 解析时间范围
	timeRange, ok := payload["timeRange"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid timeRange format")
	}

	startTime, endTime, err := parseTimeRange(timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %v", err)
	}

	// 解析业务类型
	bizType, _ := payload["bizType"].(string)
	if bizType == "" {
		bizType = BizTypeMCPServer // 默认为MCP Server
	}

	// 构建基础查询条件
	whereClause := []string{"start_time >= ?", "start_time <= ?"}
	args := []interface{}{startTime, endTime}

	// 添加过滤条件
	if filters, ok := payload["filters"].(map[string]interface{}); ok {
		for key, value := range filters {
			if strVal, ok := value.(string); ok && strVal != "" {
				whereClause = append(whereClause, fmt.Sprintf("%s = ?", key))
				args = append(args, strVal)
			}
		}
	}
	appendBizTypeWhereClause(&whereClause, &args, bizType)
	whereSQL := "WHERE " + strings.Join(whereClause, " AND ")

	var result map[string]interface{}

	switch bizType {
	case BizTypeModelAPI:
		result, err = queryModelAPIKpi(whereSQL, args)
	case BizTypeMCPServer:
		result, err = queryMCPServerKpi(whereSQL, args)
	default:
		return nil, fmt.Errorf("unsupported bizType: %s", bizType)
	}

	if err != nil {
		return nil, err
	}

	duration := time.Since(start)
	log.Printf("[KpiQuery] ✓ SUCCESS: bizType=%s, duration=%v", bizType, duration)

	return result, nil
}

// 查询Model API的KPI数据
func queryModelAPIKpi(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	// PV 查询
	pvSQL := fmt.Sprintf("SELECT COUNT(*) FROM access_logs %s", whereSQL)
	var pv int64
	if err := db.QueryRow(pvSQL, args...).Scan(&pv); err != nil {
		return nil, fmt.Errorf("failed to query PV: %v", err)
	}

	// UV 查询（基于trace_id去重）
	uvSQL := fmt.Sprintf("SELECT COUNT(DISTINCT trace_id) FROM access_logs %s", whereSQL)
	var uv int64
	if err := db.QueryRow(uvSQL, args...).Scan(&uv); err != nil {
		return nil, fmt.Errorf("failed to query UV: %v", err)
	}

	// Token统计查询
	tokenSQL := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM access_logs %s`, whereSQL)

	var inputTokens, outputTokens, totalTokens int64
	if err := db.QueryRow(tokenSQL, args...).Scan(&inputTokens, &outputTokens, &totalTokens); err != nil {
		return nil, fmt.Errorf("failed to query token stats: %v", err)
	}

	// Fallback请求数查询
	fallbackSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM access_logs %s 
		AND response_code IN ('503', '429')`, whereSQL)
	var fallbackCount int64
	if err := db.QueryRow(fallbackSQL, args...).Scan(&fallbackCount); err != nil {
		fallbackCount = 0 // 如果查询失败，默认为0
	}

	return map[string]interface{}{
		"pv":             pv,
		"uv":             uv,
		"input_tokens":   inputTokens,
		"output_tokens":  outputTokens,
		"total_tokens":   totalTokens,
		"fallback_count": fallbackCount,
	}, nil
}

// 查询MCP Server的KPI数据
func queryMCPServerKpi(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	// PV 查询
	pvSQL := fmt.Sprintf("SELECT COUNT(*) FROM access_logs %s", whereSQL)
	var pv int64
	if err := db.QueryRow(pvSQL, args...).Scan(&pv); err != nil {
		return nil, fmt.Errorf("failed to query PV: %v", err)
	}

	// UV 查询（基于trace_id去重）
	uvSQL := fmt.Sprintf("SELECT COUNT(DISTINCT trace_id) FROM access_logs %s", whereSQL)
	var uv int64
	if err := db.QueryRow(uvSQL, args...).Scan(&uv); err != nil {
		return nil, fmt.Errorf("failed to query UV: %v", err)
	}

	// 流量统计查询
	trafficSQL := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(bytes_received), 0) as bytes_received,
			COALESCE(SUM(bytes_sent), 0) as bytes_sent
		FROM access_logs %s`, whereSQL)

	var bytesReceived, bytesSent int64
	if err := db.QueryRow(trafficSQL, args...).Scan(&bytesReceived, &bytesSent); err != nil {
		return nil, fmt.Errorf("failed to query traffic stats: %v", err)
	}

	return map[string]interface{}{
		"pv":             pv,
		"uv":             uv,
		"bytes_received": bytesReceived,
		"bytes_sent":     bytesSent,
	}, nil
}

// 处理图表查询的核心逻辑
func processChartQuery(payload map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	log.Printf("[ChartQuery] Processing chart query with payload: %+v", payload)

	// 解析时间范围和粒度
	timeRange, ok := payload["timeRange"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid timeRange format")
	}

	startTime, endTime, err := parseTimeRange(timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %v", err)
	}

	interval, _ := payload["interval"].(string)
	if interval == "" {
		interval = "60s" // 默认60秒粒度
	}

	scenario, _ := payload["scenario"].(string)
	bizType, _ := payload["bizType"].(string)

	// 构建基础查询条件
	whereClause := []string{"start_time >= ?", "start_time <= ?"}
	args := []interface{}{startTime, endTime}

	// 添加过滤条件
	if filters, ok := payload["filters"].(map[string]interface{}); ok {
		for key, value := range filters {
			if strVal, ok := value.(string); ok && strVal != "" {
				whereClause = append(whereClause, fmt.Sprintf("%s = ?", key))
				args = append(args, strVal)
			}
		}
	}
	appendBizTypeWhereClause(&whereClause, &args, bizType)
	whereSQL := "WHERE " + strings.Join(whereClause, " AND ")

	var result map[string]interface{}

	switch scenario {
	case "success_rate":
		result, err = querySuccessRateChart(whereSQL, args, interval)
	case "qps_total_simple":
		result, err = queryQPSChart(whereSQL, args, interval)
	case "token_rate": // Token消耗数/s
		result, err = queryTokenRateChart(whereSQL, args, interval)
	case "rt_distribution": // RT分布
		result, err = queryRTDistributionChart(whereSQL, args, interval)
	case "cache_hit_rate": // 缓存命中率
		result, err = queryCacheHitRateChart(whereSQL, args, interval)
	case "rate_limit": // 限流请求数/s
		result, err = queryRateLimitChart(whereSQL, args, interval)
	default:
		return nil, fmt.Errorf("unsupported scenario: %s", scenario)
	}

	if err != nil {
		return nil, err
	}

	duration := time.Since(start)
	log.Printf("[ChartQuery] ✓ SUCCESS: scenario=%s, bizType=%s, duration=%v", scenario, bizType, duration)

	return result, nil
}

// 查询成功率先图表数据
func querySuccessRateChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			COUNT(*) as total_requests,
			SUM(CASE WHEN response_code < 400 THEN 1 ELSE 0 END) as success_requests
		FROM access_logs %s 
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query success rate: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var successRates []float64

	rowCount := 0
	for rows.Next() {
		var timestampFloat float64
		var totalRequests, successRequests int64
		if err := rows.Scan(&timestampFloat, &totalRequests, &successRequests); err != nil {
			log.Printf("[DEBUG] Scan error: %v", err)
			continue
		}
		timestamp := int64(timestampFloat)

		log.Printf("[DEBUG] Row %d: timestamp=%d, total=%d, success=%d", rowCount, timestamp, totalRequests, successRequests)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		if totalRequests > 0 {
			successRate := float64(successRequests*100) / float64(totalRequests)
			successRates = append(successRates, successRate)
			log.Printf("[DEBUG] Success rate calculated: %.2f%%", successRate)
		} else {
			successRates = append(successRates, 0)
		}
		rowCount++
	}
	log.Printf("[DEBUG] Total rows processed: %d", rowCount)

	return map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"success_rate": successRates,
		},
	}, nil
}

// 查询QPS图表数据
func queryQPSChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			COUNT(*) as total_qps,
			SUM(CASE WHEN path LIKE '%%stream%%' THEN 1 ELSE 0 END) as stream_qps,
			SUM(CASE WHEN path NOT LIKE '%%stream%%' THEN 1 ELSE 0 END) as request_qps
		FROM access_logs %s 
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	log.Printf("[DEBUG] QPS Chart SQL: %s", sql)
	log.Printf("[DEBUG] QPS Chart Args: %v", args)

	rows, err := db.Query(sql, args...)
	if err != nil {
		log.Printf("[ERROR] Failed to execute QPS query: %v", err)
		return nil, fmt.Errorf("failed to query QPS: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var totalQPS, streamQPS, requestQPS []float64

	rowCount := 0
	for rows.Next() {
		var timestampFloat float64
		var totalCount, streamCount, requestCount int64
		if err := rows.Scan(&timestampFloat, &totalCount, &streamCount, &requestCount); err != nil {
			log.Printf("[ERROR] Failed to scan QPS row: %v", err)
			continue
		}
		timestamp := int64(timestampFloat)

		log.Printf("[DEBUG] QPS Row %d: timestamp=%d, total=%d, stream=%d, request=%d",
			rowCount, timestamp, totalCount, streamCount, requestCount)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		totalQPS = append(totalQPS, float64(totalCount)/float64(intervalSec))
		streamQPS = append(streamQPS, float64(streamCount)/float64(intervalSec))
		requestQPS = append(requestQPS, float64(requestCount)/float64(intervalSec))
		rowCount++
	}

	if err = rows.Err(); err != nil {
		log.Printf("[ERROR] QPS query iteration error: %v", err)
		return nil, fmt.Errorf("failed to iterate QPS results: %v", err)
	}

	log.Printf("[DEBUG] QPS Chart Results: timestamps=%d items, total_qps=%d items, stream_qps=%d items, request_qps=%d items",
		len(timestamps), len(totalQPS), len(streamQPS), len(requestQPS))

	result := map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"total_qps":   totalQPS,
			"stream_qps":  streamQPS,
			"request_qps": requestQPS,
		},
	}

	log.Printf("[DEBUG] QPS Chart Final Result: %+v", result)
	return result, nil
}

// 查询Token速率图表数据
func queryTokenRateChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	// 直接使用数据库中的token字段进行聚合
	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM access_logs %s 
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query token rate: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var inputRate, outputRate, totalRate []float64

	for rows.Next() {
		var timestampFloat float64
		var inputTokens, outputTokens, totalTokens int64
		if err := rows.Scan(&timestampFloat, &inputTokens, &outputTokens, &totalTokens); err != nil {
			continue
		}
		timestamp := int64(timestampFloat)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		inputRate = append(inputRate, float64(inputTokens)/float64(intervalSec))
		outputRate = append(outputRate, float64(outputTokens)/float64(intervalSec))
		totalRate = append(totalRate, float64(totalTokens)/float64(intervalSec))
	}

	log.Printf("[TokenRate] Processed %d time windows, interval=%ds", len(timestamps), intervalSec)

	return map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"input_token_rate":  inputRate,
			"output_token_rate": outputRate,
			"total_token_rate":  totalRate,
		},
	}, nil
}

// 查询RT分布图表数据
func queryRTDistributionChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			AVG(duration) as avg_rt,
			MAX(duration) as max_rt,
			MIN(duration) as min_rt
		FROM access_logs %s 
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query RT distribution: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var avgRT, p99RT, p95RT, p90RT, p50RT []float64

	for rows.Next() {
		var timestampFloat float64
		var avgDuration float64
		var maxDuration, minDuration int64
		if err := rows.Scan(&timestampFloat, &avgDuration, &maxDuration, &minDuration); err != nil {
			continue
		}
		timestamp := int64(timestampFloat)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		avgRT = append(avgRT, avgDuration)
		p99RT = append(p99RT, float64(maxDuration)*0.99)
		p95RT = append(p95RT, float64(maxDuration)*0.95)
		p90RT = append(p90RT, float64(maxDuration)*0.90)
		p50RT = append(p50RT, float64(minDuration)+(float64(maxDuration-minDuration)*0.5))
	}

	return map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"avg_rt": avgRT,
			"p99_rt": p99RT,
			"p95_rt": p95RT,
			"p90_rt": p90RT,
			"p50_rt": p50RT,
		},
	}, nil
}

// 查询缓存命中率图表数据
func queryCacheHitRateChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	// 这里简化处理，实际应该根据具体缓存字段判断
	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			COUNT(*) as total_requests,
			SUM(CASE WHEN response_code = 200 THEN 1 ELSE 0 END) as hit_requests,
			SUM(CASE WHEN response_code = 404 THEN 1 ELSE 0 END) as miss_requests
		FROM access_logs %s 
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query cache hit rate: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var hitRate, missRate, skipRate []float64

	for rows.Next() {
		var timestampFloat float64
		var totalRequests, hitRequests, missRequests int64
		if err := rows.Scan(&timestampFloat, &totalRequests, &hitRequests, &missRequests); err != nil {
			continue
		}
		timestamp := int64(timestampFloat)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		if totalRequests > 0 {
			hitRate = append(hitRate, float64(hitRequests*100)/float64(totalRequests))
			missRate = append(missRate, float64(missRequests*100)/float64(totalRequests))
			skipRate = append(skipRate, float64((totalRequests-hitRequests-missRequests)*100)/float64(totalRequests))
		} else {
			hitRate = append(hitRate, 0)
			missRate = append(missRate, 0)
			skipRate = append(skipRate, 0)
		}
	}

	return map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"hit_rate":  hitRate,
			"miss_rate": missRate,
			"skip_rate": skipRate,
		},
	}, nil
}

// 查询限流请求数图表数据
func queryRateLimitChart(whereSQL string, args []interface{}, interval string) (map[string]interface{}, error) {
	intervalSec := parseInterval(interval)
	groupByExpr := buildGroupByExpression(intervalSec)
	timestampExpr := buildTimestampExpression(intervalSec)

	sql := fmt.Sprintf(`
		SELECT 
			%s as timestamp,
			COUNT(*) as rate_limit_count
		FROM access_logs %s AND response_code = 429
		GROUP BY %s 
		ORDER BY timestamp`, timestampExpr, whereSQL, groupByExpr)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rate limit: %v", err)
	}
	defer rows.Close()

	var timestamps []int64
	var rateLimitCount []float64

	for rows.Next() {
		var timestampFloat float64
		var count int64
		if err := rows.Scan(&timestampFloat, &count); err != nil {
			continue
		}
		timestamp := int64(timestampFloat)

		timestamps = append(timestamps, timestamp*1000) // 转换为毫秒
		rateLimitCount = append(rateLimitCount, float64(count)/float64(intervalSec))
	}

	return map[string]interface{}{
		"timestamps": timestamps,
		"values": map[string][]float64{
			"rate_limit_count": rateLimitCount,
		},
	}, nil
}

// 处理表格查询的核心逻辑
func processTableQuery(payload map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	log.Printf("[TableQuery] Processing table query with payload: %+v", payload)

	// 解析时间范围
	timeRange, ok := payload["timeRange"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid timeRange format")
	}

	startTime, endTime, err := parseTimeRange(timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %v", err)
	}

	tableType, _ := payload["tableType"].(string)
	bizType, _ := payload["bizType"].(string)

	// 构建基础查询条件
	whereClause := []string{"start_time >= ?", "start_time <= ?"}
	args := []interface{}{startTime, endTime}

	// 添加过滤条件
	if filters, ok := payload["filters"].(map[string]interface{}); ok {
		for key, value := range filters {
			if strVal, ok := value.(string); ok && strVal != "" {
				whereClause = append(whereClause, fmt.Sprintf("%s = ?", key))
				args = append(args, strVal)
			}
		}
	}
	appendBizTypeWhereClause(&whereClause, &args, bizType)
	whereSQL := "WHERE " + strings.Join(whereClause, " AND ")

	var result map[string]interface{}

	switch tableType {
	case "method_distribution":
		result, err = queryMethodDistributionTable(whereSQL, args)
	case "status_code_distribution":
		result, err = queryStatusCodeDistributionTable(whereSQL, args)
	case "model_token_stats":
		result, err = queryModelTokenStatsTable(whereSQL, args)
	case "consumer_token_stats":
		result, err = queryConsumerTokenStatsTable(whereSQL, args)
	case "service_token_stats":
		result, err = queryServiceTokenStatsTable(whereSQL, args)
	case "error_requests":
		result, err = queryErrorRequestsTable(whereSQL, args)
	case "rate_limited_consumers":
		result, err = queryRateLimitedConsumersTable(whereSQL, args)
	case "risk_types":
		result, err = queryRiskTypesTable(whereSQL, args)
	case "risk_consumers":
		result, err = queryRiskConsumersTable(whereSQL, args)
	default:
		return nil, fmt.Errorf("unsupported tableType: %s", tableType)
	}

	if err != nil {
		return nil, err
	}

	duration := time.Since(start)
	log.Printf("[TableQuery] ✓ SUCCESS: tableType=%s, bizType=%s, duration=%v", tableType, bizType, duration)

	return result, nil
}

// 查询方法分布表格数据
func queryMethodDistributionTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			method,
			COUNT(*) as request_count,
			AVG(duration) as avg_duration
		FROM access_logs %s
		GROUP BY method 
		ORDER BY request_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query method distribution: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var method string
		var requestCount int64
		var avgDuration float64

		if err := rows.Scan(&method, &requestCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"method":        method,
			"request_count": requestCount,
			"avg_duration":  avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询状态码分布表格数据
func queryStatusCodeDistributionTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			response_code,
			COUNT(*) as request_count,
			AVG(duration) as avg_duration
		FROM access_logs %s 
		GROUP BY response_code 
		ORDER BY request_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query status code distribution: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var statusCode string
		var requestCount int64
		var avgDuration float64

		if err := rows.Scan(&statusCode, &requestCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"status_code":   statusCode,
			"request_count": requestCount,
			"avg_duration":  avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 构建模型非空过滤条件
func buildModelNotNullCondition() string {
	return "model IS NOT NULL AND model != '' AND model != 'unknown'"
}

// 构建MCP工具非空过滤条件
// mcp工具调用 total_tokens = 0 且 model 为 unknown 或空字符串的记录
func buildMCPNotNullCondition() string {
	return "mcp_server != '' AND mcp_server != 'unknown' AND total_tokens = 0 OR model IS NULL OR model = '' OR model = 'unknown'"
}

// 查询模型token统计数据
func queryModelTokenStatsTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			model,
			COUNT(*) as request_count,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens
		FROM access_logs %s
		GROUP BY model 
		ORDER BY total_tokens DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query model token stats: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var model string
		var requestCount, inputTokens, outputTokens, totalTokens int64

		if err := rows.Scan(&model, &requestCount, &inputTokens, &outputTokens, &totalTokens); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"model":         model,
			"request_count": requestCount,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  totalTokens,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询消费者token统计数据
func queryConsumerTokenStatsTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			consumer,
			COUNT(*) as request_count,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens
		FROM access_logs %s 
		AND consumer IS NOT NULL AND consumer != ''
		GROUP BY consumer 
		ORDER BY total_tokens DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query consumer token stats: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var consumer string
		var requestCount, inputTokens, outputTokens, totalTokens int64

		if err := rows.Scan(&consumer, &requestCount, &inputTokens, &outputTokens, &totalTokens); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"consumer":      consumer,
			"request_count": requestCount,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  totalTokens,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询服务token统计数据
func queryServiceTokenStatsTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			service,
			COUNT(*) as request_count,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens
		FROM access_logs %s 
		AND service IS NOT NULL AND service != ''
		GROUP BY service 
		ORDER BY total_tokens DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query service token stats: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var service string
		var requestCount, inputTokens, outputTokens, totalTokens int64

		if err := rows.Scan(&service, &requestCount, &inputTokens, &outputTokens, &totalTokens); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"service":       service,
			"request_count": requestCount,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  totalTokens,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询错误请求表格数据
func queryErrorRequestsTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			model,
			consumer,
			response_code,
			COUNT(*) as error_count,
			AVG(duration) as avg_duration
		FROM access_logs %s AND response_code >= 400
		GROUP BY model, consumer, response_code 
		ORDER BY error_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query error requests: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var model, consumer, statusCode string
		var errorCount int64
		var avgDuration float64

		if err := rows.Scan(&model, &consumer, &statusCode, &errorCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"model":        model,
			"consumer":     consumer,
			"status_code":  statusCode,
			"error_count":  errorCount,
			"avg_duration": avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询限流消费者表格数据
func queryRateLimitedConsumersTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			consumer,
			COUNT(*) as rate_limit_count,
			AVG(duration) as avg_duration
		FROM access_logs %s AND response_code = 429
		GROUP BY consumer 
		ORDER BY rate_limit_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rate limited consumers: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var consumer string
		var rateLimitCount int64
		var avgDuration float64

		if err := rows.Scan(&consumer, &rateLimitCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"consumer":         consumer,
			"rate_limit_count": rateLimitCount,
			"avg_duration":     avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询风险类型表格数据
func queryRiskTypesTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	// 这里简化处理，实际应该根据具体风险字段判断
	sql := fmt.Sprintf(`
		SELECT 
			response_code as risk_type,
			COUNT(*) as risk_count,
			AVG(duration) as avg_duration
		FROM access_logs %s AND response_code >= 400
		GROUP BY response_code 
		ORDER BY risk_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk types: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var riskType string
		var riskCount int64
		var avgDuration float64

		if err := rows.Scan(&riskType, &riskCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"risk_type":    riskType,
			"risk_count":   riskCount,
			"avg_duration": avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 查询风险消费者表格数据
func queryRiskConsumersTable(whereSQL string, args []interface{}) (map[string]interface{}, error) {
	sql := fmt.Sprintf(`
		SELECT 
			consumer,
			COUNT(*) as risk_count,
			AVG(duration) as avg_duration
		FROM access_logs %s AND response_code >= 400
		GROUP BY consumer 
		ORDER BY risk_count DESC`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk consumers: %v", err)
	}
	defer rows.Close()

	var data []map[string]interface{}

	for rows.Next() {
		var consumer string
		var riskCount int64
		var avgDuration float64

		if err := rows.Scan(&consumer, &riskCount, &avgDuration); err != nil {
			continue
		}

		data = append(data, map[string]interface{}{
			"consumer":     consumer,
			"risk_count":   riskCount,
			"avg_duration": avgDuration,
		})
	}

	return map[string]interface{}{
		"data": data,
	}, nil
}

// 解析时间范围
func parseTimeRange(timeRange map[string]interface{}) (string, string, error) {
	start, ok := timeRange["start"].(string)
	if !ok {
		return "", "", fmt.Errorf("missing start time")
	}

	end, ok := timeRange["end"].(string)
	if !ok {
		return "", "", fmt.Errorf("missing end time")
	}

	return start, end, nil
}

// 解析时间间隔
func parseInterval(interval string) int {
	switch interval {
	case "1s":
		return 1
	case "15s":
		return 15
	case "60s":
		return 60
	case "300s": // 5分钟
		return 300
	case "600s": // 10分钟
		return 600
	case "1800s": // 30分钟
		return 1800
	case "3600s": // 1小时
		return 3600
	case "86400s": // 1天
		return 86400
	default:
		return 60 // 默认60秒
	}
}

// 构建GROUP BY表达式
func buildGroupByExpression(intervalSec int) string {
	switch {
	case intervalSec == 1: // 1s
		// 按秒精确分组
		return "DATE_FORMAT(start_time, '%Y-%m-%d %H:%i:%s')"
	case intervalSec <= 60: // 15s, 30s, 60s
		// 按指定秒数区间分组
		return fmt.Sprintf("CONCAT(DATE_FORMAT(start_time, '%%Y-%%m-%%d %%H:%%i:'), LPAD(FLOOR(SECOND(start_time)/%d)*%d, 2, '0'))", intervalSec, intervalSec)
	case intervalSec <= 3600: // 5分钟到1小时
		// 按分钟分组
		minutes := intervalSec / 60
		return fmt.Sprintf("CONCAT(DATE_FORMAT(start_time, '%%Y-%%m-%%d %%H:'), LPAD(FLOOR(MINUTE(start_time)/%d)*%d, 2, '0'))", minutes, minutes)
	case intervalSec <= 86400: // 1天
		// 按小时分组
		return "DATE_FORMAT(start_time, '%Y-%m-%d %H')"
	default: // 超过1天
		// 按天分组
		return "DATE_FORMAT(start_time, '%Y-%m-%d')"
	}
}

// 构建时间戳表达式
func buildTimestampExpression(intervalSec int) string {
	switch {
	case intervalSec == 1: // 1s
		// 秒级精度时间戳
		return "UNIX_TIMESTAMP(DATE_FORMAT(MIN(start_time), '%Y-%m-%d %H:%i:%s'))"
	case intervalSec <= 60: // 15s, 30s, 60s
		// 按指定秒数区间的起始时间
		return fmt.Sprintf("UNIX_TIMESTAMP(CONCAT(DATE_FORMAT(MIN(start_time), '%%Y-%%m-%%d %%H:%%i:'), LPAD(FLOOR(SECOND(MIN(start_time))/%d)*%d, 2, '0')))", intervalSec, intervalSec)
	case intervalSec <= 3600: // 5分钟到1小时
		// 分钟级精度时间戳
		minutes := intervalSec / 60
		return fmt.Sprintf("UNIX_TIMESTAMP(CONCAT(DATE_FORMAT(MIN(start_time), '%%Y-%%m-%%d %%H:'), LPAD(FLOOR(MINUTE(MIN(start_time))/%d)*%d, 2, '0'), ':00'))", minutes, minutes)
	case intervalSec <= 86400: // 1天
		// 小时级精度时间戳
		return "UNIX_TIMESTAMP(DATE_FORMAT(MIN(start_time), '%Y-%m-%d %H:00:00'))"
	default: // 超过1天
		// 天级精度时间戳
		return "UNIX_TIMESTAMP(DATE_FORMAT(MIN(start_time), '%Y-%m-%d 00:00:00'))"
	}
}
