package test

import (
	"bytes"
	"fmt"

	"github.com/tidwall/resp"
)

// CreateRedisResp create the correct RESP format response for any type.
// Supports: string, int, bool, float64, nil, []interface{}, error
func CreateRedisResp(value interface{}) []byte {
	var buf bytes.Buffer
	wr := resp.NewWriter(&buf)

	switch v := value.(type) {
	case nil:
		wr.WriteNull()
	case string:
		wr.WriteString(v)
	case int:
		wr.WriteInteger(v)
	case bool:
		wr.WriteInteger(boolToInt(v))
	case float64:
		wr.WriteString(fmt.Sprintf("%v", v))
	case []interface{}:
		// Handle array type
		arr := make([]resp.Value, 0, len(v))
		for _, item := range v {
			arr = append(arr, createRespValue(item))
		}
		wr.WriteArray(arr)
	case error:
		wr.WriteError(v)
	default:
		// For other types, convert to string
		wr.WriteString(fmt.Sprintf("%v", v))
	}

	return buf.Bytes()
}

// CreateRedisRespNull create the correct RESP format null response.
// Convenience function for null values.
func CreateRedisRespNull() []byte {
	return CreateRedisResp(nil)
}

// CreateRedisRespError create the correct RESP format error response.
// Convenience function for error values.
func CreateRedisRespError(message string) []byte {
	return CreateRedisResp(fmt.Errorf("%s", message))
}

// CreateRedisRespArray create the correct RESP format array response.
// Convenience function for array values.
// Supports different types of values: string, int, bool, float64, etc.
func CreateRedisRespArray(values []interface{}) []byte {
	return CreateRedisResp(values)
}

// CreateRedisRespString create the correct RESP format string response.
// Convenience function for string values.
func CreateRedisRespString(value string) []byte {
	return CreateRedisResp(value)
}

// CreateRedisRespInt create the correct RESP format int response.
// Convenience function for int values.
func CreateRedisRespInt(value int) []byte {
	return CreateRedisResp(value)
}

// CreateRedisRespBool create the correct RESP format bool response.
// Convenience function for bool values.
func CreateRedisRespBool(value bool) []byte {
	return CreateRedisResp(value)
}

// CreateRedisRespFloat create the correct RESP format float response.
// Convenience function for float64 values.
func CreateRedisRespFloat(value float64) []byte {
	return CreateRedisResp(value)
}

// createRespValue create the correct RESP format value.
func createRespValue(value interface{}) resp.Value {
	switch v := value.(type) {
	case nil:
		return resp.NullValue()
	case string:
		return resp.StringValue(v)
	case int:
		return resp.IntegerValue(v)
	case bool:
		return resp.BoolValue(v)
	case float64:
		return resp.FloatValue(v)
	default:
		return resp.StringValue(fmt.Sprintf("%v", v))
	}
}

// boolToInt convert bool to int.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
