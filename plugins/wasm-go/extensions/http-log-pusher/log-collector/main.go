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

// 1. 定义与 Wasm 插件发送格式一致的结构体
type LogEntry struct {
	StartTime    string `json:"start_time"`    // 请求开始时间 (RFC3339)
	Authority    string `json:"authority"`     // 对应数据库中的 service
	TraceID      string `json:"trace_id"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	ResponseCode int    `json:"response_code"` // 响应状态码
	Duration     int64  `json:"duration"`      // 请求总耗时(ms)
	AILog        string `json:"ai_log"`        // WASM AI 日志 (JSON 字符串)
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
		log.Printf("Error decoding JSON: %v", err)
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

	// 警告：这里的代码是为了 POC 写的，简单粗暴。
	// 生产环境应该使用 sqlx 或者 GORM 的 Batch Insert。
	valueStrings := []string{}
	valueArgs := []interface{}{}

	for _, entry := range chunk {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?)")

		// 转换 RFC3339 时间为 MySQL datetime 格式
		startTime := entry.StartTime
		if t, err := time.Parse(time.RFC3339, entry.StartTime); err == nil {
			startTime = t.Format("2006-01-02 15:04:05")
		}

		valueArgs = append(valueArgs,
			startTime,
			entry.TraceID,
			entry.Authority,
			entry.Method,
			entry.Path,
			entry.ResponseCode,
			entry.Duration,
			entry.AILog,
		)
	}

	stmt := fmt.Sprintf("INSERT INTO access_logs (start_time, trace_id, authority, method, path, response_code, duration, ai_log) VALUES %s",
		strings.Join(valueStrings, ","))

	// 执行写入
	start := time.Now()
	_, err := db.Exec(stmt, valueArgs...)
	if err != nil {
		// 这里体现了 POC 方案的脆弱性：如果 DB 挂了，这一批日志就直接丢了
		log.Printf("Failed to insert batch logs: %v", err)
	} else {
		log.Printf("Flushed %d logs to MySQL in %v", len(chunk), time.Since(start))
	}
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
		whereClause = append(whereClause, "ts >= ?")
		args = append(args, start)
	}
	if end := params.Get("end"); end != "" {
		whereClause = append(whereClause, "ts <= ?")
		args = append(args, end)
	}
	
	// 服务名查询
	if service := params.Get("service"); service != "" {
		whereClause = append(whereClause, "service = ?")
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
		whereClause = append(whereClause, "status = ?")
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
	
	// 排序参数
	sortBy := "ts"
	sortOrder := "DESC"
	if sb := params.Get("sort_by"); sb != "" {
		if sb == "ts" || sb == "status" || sb == "latency" || sb == "service" || sb == "method" {
			sortBy = sb
		}
	}
	if so := params.Get("sort_order"); so != "" {
		if so == "ASC" || so == "asc" {
			sortOrder = "ASC"
		}
	}
	
	// 构建查询 SQL
	querySQL := fmt.Sprintf(
		"SELECT start_time, trace_id, authority, method, path, response_code, duration, ai_log FROM access_logs %s ORDER BY %s %s LIMIT ? OFFSET ?",
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
	
	// 解析查询结果
	logs := []LogEntry{}
	for rows.Next() {
		var entry LogEntry
		var startTime time.Time

		err := rows.Scan(
			&startTime, &entry.TraceID, &entry.Authority, &entry.Method,
			&entry.Path, &entry.ResponseCode, &entry.Duration, &entry.AILog,
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