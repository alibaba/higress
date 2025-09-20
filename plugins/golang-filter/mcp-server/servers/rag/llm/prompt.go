package llm

import (
	"strings"
)

const RAGPromptTemplate = `You are a professional knowledge Q&A assistant. Your task is to provide accurate, complete, and strictly relevant answers based on the user's question and retrieved context.

Retrieved relevant context (may be empty, multiple segments separated by line breaks):
{contexts}

User question:
{query}

Requirements:
1. If the context provides sufficient information, answer directly based on the context. You may use domain knowledge to supplement, but do not fabricate facts beyond the context.
2. If the context is insufficient or unrelated to the question, respond with: "I am unable to answer this question."
3. Your response must correctly answer the user's question and must not contain any irrelevant or unrelated content.`

func BuildPrompt(query string, contexts []string, join string) string {
	rendered := strings.ReplaceAll(RAGPromptTemplate, "{query}", query)
	rendered = strings.ReplaceAll(rendered, "{contexts}", strings.Join(contexts, join))
	return rendered
}
