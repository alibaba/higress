package wrapper

import (
	"bytes"
	"encoding/json"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

func UnmarshalStr(marshalledJsonStr string) string {
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

func MarshalStr(raw string) string {
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

func GetPluginFingerPrint() string {
	pluginName, _ := proxywasm.GetProperty([]string{"plugin_name"})
	return string(pluginName)
}

func GetValueFromBody(data []byte, paths []string) *gjson.Result {
	for _, path := range paths {
		obj := gjson.GetBytes(data, path)
		if obj.Exists() {
			return &obj
		}
	}
	return nil
}

func UnifySSEChunk(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data
}
