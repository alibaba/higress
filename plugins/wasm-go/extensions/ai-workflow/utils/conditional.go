package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/*
用来比较数据大小:
eq arg1 arg2： arg1 == arg2时为true
ne arg1 arg2： arg1 != arg2时为true
lt arg1 arg2： arg1 < arg2时为true
le arg1 arg2： arg1 <= arg2时为true
gt arg1 arg2： arg1 > arg2时为true
ge arg1 arg2： arg1 >= arg2时为true
*/
type CompareFunc func(a, b float64) bool

var operators = map[string]interface{}{
	//CompareFunc 用来比较float64 数据大小:
	"eq": func(a, b float64) bool { return a == b },
	"ge": func(a, b float64) bool { return a >= b },
	"le": func(a, b float64) bool { return a <= b },
	"gt": func(a, b float64) bool { return a > b },
	"lt": func(a, b float64) bool { return a < b },
	//todo 添加别的判断函数
}

// 执行判断条件
func ExecConditionalStr(ConditionalStr string) (bool, error) {
	fields := strings.Fields(ConditionalStr)
	if len(fields) != 3 {
		return false, fmt.Errorf("invalid conditional str %s,fields num is %d", ConditionalStr, len(fields))
	}
	compareFunc := operators[fields[0]]
	switch fc := compareFunc.(type) {
	default:
		return false, fmt.Errorf("invalid conditional func %v", compareFunc)
	case func(a, b float64) bool:
		a, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return false, fmt.Errorf("invalid conditional str %s", ConditionalStr)
		}
		b, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return false, fmt.Errorf("invalid conditional str %s", ConditionalStr)
		}
		return fc(a, b), nil
	}

}

// 通过正泽表达式寻找模板中的 ｛｛foo｝｝ 字符串foo
// 返回 {{foo}} : foo
func ParseTmplStr(tmpl string) map[string]string {
	result := make(map[string]string)
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)
	matches := re.FindAllStringSubmatch(tmpl, -1)
	for _, match := range matches {
		result[match[0]] = match[1]
	}
	return result
}

// 使用kv替换模板中的字符
// 例如 模板是`hello,{{foo}}` 使用{"{{foo}}":"bot"} 替换后为`hello,bot`
func ReplacedStr(tmpl string, kvs map[string]string) string {

	for k, v := range kvs {
		tmpl = strings.Replace(tmpl, k, v, -1)
	}

	return tmpl
}
