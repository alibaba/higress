package textsplitter

import (
	"strings"
	"testing"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:dupword,funlen
func TestChineseRecursiveCharacterSplitter(t *testing.T) {
	// t.Parallel()
	type testCase struct {
		name          string
		text          string
		chunkOverlap  int
		chunkSize     int
		separators    []string
		expected      []schema.Document
		keepSeparator bool
		lenFunc       func(string) int
	}
	testCases := []testCase{
		{
			name:         "Basic Chinese text splitting",
			text:         "第一段文字\n第二段文字\n第三段文字",
			chunkSize:    10,
			chunkOverlap: 2,
			expected: []schema.Document{
				{Content: "第一段文字\n第二段文字", Metadata: map[string]any{}},
				{Content: "第三段文字", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Chinese text with punctuation",
			text:         "这是第一句。这是第二句！这是第三句？",
			chunkSize:    15,
			chunkOverlap: 3,
			expected: []schema.Document{
				{Content: "这是第一句。这是第二句！", Metadata: map[string]any{}},
				{Content: "这是第三句？", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Mixed Chinese and English",
			text:         "中文text混合。English and 中文。",
			chunkSize:    25,
			chunkOverlap: 2,
			expected: []schema.Document{
				{Content: "中文text混合。English and 中文。", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Long text requiring splitting",
			text:         "这是一个很长的中文段落，需要被分割成多个部分。这里有更多的内容，应该会被分割。最后一部分内容。",
			chunkSize:    30,
			chunkOverlap: 5,
			expected: []schema.Document{
				{Content: "这是一个很长的中文段落，需要被分割成多个部分。这里有更多的内容，应该会被分割。", Metadata: map[string]any{}},
				{Content: "最后一部分内容。", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Text with commas",
			text:         "第一部分，第二部分，第三部分",
			chunkSize:    12,
			chunkOverlap: 1,
			expected: []schema.Document{
				{Content: "第一部分，第二部分，", Metadata: map[string]any{}},
				{Content: "第三部分", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Text with semicolons",
			text:         "项目一；项目二；项目三",
			chunkSize:    10,
			chunkOverlap: 1,
			expected: []schema.Document{
				{Content: "项目一；项目二；", Metadata: map[string]any{}},
				{Content: "项目三", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Numbers and Chinese characters",
			text:         "第1章内容。第2章内容。第3章内容。",
			chunkSize:    15,
			chunkOverlap: 2,
			expected: []schema.Document{
				{Content: "第1章内容。第2章内容。", Metadata: map[string]any{}},
				{Content: "第3章内容。", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Custom separators test",
			text:         "你好世界。这是一个测试。欢迎使用！",
			chunkOverlap: 0,
			chunkSize:    10,
			separators:   []string{"。|！|？", "，|,\\s", ""},
			expected: []schema.Document{
				{Content: "你好世界", Metadata: map[string]any{}},
				{Content: "这是一个测试", Metadata: map[string]any{}},
				{Content: "欢迎使用", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Double newline separator test",
			text:         "第一段文字\n\n第二段文字\n\n第三段文字",
			chunkOverlap: 1,
			chunkSize:    15,
			separators:   []string{"\n\n", "\n", "。|！|？"},
			expected: []schema.Document{
				{Content: "第一段文字", Metadata: map[string]any{}},
				{Content: "第二段文字", Metadata: map[string]any{}},
				{Content: "第三段文字", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Complex punctuation test",
			text:         "这是一个很长的句子，包含了很多中文字符；它需要被正确地分割成多个部分。",
			chunkOverlap: 2,
			chunkSize:    20,
			separators:   []string{"。|！|？", "；|;\\s", "，|,\\s", ""},
			expected: []schema.Document{
				{Content: "这是一个很长的句子", Metadata: map[string]any{}},
				{Content: "包含了很多中文字符", Metadata: map[string]any{}},
				{Content: "它需要被正确地分割成多个部分", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Mixed language test",
			text:         "Hello world! 你好世界！How are you? 你好吗？",
			chunkOverlap: 0,
			chunkSize:    25,
			separators:   []string{"\\.\\s|\\!\\s|\\?\\s", "。|！|？", ""},
			expected: []schema.Document{
				{Content: "Hello world", Metadata: map[string]any{}},
				{Content: "你好世界", Metadata: map[string]any{}},
				{Content: "How are you", Metadata: map[string]any{}},
				{Content: "你好吗", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Short text test",
			text:         "短文本",
			chunkOverlap: 0,
			chunkSize:    50,
			separators:   []string{"。|！|？", "，|,\\s"},
			expected: []schema.Document{
				{Content: "短文本", Metadata: map[string]any{}},
			},
		},
		{
			name:          "Keep separator with newlines test",
			text:          "第一行\n第二行\n第三行",
			chunkOverlap:  1,
			chunkSize:     8,
			separators:    []string{"\n\n", "\n", ""},
			keepSeparator: true,
			expected: []schema.Document{
				{Content: "第一行", Metadata: map[string]any{}},
				{Content: "\n第二行", Metadata: map[string]any{}},
				{Content: "\n第三行", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Multiple punctuation marks test",
			text:         "这是一个包含多种标点符号的文本：句号。感叹号！问号？分号；逗号，还有其他内容。",
			chunkOverlap: 3,
			chunkSize:    15,
			separators:   []string{"。|！|？", "；|;\\s", "，|,\\s", "：", ""},
			expected: []schema.Document{
				{Content: "这是一个包含多种标点符号的文本", Metadata: map[string]any{}},
				{Content: "句号", Metadata: map[string]any{}},
				{Content: "感叹号", Metadata: map[string]any{}},
				{Content: "问号", Metadata: map[string]any{}},
				{Content: "分号", Metadata: map[string]any{}},
				{Content: "逗号", Metadata: map[string]any{}},
				{Content: "还有其他内容", Metadata: map[string]any{}},
			},
		},
		{
			name:         "Long word test",
			text:         "超长单词测试abcdefghijklmnopqrstuvwxyz1234567890",
			chunkOverlap: 0,
			chunkSize:    10,
			separators:   []string{"。|！|？", "，|,\\s", ""},
			expected: []schema.Document{
				{Content: "超长单词测试abcdefghijklmnopqrstuvwxyz1234567890", Metadata: map[string]any{}},
			},
		},
	}

	splitter := NewChineseRecursiveCharacter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			splitter.ChunkOverlap = tc.chunkOverlap
			splitter.ChunkSize = tc.chunkSize
			if len(tc.separators) > 0 {
				splitter.Separators = tc.separators
			}
			splitter.KeepSeparator = tc.keepSeparator
			if tc.lenFunc != nil {
				splitter.LenFunc = tc.lenFunc
			}

			docs, err := CreateDocuments(splitter, []string{tc.text}, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, docs)
		})
	}
}

func TestChineseRecursiveCharacterSplitterWithCustomOptions(t *testing.T) {
	// 测试自定义选项
	splitter := NewChineseRecursiveCharacter(
		WithChunkSize(20),
		WithChunkOverlap(5),
		WithSeparators([]string{"。", "！", "？", "，", ""}),
		WithKeepSeparator(true),
	)

	text := "这是第一句。这是第二句！这是第三句？这是第四句，还有更多内容。"
	docs, err := CreateDocuments(splitter, []string{text}, nil)
	require.NoError(t, err)

	// 验证结果不为空
	assert.NotEmpty(t, docs)
	// 验证每个文档的内容不为空
	for _, doc := range docs {
		assert.NotEmpty(t, doc.Content)
	}
}

func TestChineseRecursiveCharacterSplitterRegexFeatures(t *testing.T) {
	// 测试正则表达式功能
	splitter := NewChineseRecursiveCharacter(
		WithChunkSize(15),
		WithChunkOverlap(2),
	)

	// 测试包含英文标点的混合文本
	text := "Hello world! 你好世界。How are you? 你好吗？Fine, thanks. 很好，谢谢。"
	docs, err := CreateDocuments(splitter, []string{text}, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, docs)
	// 验证分割结果包含预期的片段
	contents := make([]string, len(docs))
	for i, doc := range docs {
		contents[i] = doc.Content
	}

	// 检查是否正确分割了中英文混合文本
	found := false
	for _, content := range contents {
		if strings.Contains(content, "Hello world") || strings.Contains(content, "你好世界") {
			found = true
			break
		}
	}
	assert.True(t, found, "应该能够正确分割中英文混合文本")
}

func TestChineseRecursiveCharacterSplitterEmptyText(t *testing.T) {
	// 测试空文本
	splitter := NewChineseRecursiveCharacter()
	docs, err := CreateDocuments(splitter, []string{""}, nil)
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestChineseRecursiveCharacterSplitterSingleCharacter(t *testing.T) {
	// 测试单个字符
	splitter := NewChineseRecursiveCharacter(
		WithChunkSize(1),
		WithChunkOverlap(0),
	)

	text := "你好"
	docs, err := CreateDocuments(splitter, []string{text}, nil)
	require.NoError(t, err)

	assert.Len(t, docs, 2)
	assert.Equal(t, "你", docs[0].Content)
	assert.Equal(t, "好", docs[1].Content)
}

func TestChineseRecursiveCharacterSplitterMultipleNewlines(t *testing.T) {
	// 测试多个换行符的清理
	splitter := NewChineseRecursiveCharacter(
		WithChunkSize(50),
		WithChunkOverlap(0),
	)

	text := "第一段\n\n\n\n第二段\n\n\n第三段"
	docs, err := CreateDocuments(splitter, []string{text}, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, docs)
	// 验证多余的换行符被清理
	for _, doc := range docs {
		assert.NotContains(t, doc.Content, "\n\n\n")
	}
}
