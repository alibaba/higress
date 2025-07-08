package tools

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // Doris 兼容 MySQL 协议，使用 MySQL 驱动
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

// DorisQueryer 用于管理 Doris 数据库连接和查询
// 你可以在这里实现连接池、查询、参数校验等功能

type DorisQueryer struct {
	DB *sql.DB
}

// NewDorisQueryer 创建新的 DorisQueryer 实例
func NewDorisQueryer(dsn string) (*DorisQueryer, error) {
	db, err := sql.Open("doris", dsn)
	if err != nil {
		return nil, err
	}
	return &DorisQueryer{DB: db}, nil
}

// QueryTable 查询表数据，支持字段、条件、分页
func (q *DorisQueryer) QueryTable(table string, fields []string, where string, limit, offset int, perm *PermissionConfig) ([]map[string]interface{}, error) {
	// 参数校验
	if err := perm.CheckTableAndFields(table, fields); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM %s", joinFields(fields), table)
	if where != "" {
		query += " WHERE " + where
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}
	rows, err := q.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToMap(rows)
}

// ExecuteSQL 执行自定义SQL
func (q *DorisQueryer) ExecuteSQL(sqlStr string, args []interface{}, perm *PermissionConfig) ([]map[string]interface{}, error) {
	// SQL安全校验
	if !IsSafeSQL(sqlStr) {
		return nil, fmt.Errorf("SQL语句包含危险操作")
	}
	rows, err := q.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToMap(rows)
}

// joinFields 辅助函数，将字段数组拼接为逗号分隔字符串
func joinFields(fields []string) string {
	if len(fields) == 0 {
		return "*"
	}
	return fmt.Sprintf("%s", sqlFields(fields))
}

// sqlFields 用于安全拼接字段名
func sqlFields(fields []string) string {
	res := ""
	for i, f := range fields {
		if i > 0 {
			res += ", "
		}
		res += "`" + f + "`"
	}
	return res
}

// rowsToMap 将查询结果转换为map数组
func rowsToMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(values))
		for i := range values {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}
		results = append(results, rowMap)
	}
	return results, nil
}

// PermissionConfig 用于加载权限白名单
type PermissionConfig struct {
	Whitelist map[string][]string `yaml:"whitelist"`
}

// LoadPermissionConfig 加载权限配置
func LoadPermissionConfig(path string) (*PermissionConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PermissionConfig
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// CheckTableAndFields 校验表名和字段名是否在白名单
func (cfg *PermissionConfig) CheckTableAndFields(table string, fields []string) error {
	allowedFields, ok := cfg.Whitelist[table]
	if !ok {
		return fmt.Errorf("表 %s 不在白名单", table)
	}
	if len(fields) == 0 {
		return nil // 允许全部字段
	}
	allowed := make(map[string]struct{})
	for _, f := range allowedFields {
		allowed[f] = struct{}{}
	}
	for _, f := range fields {
		if _, ok := allowed[f]; !ok {
			return fmt.Errorf("字段 %s 不在表 %s 白名单", f, table)
		}
	}
	return nil
}

// IsSafeSQL 检查SQL语句是否安全（禁止危险操作）
func IsSafeSQL(sqlStr string) bool {
	s := strings.ToLower(sqlStr)
	forbidden := []string{"drop ", "delete ", "update ", "truncate ", "alter ", ";"}
	for _, f := range forbidden {
		if strings.Contains(s, f) {
			return false
		}
	}
	return true
} 