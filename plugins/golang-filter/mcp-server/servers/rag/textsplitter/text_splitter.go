package textsplitter

import (
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

// TextSplitter is the standard interface for splitting texts.
type TextSplitter interface {
	SplitText(text string) ([]string, error)
}

func NewTextSplitter(cfg *config.SplitterConfig) (TextSplitter, error) {
	switch cfg.Type {
	case "recursive":
		return NewRecursiveCharacter(WithChunkSize(cfg.ChunkSize), WithChunkOverlap(cfg.ChunkOverlap), WithSeparators([]string{"\n\n", "\n", ".", "。", "?", "!", "；"})), nil
	default:
		return nil, fmt.Errorf("unknown text splitter type: %s", cfg.Type)
	}
}
