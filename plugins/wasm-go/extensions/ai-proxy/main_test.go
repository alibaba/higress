package main

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/provider"
)

func Test_getApiName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want provider.ApiName
	}{
		// OpenAI style
		{"openai chat completions", "/v1/chat/completions", provider.ApiNameChatCompletion},
		{"openai completions", "/v1/completions", provider.ApiNameCompletion},
		{"openai embeddings", "/v1/embeddings", provider.ApiNameEmbeddings},
		{"openai audio speech", "/v1/audio/speech", provider.ApiNameAudioSpeech},
		{"openai image generation", "/v1/images/generations", provider.ApiNameImageGeneration},
		{"openai image variation", "/v1/images/variations", provider.ApiNameImageVariation},
		{"openai image edit", "/v1/images/edits", provider.ApiNameImageEdit},
		{"openai batches", "/v1/batches", provider.ApiNameBatches},
		{"openai retrieve batch", "/v1/batches/batchid", provider.ApiNameRetrieveBatch},
		{"openai cancel batch", "/v1/batches/batchid/cancel", provider.ApiNameCancelBatch},
		{"openai files", "/v1/files", provider.ApiNameFiles},
		{"openai retrieve file", "/v1/files/fileid", provider.ApiNameRetrieveFile},
		{"openai retrieve file content", "/v1/files/fileid/content", provider.ApiNameRetrieveFileContent},
		{"openai models", "/v1/models", provider.ApiNameModels},
		{"openai fine tuning jobs", "/v1/fine_tuning/jobs", provider.ApiNameFineTuningJobs},
		{"openai retrieve fine tuning job", "/v1/fine_tuning/jobs/jobid", provider.ApiNameRetrieveFineTuningJob},
		{"openai fine tuning job events", "/v1/fine_tuning/jobs/jobid/events", provider.ApiNameFineTuningJobEvents},
		{"openai fine tuning job checkpoints", "/v1/fine_tuning/jobs/jobid/checkpoints", provider.ApiNameFineTuningJobCheckpoints},
		{"openai cancel fine tuning job", "/v1/fine_tuning/jobs/jobid/cancel", provider.ApiNameCancelFineTuningJob},
		{"openai resume fine tuning job", "/v1/fine_tuning/jobs/jobid/resume", provider.ApiNameResumeFineTuningJob},
		{"openai pause fine tuning job", "/v1/fine_tuning/jobs/jobid/pause", provider.ApiNamePauseFineTuningJob},
		{"openai fine tuning checkpoint permissions", "/v1/fine_tuning/checkpoints/checkpointid/permissions", provider.ApiNameFineTuningCheckpointPermissions},
		{"openai delete fine tuning checkpoint permission", "/v1/fine_tuning/checkpoints/checkpointid/permissions/permissionid", provider.ApiNameDeleteFineTuningCheckpointPermission},
		{"openai responses", "/v1/responses", provider.ApiNameResponses},
		// Anthropic
		{"anthropic messages", "/v1/messages", provider.ApiNameAnthropicMessages},
		{"anthropic complete", "/v1/complete", provider.ApiNameAnthropicComplete},
		// Gemini
		{"gemini generate content", "/v1beta/models/gemini-1.0-pro:generateContent", provider.ApiNameGeminiGenerateContent},
		{"gemini stream generate content", "/v1beta/models/gemini-1.0-pro:streamGenerateContent", provider.ApiNameGeminiStreamGenerateContent},
		// Cohere
		{"cohere rerank", "/v1/rerank", provider.ApiNameCohereV1Rerank},
		// Unknown
		{"unknown", "/v1/unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getApiName(tt.path)
			if got != tt.want {
				t.Errorf("getApiName(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
