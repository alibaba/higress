package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"net/http"
	"strings"
	"time"
)

const (
	difyDomain         = "api.dify.ai"
	difyChatPath       = "/v1/chat-messages"
	difyCompletionPath = "/v1/completion-messages"
	difyWorkflowPath   = "/v1/workflows/run"
	BotTypeChat        = "Chat"
	BotTypeCompletion  = "Completion"
	BotTypeWorkflow    = "Workflow"
	BotTypeAgent       = "Agent"
)

type difyProviderInitializer struct{}

func (d *difyProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (d *difyProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &difyProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type difyProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (d *difyProvider) GetProviderType() string {
	return providerTypeDify
}

func (d *difyProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	d.config.handleRequestHeaders(d, ctx, apiName, log)
	return nil
}

func (d *difyProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	if d.config.difyApiUrl != "" {
		log.Debugf("use local host: %s", d.config.difyApiUrl)
		util.OverwriteRequestHostHeader(headers, d.config.difyApiUrl)
	} else {
		util.OverwriteRequestHostHeader(headers, difyDomain)
	}
	switch d.config.botType {
	case BotTypeChat, BotTypeAgent:
		util.OverwriteRequestPathHeader(headers, difyChatPath)
	case BotTypeCompletion:
		util.OverwriteRequestPathHeader(headers, difyCompletionPath)
	case BotTypeWorkflow:
		util.OverwriteRequestPathHeader(headers, difyWorkflowPath)
	}
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+d.config.GetApiTokenInUse(ctx))
}

func (d *difyProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return d.config.handleRequestBody(d, d.contextCache, ctx, apiName, body, log)
}

func (d *difyProvider) TransformRequestBodyHeaders(ctx wrapper.HttpContext, apiName ApiName, body []byte, headers http.Header, log wrapper.Log) ([]byte, error) {
	request := &chatCompletionRequest{}
	err := d.config.parseRequestAndMapModel(ctx, request, body, log)
	if err != nil {
		return nil, err
	}

	difyRequest := d.difyChatGenRequest(request)

	return json.Marshal(difyRequest)
}

func (d *difyProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) ([]byte, error) {
	difyResponse := &DifyChatResponse{}
	if err := json.Unmarshal(body, difyResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal dify response: %v", err)
	}
	response := d.responseDify2OpenAI(ctx, difyResponse)
	return json.Marshal(response)
}

func (d *difyProvider) responseDify2OpenAI(ctx wrapper.HttpContext, response *DifyChatResponse) *chatCompletionResponse {
	var choice chatCompletionChoice
	var id string
	switch d.config.botType {
	case BotTypeChat, BotTypeAgent:
		choice = chatCompletionChoice{
			Index:        0,
			Message:      &chatMessage{Role: roleAssistant, Content: response.Answer},
			FinishReason: finishReasonStop,
		}
		//response header中增加conversationId字段
		_ = proxywasm.ReplaceHttpResponseHeader("ConversationId", response.ConversationId)
		id = response.ConversationId
	case BotTypeCompletion:
		choice = chatCompletionChoice{
			Index:        0,
			Message:      &chatMessage{Role: roleAssistant, Content: response.Answer},
			FinishReason: finishReasonStop,
		}
		id = response.MessageId
	case BotTypeWorkflow:
		choice = chatCompletionChoice{
			Index:        0,
			Message:      &chatMessage{Role: roleAssistant, Content: response.Data.Outputs[d.config.outputVariable]},
			FinishReason: finishReasonStop,
		}
		id = response.Data.WorkflowId
	}
	return &chatCompletionResponse{
		Id:                id,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           []chatCompletionChoice{choice},
		Usage:             response.MetaData.Usage,
	}
}

func (d *difyProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	// sample event response:
	// data: {"event": "agent_thought", "id": "8dcf3648-fbad-407a-85dd-73a6f43aeb9f", "task_id": "9cf1ddd7-f94b-459b-b942-b77b26c59e9b", "message_id": "1fb10045-55fd-4040-99e6-d048d07cbad3", "position": 1, "thought": "", "observation": "", "tool": "", "tool_input": "", "created_at": 1705639511, "message_files": [], "conversation_id": "c216c595-2d89-438c-b33c-aae5ddddd142"}

	// sample end event response:
	// data: {"event": "message_end", "id": "5e52ce04-874b-4d27-9045-b3bc80def685", "conversation_id": "45701982-8118-4bc5-8e9b-64562b4555f2", "metadata": {"usage": {"prompt_tokens": 1033, "prompt_unit_price": "0.001", "prompt_price_unit": "0.001", "prompt_price": "0.0010330", "completion_tokens": 135, "completion_unit_price": "0.002", "completion_price_unit": "0.001", "completion_price": "0.0002700", "total_tokens": 1168, "total_price": "0.0013030", "currency": "USD", "latency": 1.381760165997548}, "retriever_resources": [{"position": 1, "dataset_id": "101b4c97-fc2e-463c-90b1-5261a4cdcafb", "dataset_name": "iPhone", "document_id": "8dd1ad74-0b5f-4175-b735-7d98bbbb4e00", "document_name": "iPhone List", "segment_id": "ed599c7f-2766-4294-9d1d-e5235a61270a", "score": 0.98457545, "content": "\"Model\",\"Release Date\",\"Display Size\",\"Resolution\",\"Processor\",\"RAM\",\"Storage\",\"Camera\",\"Battery\",\"Operating System\"\n\"iPhone 13 Pro Max\",\"September 24, 2021\",\"6.7 inch\",\"1284 x 2778\",\"Hexa-core (2x3.23 GHz Avalanche + 4x1.82 GHz Blizzard)\",\"6 GB\",\"128, 256, 512 GB, 1TB\",\"12 MP\",\"4352 mAh\",\"iOS 15\""}]}}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		var difyResponse DifyChunkChatResponse
		if err := json.Unmarshal([]byte(data), &difyResponse); err != nil {
			log.Errorf("unable to unmarshal dify response: %v", err)
			continue
		}
		response := d.streamResponseDify2OpenAI(ctx, &difyResponse)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		d.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (d *difyProvider) streamResponseDify2OpenAI(ctx wrapper.HttpContext, response *DifyChunkChatResponse) *chatCompletionResponse {
	var choice chatCompletionChoice
	var id string
	switch d.config.botType {
	case BotTypeChat, BotTypeAgent:
		choice = chatCompletionChoice{
			Index: 0,
			Delta: &chatMessage{Role: roleAssistant, Content: response.Answer},
		}
		id = response.ConversationId
		_ = proxywasm.ReplaceHttpResponseHeader("ConversationId", response.ConversationId)
	case BotTypeCompletion:
		choice = chatCompletionChoice{
			Index: 0,
			Delta: &chatMessage{Role: roleAssistant, Content: response.Answer},
		}
		id = response.MessageId
	case BotTypeWorkflow:
		choice = chatCompletionChoice{
			Index: 0,
			Delta: &chatMessage{Role: roleAssistant, Content: response.Data.Outputs[d.config.outputVariable]},
		}
		id = response.Data.WorkflowId
	}
	if response.Event == "message_end" || response.Event == "workflow_finished" {
		choice.FinishReason = finishReasonStop
	}
	return &chatCompletionResponse{
		Id:                id,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletionChunk,
		Choices:           []chatCompletionChoice{choice},
	}
}

func (d *difyProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}

func (d *difyProvider) difyChatGenRequest(request *chatCompletionRequest) *DifyChatRequest {
	content := ""
	for _, message := range request.Messages {
		if message.Role == "system" {
			content += "SYSTEM: \n" + message.StringContent() + "\n"
		} else if message.Role == "assistant" {
			content += "ASSISTANT: \n" + message.StringContent() + "\n"
		} else {
			content += "USER: \n" + message.StringContent() + "\n"
		}
	}
	mode := "blocking"
	if request.Stream {
		mode = "streaming"
	}
	user := request.User
	if user == "" {
		user = "api-user"
	}
	switch d.config.botType {
	case BotTypeChat, BotTypeAgent:
		conversationId, _ := proxywasm.GetHttpRequestHeader("ConversationId")
		return &DifyChatRequest{
			Inputs:           make(map[string]interface{}),
			Query:            content,
			ResponseMode:     mode,
			User:             user,
			AutoGenerateName: false,
			ConversationId:   conversationId,
		}
	case BotTypeCompletion:
		return &DifyChatRequest{
			Inputs: map[string]interface{}{
				"query": content,
			},
			ResponseMode: mode,
			User:         user,
		}
	case BotTypeWorkflow:
		return &DifyChatRequest{
			Inputs: map[string]interface{}{
				d.config.inputVariable: content,
			},
			ResponseMode: mode,
			User:         user,
		}
	default:
		return &DifyChatRequest{}
	}
}

type DifyChatRequest struct {
	Inputs           map[string]interface{} `json:"inputs"`
	Query            string                 `json:"query"`
	ResponseMode     string                 `json:"response_mode"`
	User             string                 `json:"user"`
	AutoGenerateName bool                   `json:"auto_generate_name"`
	ConversationId   string                 `json:"conversation_id"`
}

type DifyMetaData struct {
	Usage usage `json:"usage"`
}

type DifyData struct {
	WorkflowId string                 `json:"workflow_id"`
	Id         string                 `json:"id"`
	Outputs    map[string]interface{} `json:"outputs"`
}

type DifyChatResponse struct {
	ConversationId string       `json:"conversation_id"`
	MessageId      string       `json:"message_id"`
	Answer         string       `json:"answer"`
	CreateAt       int64        `json:"create_at"`
	Data           DifyData     `json:"data"`
	MetaData       DifyMetaData `json:"metadata"`
}

type DifyChunkChatResponse struct {
	Event          string       `json:"event"`
	ConversationId string       `json:"conversation_id"`
	MessageId      string       `json:"message_id"`
	Answer         string       `json:"answer"`
	Data           DifyData     `json:"data"`
	MetaData       DifyMetaData `json:"metadata"`
}
