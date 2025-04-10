package embedding

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	XFYUN_DOMAIN = "emb-cn-huabei-1.xf-yun.com"
	XFYUN_PORT   = 443
)

type xfyunProviderInitializer struct {
}

var xfyunConfig xfyunProviderConfig

type xfyunProviderConfig struct {
	// @Title zh-CN 文本特征提取服务 API Key
	// @Description zh-CN 文本特征提取服务 API Key。
	apiKey string
	// @Title zh-CN 文本特征提取服务 APPID
	// @Description zh-CN 文本特征提取服务 APPID。仅适用与 Xfyun
	xfyunAppID string
	// @Title zh-CN 文本特征提取服务 APISecret
	// @Description zh-CN 文本特征提取服务 APISecret。仅适用与 Xfyun
	xfyunApiSecret string
}

func (c *xfyunProviderInitializer) InitConfig(json gjson.Result) {
	xfyunConfig.xfyunAppID = json.Get("appId").String()
	xfyunConfig.xfyunApiSecret = json.Get("apiSecret").String()
	xfyunConfig.apiKey = json.Get("apiKey").String()
}

func (c *xfyunProviderInitializer) ValidateConfig() error {
	if xfyunConfig.apiKey == "" {
		return errors.New("[Xfyun] apiKey is required")
	}
	if xfyunConfig.xfyunAppID == "" {
		return errors.New("[Xfyun] appId is required")
	}
	if xfyunConfig.xfyunApiSecret == "" {
		return errors.New("[Xfyun] apiSecret is required")
	}
	return nil
}

func (t *xfyunProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = XFYUN_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = XFYUN_DOMAIN
	}

	return &XfyunProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: c.servicePort,
		}),
	}, nil
}

func (t *XfyunProvider) GetProviderType() string {
	return PROVIDER_TYPE_XFYUN
}

type XfyunProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

type XfyunHeader struct {
	AppID  string `json:"app_id"`
	Status int    `json:"status"`
}

type ReqFeature struct {
	Encoding string `json:"encoding"`
}

type XfyunEmb struct {
	Domain  string     `json:"domain"`
	Feature ReqFeature `json:"feature"`
}

type XfyunParameter struct {
	Emb XfyunEmb `json:"emb"`
}

type XfyunPayload struct {
	Messages struct {
		Text string `json:"text"`
	} `json:"messages"`
}

type XfyunText struct {
	MainMessages []struct {
		Content *string `json:"content"`
		Role    *string `json:"role"`
	} `json:"messages"`
}

type XfyunReqBody struct {
	Header    XfyunHeader    `json:"header"`
	Parameter XfyunParameter `json:"parameter"`
	Payload   XfyunPayload   `json:"payload"`
}

type XfyunResponse struct {
	Header  XfyunResHeader  `json:"header"`
	Payload XfyunResPayload `json:"payload"`
}

type XfyunResHeader struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Sid     string `json:"sid"`
}

type XfyunResPayload struct {
	Feature struct {
		Text string `json:"text"`
	} `json:"feature"`
}

func constructAuth(requestURL, method, apiKey, apiSecret string) (string, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Format(http.TimeFormat)
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\n%s %s HTTP/1.1", u.Host, now, method, u.Path)
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	authorizationOrigin := fmt.Sprintf("api_key=\"%s\", algorithm=\"hmac-sha256\", headers=\"host date request-line\", signature=\"%s\"", apiKey, signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	params := url.Values{}
	params.Add("host", u.Host)
	params.Add("date", now)
	params.Add("authorization", authorization)

	return "?" + params.Encode(), nil
}

func (t *XfyunProvider) constructParameters(text string) (string, [][2]string, []byte, error) {
	if text == "" {
		err := errors.New("queryString text cannot be empty")
		return "", nil, nil, err
	}

	host := "https://" + t.config.serviceHost + "/"
	auth, err := constructAuth(host, "POST", xfyunConfig.apiKey, xfyunConfig.xfyunApiSecret)
	if err != nil {
		return "", nil, nil, err
	}

	role := "user"

	xfyunText := XfyunText{
		MainMessages: []struct {
			Content *string `json:"content"`
			Role    *string `json:"role"`
		}{
			{
				Content: &text,
				Role:    &role,
			},
		},
	}

	// 将 XfyunText 转换为 JSON
	xfyunTextJSON, err := json.Marshal(xfyunText)
	if err != nil {
		log.Errorf("Error marshaling XfyunText: %v", err)
		return "", nil, nil, err
	}

	// 将整个 XfyunText JSON 字符串转换为 Base64 编码
	encodedText := base64.StdEncoding.EncodeToString(xfyunTextJSON)

	// 构建请求体
	data := XfyunReqBody{
		Header: XfyunHeader{
			AppID:  xfyunConfig.xfyunAppID,
			Status: 3,
		},
		Parameter: XfyunParameter{
			Emb: XfyunEmb{
				Domain: "query",
				Feature: ReqFeature{
					Encoding: "utf8",
				},
			},
		},
		Payload: XfyunPayload{
			Messages: struct {
				Text string `json:"text"`
			}{Text: encodedText}, // 填充经过 Base64 编码的文本
		},
	}

	// 序列化请求数据
	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	// 构建请求头
	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	return "/" + auth, headers, requestBody, nil
}

func (t *XfyunProvider) parseTextEmbedding(responseBody []byte) ([]float32, error) {
	var resp XfyunResponse
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}

	base64Text := resp.Payload.Feature.Text
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return nil, err
	}

	if len(decodedBytes) == 0 {
		return nil, errors.New("decoded embedding is empty")
	}

	if len(decodedBytes)%4 != 0 {
		return nil, errors.New("decoded data is not a valid float32 array")
	}

	floatArray := make([]float32, len(decodedBytes)/4)
	for i := 0; i < len(floatArray); i++ {
		bits := binary.LittleEndian.Uint32(decodedBytes[i*4 : (i+1)*4])
		floatArray[i] = math.Float32frombits(bits)
	}

	return floatArray, nil
}

func (t *XfyunProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters(queryString)
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	err = t.client.Post(embUrl, embHeaders, embRequestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {

			if statusCode != http.StatusOK {
				err = errors.New("failed to get embedding due to status code: " + strconv.Itoa(statusCode))
				callback(nil, err)
				return
			}

			var resp []float32
			resp, err = t.parseTextEmbedding(responseBody)
			if err != nil {
				err = fmt.Errorf("failed to parse response: %v", err)
				callback(nil, err)
				return
			}

			log.Debugf("get embedding response: %d, %s", statusCode, responseBody)

			if len(resp) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			embedding := make([]float64, len(resp))
			for i, v := range resp {
				embedding[i] = float64(v)
			}

			callback(embedding, nil)

		}, t.config.timeout)
	return err
}
