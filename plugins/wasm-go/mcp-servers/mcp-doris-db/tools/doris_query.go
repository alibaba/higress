package tools

import (
	"database/sql"
	_ "github.com/apache/doris-go-driver/doris"
	"fmt"
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
func (q *DorisQueryer) QueryTable(table string, fields []string, where string, limit, offset int) ([]map[string]interface{}, error) {
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
func (q *DorisQueryer) ExecuteSQL(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
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