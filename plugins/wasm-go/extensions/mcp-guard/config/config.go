package config

import (
    "github.com/tidwall/gjson"
)

// PluginConfig is a minimal config for the demo MVP.
// In a full implementation, allowed capabilities would come per-route
// and subject policies would be distributed via ECDS.
type PluginConfig struct {
    // Allowed capabilities for this route (per-route in real control-plane)
    AllowedCapabilities []string `json:"allowedCapabilities"`

    // Subject -> capabilities mapping (simulated ECDS subject policy)
    SubjectPolicy map[string][]string `json:"subjectPolicy"`

    // From where to read requested capability (header name)
    RequestedCapabilityHeader string `json:"requestedCapabilityHeader"`

    // Shadow mode: only log decision, never block
    Shadow bool `json:"shadow"`

    // Optional: rules to decide allowed capabilities by path prefix
    Rules []Rule `json:"rules"`
}

type Rule struct {
    PathPrefix string   `json:"pathPrefix"`
    AllowedCapabilities []string `json:"allowedCapabilities"`
}

func (c *PluginConfig) FromJson(json gjson.Result) {
    if v := json.Get("allowedCapabilities"); v.Exists() {
        c.AllowedCapabilities = nil
        for _, it := range v.Array() {
            c.AllowedCapabilities = append(c.AllowedCapabilities, it.String())
        }
    }
    if v := json.Get("subjectPolicy"); v.Exists() {
        c.SubjectPolicy = map[string][]string{}
        v.ForEach(func(key, val gjson.Result) bool {
            arr := []string{}
            for _, it := range val.Array() {
                arr = append(arr, it.String())
            }
            c.SubjectPolicy[key.String()] = arr
            return true
        })
    }
    if v := json.Get("requestedCapabilityFrom.header"); v.Exists() {
        c.RequestedCapabilityHeader = v.String()
    }
    if v := json.Get("shadow"); v.Exists() {
        c.Shadow = v.Bool()
    }
    if v := json.Get("rules"); v.Exists() {
        c.Rules = nil
        for _, r := range v.Array() {
            c.Rules = append(c.Rules, Rule{
                PathPrefix: r.Get("pathPrefix").String(),
                AllowedCapabilities: func() []string {
                    arr := []string{}
                    for _, it := range r.Get("allowedCapabilities").Array() {
                        arr = append(arr, it.String())
                    }
                    return arr
                }(),
            })
        }
    }
}

func (c *PluginConfig) Validate() error { return nil }
func (c *PluginConfig) Complete() error { return nil }
