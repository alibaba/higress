package config

import (
	"regexp"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	MaxRisk    = "max"
	HighRisk   = "high"
	MediumRisk = "medium"
	LowRisk    = "low"
	NoRisk     = "none"

	S4Sensitive = "s4"
	S3Sensitive = "s3"
	S2Sensitive = "s2"
	S1Sensitive = "s1"
	NoSensitive = "s0"

	ContentModerationType      = "contentModeration"
	PromptAttackType           = "promptAttack"
	SensitiveDataType          = "sensitiveData"
	MaliciousUrlDataType       = "maliciousUrl"
	ModelHallucinationDataType = "modelHallucination"

	// Actions
	MultiModalGuard          = "MultiModalGuard"
	MultiModalGuardForBase64 = "MultiModalGuardForBase64"
	TextModerationPlus       = "TextModerationPlus"
)

type Response struct {
	Code      int    `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
	Data      Data   `json:"Data"`
}

type Data struct {
	RiskLevel   string   `json:"RiskLevel"`
	AttackLevel string   `json:"AttackLevel,omitempty"`
	Result      []Result `json:"Result,omitempty"`
	Advice      []Advice `json:"Advice,omitempty"`
	Detail      []Detail `json:"Detail,omitempty"`
}

type Result struct {
	RiskWords   string  `json:"RiskWords,omitempty"`
	Description string  `json:"Description,omitempty"`
	Confidence  float64 `json:"Confidence,omitempty"`
	Label       string  `json:"Label,omitempty"`
}

type Advice struct {
	Answer     string `json:"Answer,omitempty"`
	HitLabel   string `json:"HitLabel,omitempty"`
	HitLibName string `json:"HitLibName,omitempty"`
}

type Detail struct {
	Suggestion string `json:"Suggestion,omitempty"`
	Type       string `json:"Type,omitempty"`
	Level      string `json:"Level,omitempty"`
}

type AISecurityConfig struct {
	Client                        wrapper.HttpClient
	Host                          string
	AK                            string
	SK                            string
	Token                         string
	Action                        string
	CheckRequest                  bool
	RequestCheckService           string
	RequestContentJsonPath        string
	CheckResponse                 bool
	ResponseCheckService          string
	ResponseContentJsonPath       string
	ResponseStreamContentJsonPath string
	DenyCode                      int64
	DenyMessage                   string
	ProtocolOriginal              bool
	RiskLevelBar                  string
	ContentModerationLevelBar     string
	PromptAttackLevelBar          string
	SensitiveDataLevelBar         string
	MaliciousUrlLevelBar          string
	ModelHallucinationLevelBar    string
	Timeout                       uint32
	BufferLimit                   int
	Metrics                       map[string]proxywasm.MetricCounter
	ConsumerRequestCheckService   []map[string]interface{}
	ConsumerResponseCheckService  []map[string]interface{}
	ConsumerRiskLevel             []map[string]interface{}
}

type Matcher struct {
	Exact  string
	Prefix string
	Re     *regexp.Regexp
}

func (m *Matcher) match(consumer string) bool {
	if m.Exact != "" {
		return consumer == m.Exact
	} else if m.Prefix != "" {
		return strings.HasPrefix(consumer, m.Prefix)
	} else if m.Re != nil {
		return m.Re.MatchString(consumer)
	} else {
		return false
	}
}

func (config *AISecurityConfig) IncrementCounter(metricName string, inc uint64) {
	counter, ok := config.Metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.Metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func (config *AISecurityConfig) GetRequestCheckService(consumer string) string {
	result := config.RequestCheckService
	for _, obj := range config.ConsumerRequestCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if requestCheckService, ok := obj["requestCheckService"]; ok {
					result, _ = requestCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetResponseCheckService(consumer string) string {
	result := config.ResponseCheckService
	for _, obj := range config.ConsumerResponseCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if responseCheckService, ok := obj["responseCheckService"]; ok {
					result, _ = responseCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetRiskLevelBar(consumer string) string {
	result := config.RiskLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if riskLevelBar, ok := obj["riskLevelBar"]; ok {
					result, _ = riskLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetContentModerationLevelBar(consumer string) string {
	result := config.ContentModerationLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if contentModerationLevelBar, ok := obj["contentModerationLevelBar"]; ok {
					result, _ = contentModerationLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetPromptAttackLevelBar(consumer string) string {
	result := config.PromptAttackLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if promptAttackLevelBar, ok := obj["promptAttackLevelBar"]; ok {
					result, _ = promptAttackLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetSensitiveDataLevelBar(consumer string) string {
	result := config.SensitiveDataLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if sensitiveDataLevelBar, ok := obj["sensitiveDataLevelBar"]; ok {
					result, _ = sensitiveDataLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetMaliciousUrlLevelBar(consumer string) string {
	result := config.MaliciousUrlLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if maliciousUrlLevelBar, ok := obj["maliciousUrlLevelBar"]; ok {
					result, _ = maliciousUrlLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetModelHallucinationLevelBar(consumer string) string {
	result := config.ModelHallucinationLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if modelHallucinationLevelBar, ok := obj["modelHallucinationLevelBar"]; ok {
					result, _ = modelHallucinationLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func LevelToInt(riskLevel string) int {
	// First check against our defined constants
	switch strings.ToLower(riskLevel) {
	case MaxRisk, S4Sensitive:
		return 4
	case HighRisk, S3Sensitive:
		return 3
	case MediumRisk, S2Sensitive:
		return 2
	case LowRisk, S1Sensitive:
		return 1
	case NoRisk, NoSensitive:
		return 0
	default:
		return -1
	}
}

func IsRiskLevelAcceptable(action string, data Data, config AISecurityConfig, consumer string) bool {
	if action == MultiModalGuard || action == MultiModalGuardForBase64 {
		// Check top-level risk levels for MultiModalGuard
		if LevelToInt(data.RiskLevel) >= LevelToInt(config.GetContentModerationLevelBar(consumer)) {
			return false
		}
		// Also check AttackLevel for prompt attack detection
		if LevelToInt(data.AttackLevel) >= LevelToInt(config.GetPromptAttackLevelBar(consumer)) {
			return false
		}

		// Check detailed results for backward compatibility
		for _, detail := range data.Detail {
			switch detail.Type {
			case ContentModerationType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetContentModerationLevelBar(consumer)) {
					return false
				}
			case PromptAttackType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetPromptAttackLevelBar(consumer)) {
					return false
				}
			case SensitiveDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetSensitiveDataLevelBar(consumer)) {
					return false
				}
			case MaliciousUrlDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetMaliciousUrlLevelBar(consumer)) {
					return false
				}
			case ModelHallucinationDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetModelHallucinationLevelBar(consumer)) {
					return false
				}
			}
		}
		return true
	} else {
		return LevelToInt(data.RiskLevel) < LevelToInt(config.GetRiskLevelBar(consumer))
	}
}
