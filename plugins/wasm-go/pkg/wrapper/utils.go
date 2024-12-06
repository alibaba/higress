package wrapper

import (
	"encoding/json"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
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
