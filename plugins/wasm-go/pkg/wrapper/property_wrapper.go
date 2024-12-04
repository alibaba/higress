package wrapper

import (
	"encoding/json"
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

const (
	CustomLogKey = "custom_log"
	AILogKey     = "ai_log"
)

func unmarshalStr(marshalledJsonStr string) string {
	// e.g. "{\"field1\":\"value1\",\"field2\":\"value2\"}"
	var jsonStr string
	err := json.Unmarshal([]byte(marshalledJsonStr), &jsonStr)
	if err != nil {
		proxywasm.LogErrorf("failed to unmarshal json string, raw string is: %s, err is: %v", marshalledJsonStr, err)
		return ""
	}
	// e.g. {"field1":"value1","field2":"value2"}
	return jsonStr
}

func marshalStr(raw string) string {
	// e.g. {"field1":"value1","field2":"value2"}
	helper := map[string]string{
		"placeholder": raw,
	}
	marshalledHelper, _ := json.Marshal(helper)
	marshalledRaw := gjson.GetBytes(marshalledHelper, "placeholder").Raw
	if len(marshalledRaw) >= 2 {
		// e.g. {\"field1\":\"value1\",\"field2\":\"value2\"}
		return marshalledRaw[1 : len(marshalledRaw)-1]
	} else {
		proxywasm.LogErrorf("failed to marshal json string, raw string is: %s", raw)
		return ""
	}
}

func extendAccessLog(propertyKey string, items map[string]interface{}) error {
	// e.g. {\"field1\":\"value1\",\"field2\":\"value2\"}
	preMarshalledJsonLogStr, _ := proxywasm.GetProperty([]string{propertyKey})
	customLog := map[string]interface{}{}
	if string(preMarshalledJsonLogStr) != "" {
		// e.g. {"field1":"value1","field2":"value2"}
		preJsonLogStr := unmarshalStr(fmt.Sprintf(`"%s"`, string(preMarshalledJsonLogStr)))
		err := json.Unmarshal([]byte(preJsonLogStr), &customLog)
		if err != nil {
			proxywasm.LogErrorf("failed to unmarshal custom_log, will overwrite old custom_log, err is: %v", err)
		}
	}
	// update customLog
	for k, v := range items {
		customLog[k] = v
	}
	// e.g. {"field1":"value1","field2":2,"field3":"value3"}
	jsonStr, _ := json.Marshal(customLog)
	// e.g. {\"field1\":\"value1\",\"field2\":2,\"field3\":\"value3\"}
	marshalledJsonStr := marshalStr(string(jsonStr))
	if err := proxywasm.SetProperty([]string{propertyKey}, []byte(marshalledJsonStr)); err != nil {
		proxywasm.LogErrorf("failed to set custom_log in filter state, raw is %s, err is %v", marshalledJsonStr, err)
		return err
	}
	return nil
}
