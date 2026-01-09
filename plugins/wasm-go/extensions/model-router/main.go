package main

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	DefaultMaxBodyBytes = 100 * 1024 * 1024 // 100MB
)

func main() {}

func init() {
	wrapper.SetCtx(
		"model-router",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.WithRebuildAfterRequests[ModelRouterConfig](1000),
		wrapper.WithRebuildMaxMemBytes[ModelRouterConfig](200*1024*1024),
	)
}

type ModelRouterConfig struct {
	modelKey           string
	addProviderHeader  string
	modelToHeader      string
	enableOnPathSuffix []string
}

func parseConfig(json gjson.Result, config *ModelRouterConfig) error {
	config.modelKey = json.Get("modelKey").String()
	if config.modelKey == "" {
		config.modelKey = "model"
	}
	config.addProviderHeader = json.Get("addProviderHeader").String()
	config.modelToHeader = json.Get("modelToHeader").String()

	enableOnPathSuffix := json.Get("enableOnPathSuffix")
	if enableOnPathSuffix.Exists() && enableOnPathSuffix.IsArray() {
		for _, item := range enableOnPathSuffix.Array() {
			config.enableOnPathSuffix = append(config.enableOnPathSuffix, item.String())
		}
	} else {
		// Default suffixes if not provided
		config.enableOnPathSuffix = []string{
			"/completions",
			"/embeddings",
			"/images/generations",
			"/audio/speech",
			"/fine_tuning/jobs",
			"/moderations",
			"/image-synthesis",
			"/video-synthesis",
		}
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ModelRouterConfig) types.Action {
	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		return types.ActionContinue
	}

	// Remove query parameters for suffix check
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	enable := false
	for _, suffix := range config.enableOnPathSuffix {
		if suffix == "*" || strings.HasSuffix(path, suffix) {
			enable = true
			break
		}
	}

	if !enable {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	contentType, err := proxywasm.GetHttpRequestHeader("content-type")
	if err != nil {
		return types.ActionContinue
	}

	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "multipart/form-data") {
		_ = proxywasm.RemoveHttpRequestHeader("content-length")
		// We need to buffer the body to parse it
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte) types.Action {
	contentType, err := proxywasm.GetHttpRequestHeader("content-type")
	if err != nil {
		return types.ActionContinue
	}

	if strings.Contains(contentType, "application/json") {
		return handleJsonBody(ctx, config, body)
	} else if strings.Contains(contentType, "multipart/form-data") {
		return handleMultipartBody(ctx, config, body, contentType)
	}

	return types.ActionContinue
}

func handleJsonBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte) types.Action {

	modelValue := gjson.GetBytes(body, config.modelKey).String()
	if modelValue == "" {
		return types.ActionContinue
	}

	if config.modelToHeader != "" {
		_ = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, modelValue)
	}

	if config.addProviderHeader != "" {
		parts := strings.SplitN(modelValue, "/", 2)
		if len(parts) == 2 {
			provider := parts[0]
			model := parts[1]
			_ = proxywasm.ReplaceHttpRequestHeader(config.addProviderHeader, provider)

			newBody, err := sjson.SetBytes(body, config.modelKey, model)
			if err != nil {
				log.Errorf("failed to update model in json body: %v", err)
				return types.ActionContinue
			}
			_ = proxywasm.ReplaceHttpRequestBody(newBody)
			log.Debugf("model route to provider: %s, model: %s", provider, model)
		} else {
			log.Debugf("model route to provider not work, model: %s", modelValue)
		}
	}

	return types.ActionContinue
}

func handleMultipartBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte, contentType string) types.Action {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Errorf("failed to parse content type: %v", err)
		return types.ActionContinue
	}
	boundary, ok := params["boundary"]
	if !ok {
		log.Errorf("no boundary in content type")
		return types.ActionContinue
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var newBody bytes.Buffer
	writer := multipart.NewWriter(&newBody)
	writer.SetBoundary(boundary)

	modified := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("failed to read multipart part: %v", err)
			return types.ActionContinue
		}

		// Read part content
		partContent, err := io.ReadAll(part)
		if err != nil {
			log.Errorf("failed to read part content: %v", err)
			return types.ActionContinue
		}

		formName := part.FormName()
		if formName == config.modelKey {
			modelValue := string(partContent)

			if config.modelToHeader != "" {
				_ = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, modelValue)
			}

			if config.addProviderHeader != "" {
				parts := strings.SplitN(modelValue, "/", 2)
				if len(parts) == 2 {
					provider := parts[0]
					model := parts[1]
					_ = proxywasm.ReplaceHttpRequestHeader(config.addProviderHeader, provider)

					// Write modified part
					h := make(http.Header)
					for k, v := range part.Header {
						h[k] = v
					}

					pw, err := writer.CreatePart(textproto.MIMEHeader(h))
					if err != nil {
						log.Errorf("failed to create part: %v", err)
						return types.ActionContinue
					}
					_, err = pw.Write([]byte(model))
					if err != nil {
						log.Errorf("failed to write part content: %v", err)
						return types.ActionContinue
					}
					modified = true
					log.Debugf("model route to provider: %s, model: %s", provider, model)
					continue
				} else {
					log.Debugf("model route to provider not work, model: %s", modelValue)
				}
			}
		}

		// Write original part
		h := make(http.Header)
		for k, v := range part.Header {
			h[k] = v
		}
		pw, err := writer.CreatePart(textproto.MIMEHeader(h))
		if err != nil {
			log.Errorf("failed to create part: %v", err)
			return types.ActionContinue
		}
		_, err = pw.Write(partContent)
		if err != nil {
			log.Errorf("failed to write part content: %v", err)
			return types.ActionContinue
		}
	}

	writer.Close()

	if modified {
		_ = proxywasm.ReplaceHttpRequestBody(newBody.Bytes())
	}

	return types.ActionContinue
}
