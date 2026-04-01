package text

import (
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func HandleTextGenerationRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	if config.CheckAllMessages && config.RedisClient != nil {
		return handleRequestWithDedup(ctx, config, body)
	}
	return handleDefaultRequest(ctx, config, body)
}

func handleDefaultRequest(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	startTime := time.Now().UnixMilli()
	content := gjson.GetBytes(body, config.RequestContentJsonPath).String()
	log.Debugf("Raw request content is: %s", content)
	if len(content) == 0 {
		log.Info("request content is empty. skip")
		return types.ActionContinue
	}
	textCheckFn := func(contentPiece, sessionID string) (string, [][2]string, []byte) {
		checkService := config.GetRequestCheckService(consumer)
		return common.GenerateRequestForText(config, cfg.TextModerationPlus, checkService, contentPiece, sessionID)
	}
	common.RunChunkedTextCheck(ctx, config, body, content, startTime, consumer, textCheckFn, func() {
		proxywasm.ResumeHttpRequest()
	})
	return types.ActionPause
}

func handleRequestWithDedup(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	startTime := time.Now().UnixMilli()
	policyFingerprint := config.BuildPolicyFingerprint(consumer)

	allMessages := utils.ParseAllMessages(body)
	messages := utils.FilterByRole(allMessages, "system", "user")
	log.Infof("[dedup] %d messages after role filter (system/user only), %d total", len(messages), len(allMessages))
	if len(messages) == 0 {
		log.Info("no messages to check after role filter, skip")
		return types.ActionContinue
	}

	keys := utils.BuildRedisKeys(messages, consumer, policyFingerprint)
	err := config.RedisClient.MGet(keys, func(redisResponse resp.Value) {
		unchecked := utils.FilterUnchecked(messages, redisResponse)
		log.Infof("[dedup] total=%d, unchecked=%d, cached=%d", len(messages), len(unchecked), len(messages)-len(unchecked))
		if len(unchecked) == 0 {
			log.Info("all messages already checked, skip security check")
			proxywasm.ResumeHttpRequest()
			return
		}

		content := utils.ConcatTextContent(unchecked)
		if len(content) == 0 {
			log.Info("no text content in unchecked messages, marking as checked")
			utils.MarkChecked(config.RedisClient, unchecked, consumer, policyFingerprint, config.CheckRecordTTL, func() {
				proxywasm.ResumeHttpRequest()
			})
			return
		}

		textCheckFn := func(contentPiece, sessionID string) (string, [][2]string, []byte) {
			checkService := config.GetRequestCheckService(consumer)
			return common.GenerateRequestForText(config, cfg.TextModerationPlus, checkService, contentPiece, sessionID)
		}
		common.RunChunkedTextCheck(ctx, config, body, content, startTime, consumer, textCheckFn, func() {
			utils.MarkChecked(config.RedisClient, unchecked, consumer, policyFingerprint, config.CheckRecordTTL, func() {
				proxywasm.ResumeHttpRequest()
			})
		})
	})
	if err != nil {
		log.Warnf("redis MGet failed: %v, fallback to default check", err)
		return handleDefaultRequest(ctx, config, body)
	}
	return types.ActionPause
}
