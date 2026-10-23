package llm

import (
	"strings"
)

const RAGPromptTemplate = `You are a professional knowledge Q&A assistant. Your task is to provide direct and concise answers based on the user's question and retrieved context.

Retrieved relevant context (may be empty, multiple segments separated by line breaks):
{contexts}

User question:
{query}

Requirements:
1. Provide ONLY the direct answer without any explanation, reasoning, or additional context.
2. If the context provides sufficient information, output the answer in the most concise form possible.
3. If the context is insufficient or unrelated to the question, respond with: "I am unable to answer this question."
4. Do not include any phrases like "The answer is", "Based on the context", etc. Just output the answer directly.
`

func BuildPrompt(query string, contexts []string, join string) string {
	rendered := strings.ReplaceAll(RAGPromptTemplate, "{query}", query)
	rendered = strings.ReplaceAll(rendered, "{contexts}", strings.Join(contexts, join))
	return rendered
}
