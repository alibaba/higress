package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
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

type Lease struct {
	VMID      string `json:"vmID"`
	Timestamp int64  `json:"timestamp"`
}

var (
	healthCheckClient wrapper.HttpClient
)

const (
	ApiTokenInUse                      = "apiTokenInUse"
	ApiTokenHealthCheck                = "apiTokenHealthCheck"
	vmLease                            = "vmLease"
	CtxApiTokenRequestFailureCount     = "apiTokenRequestFailureCount"
	ctxApiTokenRequestSuccessCount     = "apiTokenRequestSuccessCount"
	ctxApiTokens                       = "apiTokens"
	ctxUnavailableApiTokens            = "unavailableApiTokens"
	casMaxRetries                      = 10
	addApiTokenOperation               = "addApiToken"
	removeApiTokenOperation            = "removeApiToken"
	addApiTokenRequestCountOperation   = "addApiTokenRequestCount"
	resetApiTokenRequestCountOperation = "ResetApiTokenRequestCount"
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
	// Reset shared data in case plugin configuration is updated
	resetSharedData()

	// TODO: 目前需要手动加一个 cluster 指向本地的地址，健康检测需要访问该地址
	healthCheckClient = wrapper.NewClusterClient(wrapper.StaticIpCluster{
		ServiceName: "local_cluster",
		Port:        10000,
	})

	vmID := generateVMID()
	err := c.initApiTokens()
	if err != nil {
		log.Errorf("Failed to init apiTokens: %v", err)
	}

	if c.failover != nil && c.failover.enabled {
		wrapper.RegisteTickFunc(c.failover.healthCheckInterval, func() {
			// Only the Wasm VM that successfully acquires the lease will perform health check
			if tryAcquireOrRenewLease(vmID, log) {
				log.Debugf("Successfully acquired or renewed lease: %s", vmID)
				unavailableTokens, _, err := getApiTokens(ctxUnavailableApiTokens)
				if err != nil {
					log.Errorf("Failed to get unavailable tokens: %v", err)
					return
				}
				if len(unavailableTokens) > 0 {
					for _, apiToken := range unavailableTokens {
						log.Debugf("Perform health check for unavailable apiTokens: %s", strings.Join(unavailableTokens, ", "))

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
			}
		})
	}
}

func tryAcquireOrRenewLease(vmID string, log wrapper.Log) bool {
	now := time.Now().Unix()

	data, cas, err := proxywasm.GetSharedData(vmLease)
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return setLease(vmID, now, cas, log)
		} else {
			log.Errorf("Failed to get lease: %v", err)
			return false
		}
	}
	if data == nil {
		return setLease(vmID, now, cas, log)
	}

	var lease Lease
	err = json.Unmarshal(data, &lease)
	if err != nil {
		log.Errorf("Failed to unmarshal lease data: %v", err)
		return false
	}
	// If vmID is itself, try to renew the lease directly
	// If the lease is expired (60s), try to acquire the lease
	if lease.VMID == vmID || now-lease.Timestamp > 60 {
		lease.VMID = vmID
		lease.Timestamp = now
		return setLease(vmID, now, cas, log)
	}

	return false
}

func setLease(vmID string, timestamp int64, cas uint32, log wrapper.Log) bool {
	lease := Lease{
		VMID:      vmID,
		Timestamp: timestamp,
	}
	leaseByte, err := json.Marshal(lease)
	if err != nil {
		log.Errorf("Failed to marshal lease data: %v", err)
		return false
	}

	if err := proxywasm.SetSharedData(vmLease, leaseByte, cas); err != nil {
		log.Errorf("Failed to set or renew lease: %v", err)
		return false
	}
	return true
}

func generateVMID() string {
	return uuid.New().String()
}

// When number of request successes exceeds the threshold during health check,
// add the apiToken back to the available list and remove it from the unavailable list
func (c *ProviderConfig) HandleAvailableApiToken(apiToken string, log wrapper.Log) {
	successApiTokenRequestCount, _, err := GetApiTokenRequestCount(ctxApiTokenRequestSuccessCount)
	if err != nil {
		log.Errorf("Failed to get successApiTokenRequestCount: %v", err)
		return
	}

	successCount := successApiTokenRequestCount[apiToken] + 1
	if successCount >= c.failover.successThreshold {
		log.Infof("apiToken %s is available now, add it back to the apiTokens list", apiToken)
		removeApiToken(ctxUnavailableApiTokens, apiToken, log)
		addApiToken(ctxApiTokens, apiToken, log)
		ResetApiTokenRequestCount(ctxApiTokenRequestSuccessCount, apiToken, log)
	} else {
		log.Debugf("apiToken %s is still unavailable, the number of health check passed: %d, continue to health check......", apiToken, successCount)
		addApiTokenRequestCount(ctxApiTokenRequestSuccessCount, apiToken, log)
	}
}

// When number of request failures exceeds the threshold,
// remove the apiToken from the available list and add it to the unavailable list
func (c *ProviderConfig) HandleUnavailableApiToken(apiToken string, log wrapper.Log) {
	failureApiTokenRequestCount, _, err := GetApiTokenRequestCount(CtxApiTokenRequestFailureCount)
	if err != nil {
		log.Errorf("Failed to get failureApiTokenRequestCount: %v", err)
		return
	}

	availableTokens, _, err := getApiTokens(ctxApiTokens)
	if err != nil {
		log.Errorf("Failed to get available apiToken: %v", err)
		return
	}
	// unavailable apiToken has been removed from the available list
	if !containsElement(availableTokens, apiToken) {
		return
	}

	failureCount := failureApiTokenRequestCount[apiToken] + 1
	if failureCount >= c.failover.failureThreshold {
		log.Infof("apiToken %s is unavailable now, remove it from apiTokens list", apiToken)
		removeApiToken(ctxApiTokens, apiToken, log)
		addApiToken(ctxUnavailableApiTokens, apiToken, log)
		ResetApiTokenRequestCount(CtxApiTokenRequestFailureCount, apiToken, log)
	} else {
		log.Debugf("apiToken %s is still available as it has not reached the failure threshold, the number of failed request: %d", apiToken, failureCount)
		addApiTokenRequestCount(CtxApiTokenRequestFailureCount, apiToken, log)
	}
}

func addApiToken(key, apiToken string, log wrapper.Log) {
	modifyApiToken(key, apiToken, addApiTokenOperation, log)
}

func removeApiToken(key, apiToken string, log wrapper.Log) {
	modifyApiToken(key, apiToken, removeApiTokenOperation, log)
}

func modifyApiToken(key, apiToken, op string, log wrapper.Log) {
	for attempt := 1; attempt <= casMaxRetries; attempt++ {
		apiTokens, cas, err := getApiTokens(key)
		if err != nil {
			log.Errorf("Failed to get %s: %v", key, err)
			continue
		}

		exists := containsElement(apiTokens, apiToken)
		if op == addApiTokenOperation && exists {
			log.Debugf("%s already exists in %s", apiToken, key)
			return
		} else if op == removeApiTokenOperation && !exists {
			log.Debugf("%s does not exist in %s", apiToken, key)
			return
		}

		if op == addApiTokenOperation {
			apiTokens = append(apiTokens, apiToken)
		} else {
			apiTokens = removeElement(apiTokens, apiToken)
		}

		if err := setApiTokens(key, apiTokens, cas); err == nil {
			log.Debugf("Successfully updated %s in %s", apiToken, key)
			return
		} else if !errors.Is(err, types.ErrorStatusCasMismatch) {
			log.Errorf("Failed to set %s after %d attempts: %v", key, attempt, err)
			return
		}

		log.Errorf("CAS mismatch when setting %s, retrying...", key)
	}
}

func getApiTokens(key string) ([]string, uint32, error) {
	data, cas, err := proxywasm.GetSharedData(key)
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return []string{}, cas, nil
		}
		return nil, 0, err
	}
	if data == nil {
		return []string{}, cas, nil
	}

	var apiTokens []string
	if err = json.Unmarshal(data, &apiTokens); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal tokens: %v", err)
	}

	return apiTokens, cas, nil
}

func setApiTokens(key string, apiTokens []string, cas uint32) error {
	data, err := json.Marshal(apiTokens)
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %v", err)
	}
	return proxywasm.SetSharedData(key, data, cas)
}

func removeElement(slice []string, s string) []string {
	for i := 0; i < len(slice); i++ {
		if slice[i] == s {
			slice = append(slice[:i], slice[i+1:]...)
			i--
		}
	}
	return slice
}

func containsElement(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func GetApiTokenRequestCount(key string) (map[string]int64, uint32, error) {
	data, cas, err := proxywasm.GetSharedData(key)
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return make(map[string]int64), cas, nil
		}
		return nil, 0, err
	}

	if data == nil {
		return make(map[string]int64), cas, nil
	}

	var apiTokens map[string]int64
	err = json.Unmarshal(data, &apiTokens)
	if err != nil {
		return nil, 0, err
	}
	return apiTokens, cas, nil
}

func addApiTokenRequestCount(key, apiToken string, log wrapper.Log) {
	modifyApiTokenRequestCount(key, apiToken, addApiTokenRequestCountOperation, log)
}

func ResetApiTokenRequestCount(key, apiToken string, log wrapper.Log) {
	modifyApiTokenRequestCount(key, apiToken, resetApiTokenRequestCountOperation, log)
}

func modifyApiTokenRequestCount(key, apiToken string, op string, log wrapper.Log) {
	for attempt := 1; attempt <= casMaxRetries; attempt++ {
		apiTokenRequestCount, cas, err := GetApiTokenRequestCount(key)
		if err != nil {
			log.Errorf("Failed to get %s: %v", key, err)
			continue
		}

		if op == resetApiTokenRequestCountOperation {
			delete(apiTokenRequestCount, apiToken)
		} else {
			apiTokenRequestCount[apiToken]++
		}

		apiTokenRequestCountByte, err := json.Marshal(apiTokenRequestCount)
		if err != nil {
			log.Errorf("failed to marshal apiTokenRequestCount: %v", err)
		}

		if err := proxywasm.SetSharedData(key, apiTokenRequestCountByte, cas); err == nil {
			log.Debugf("Successfully updated the count of %s in %s", apiToken, key)
			return
		} else if !errors.Is(err, types.ErrorStatusCasMismatch) {
			log.Errorf("Failed to set %s after %d attempts: %v", key, attempt, err)
			return
		}

		log.Errorf("CAS mismatch when setting %s, retrying...", key)
	}
}

func (c *ProviderConfig) initApiTokens() error {
	return setApiTokens(ctxApiTokens, c.apiTokens, 0)
}

func (c *ProviderConfig) GetGlobalRandomToken(log wrapper.Log) string {
	apiTokens, _, err := getApiTokens(ctxApiTokens)
	unavailableApiTokens, _, err := getApiTokens(ctxUnavailableApiTokens)
	log.Debugf("apiTokens: %v, unavailableApiTokens: %v", apiTokens, unavailableApiTokens)

	if err != nil {
		return ""
	}
	count := len(apiTokens)
	switch count {
	case 0:
		return ""
	case 1:
		return apiTokens[0]
	default:
		return apiTokens[rand.Intn(count)]
	}
}

func getApiTokenInUse(ctx wrapper.HttpContext) string {
	return ctx.GetContext(ApiTokenInUse).(string)
}

func (c *ProviderConfig) IsFailoverEnabled() bool {
	return c.failover != nil && c.failover.enabled
}

func resetSharedData() {
	_ = proxywasm.SetSharedData(vmLease, nil, 0)
	_ = proxywasm.SetSharedData(ctxApiTokens, nil, 0)
	_ = proxywasm.SetSharedData(ctxUnavailableApiTokens, nil, 0)
	_ = proxywasm.SetSharedData(ctxApiTokenRequestSuccessCount, nil, 0)
	_ = proxywasm.SetSharedData(CtxApiTokenRequestFailureCount, nil, 0)
}
