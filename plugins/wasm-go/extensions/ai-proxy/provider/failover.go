package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type failover struct {
	// @Title zh-CN 是否启用 apiToken 的 failover 机制
	enabled bool `required:"true" yaml:"enabled" json:"enabled"`
	// @Title zh-CN 触发 failover 的失败阈值
	failureThreshold int64 `required:"false" yaml:"failureThreshold" json:"failureThreshold"`
	// @Title zh-CN 健康检测的成功阈值
	successThreshold int64 `required:"false" yaml:"successThreshold" json:"successThreshold"`
	// @Title zh-CN 健康检测的间隔时间，单位毫秒
	healthCheckInterval int64 `required:"false" yaml:"healthCheckInterval" json:"healthCheckInterval"`
	// @Title zh-CN 健康检测的超时时间，单位毫秒
	healthCheckTimeout int64 `required:"false" yaml:"healthCheckTimeout" json:"healthCheckTimeout"`
	// @Title zh-CN 健康检测使用的模型
	healthCheckModel string `required:"true" yaml:"healthCheckModel" json:"healthCheckModel"`
}

var (
	ApiTokens                   []string
	ApiTokenRequestFailureCount = make(map[string]int64)
	ApiTokenRequestSuccessCount = make(map[string]int64)
	healthCheckClient           wrapper.HttpClient
	UnavailableApiTokens        []string
)

const (
	ApiTokenInUse       = "apiTokenInUse"
	ApiTokenHealthCheck = "apiTokenHealthCheck"
)

func (f *failover) FromJson(json gjson.Result) {
	f.enabled = json.Get("enabled").Bool()
	f.failureThreshold = json.Get("failureThreshold").Int()
	if f.failureThreshold == 0 {
		f.failureThreshold = 3
	}
	f.successThreshold = json.Get("successThreshold").Int()
	if f.successThreshold == 0 {
		f.successThreshold = 1
	}
	f.healthCheckInterval = json.Get("healthCheckInterval").Int()
	if f.healthCheckInterval == 0 {
		f.healthCheckInterval = 5000
	}
	f.healthCheckTimeout = json.Get("healthCheckTimeout").Int()
	if f.healthCheckTimeout == 0 {
		f.healthCheckTimeout = 5000
	}
	f.healthCheckModel = json.Get("healthCheckModel").String()
}

func (f *failover) Validate() error {
	if f.healthCheckModel == "" {
		return errors.New("missing healthCheckModel in failover config")
	}
	return nil
}

func (c *ProviderConfig) SetApiTokensFailover(log wrapper.Log) {
	ApiTokens = c.apiTokens
	// TODO: 目前需要手动加一个 cluster 指向本地的地址，健康检测需要访问该地址
	healthCheckClient = wrapper.NewClusterClient(wrapper.StaticIpCluster{
		ServiceName: "local_cluster",
		Port:        10000,
	})

	if c.failover != nil && c.failover.enabled {
		wrapper.RegisteTickFunc(c.failover.healthCheckTimeout, func() {
			if len(UnavailableApiTokens) > 0 {
				for _, apiToken := range UnavailableApiTokens {
					log.Debugf("Perform health check for unavailable apiTokens: %s", strings.Join(UnavailableApiTokens, ", "))

					path := "/v1/chat/completions"
					headers := [][2]string{
						{"Content-Type", "application/json"},
						{"ApiToken-Health-Check", apiToken},
					}
					body := []byte(fmt.Sprintf(`{
                      "model": "%s",
                      "messages": [
                        {
                          "role": "user",
                          "content": "who are you?"
                        }
                      ]
                    }`, c.failover.healthCheckModel))
					err := healthCheckClient.Post(path, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
						if statusCode == 200 {
							c.HandleAvailableApiToken(apiToken, log)
						}
					}, uint32(c.failover.healthCheckTimeout))
					if err != nil {
						log.Errorf("Failed to perform health check request: %v", err)
					}
				}
			}
		})
	}
}

func (c *ProviderConfig) HandleAvailableApiToken(apiToken string, log wrapper.Log) {
	ApiTokenRequestSuccessCount[apiToken]++
	if ApiTokenRequestSuccessCount[apiToken] >= c.failover.successThreshold {
		log.Infof("apiToken %s is available now, add it back to the list", apiToken)
		c.RemoveToken(&UnavailableApiTokens, apiToken)
		c.AddToken(&ApiTokens, apiToken)
		ApiTokenRequestSuccessCount[apiToken] = 0
	}
}

func (c *ProviderConfig) HandleUnavailableApiToken(apiToken string, log wrapper.Log) {
	ApiTokenRequestFailureCount[apiToken]++
	if ApiTokenRequestFailureCount[apiToken] >= c.failover.failureThreshold {
		log.Errorf("Remove unavailable apiToken from list: %s", apiToken)
		c.RemoveToken(&ApiTokens, apiToken)
		c.AddToken(&UnavailableApiTokens, apiToken)
		ApiTokenRequestFailureCount[apiToken] = 0
	}
}

func (c *ProviderConfig) RemoveToken(tokens *[]string, apiToken string) {
	tmp := make([]string, 0)
	for _, v := range *tokens {
		if v != apiToken {
			tmp = append(tmp, v)
		}
	}
	*tokens = tmp
}

func (c *ProviderConfig) AddToken(tokens *[]string, apiToken string) {
	if !contains(*tokens, apiToken) {
		*tokens = append(*tokens, apiToken)
	}
}

func contains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}
