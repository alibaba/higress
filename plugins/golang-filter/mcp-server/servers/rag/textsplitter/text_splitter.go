package textsplitter

import (
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

// TextSplitter is the standard interface for splitting texts.
type TextSplitter interface {
	SplitText(text string) ([]string, error)
}

type NoSplitterCharacter struct {
}

func (s NoSplitterCharacter) SplitText(text string) ([]string, error) {
	return []string{text}, nil
}

func NewTextSplitter(cfg *config.SplitterConfig) (TextSplitter, error) {
	switch cfg.Provider {
	case "recursive":
		return NewRecursiveCharacter(WithChunkSize(cfg.ChunkSize), WithChunkOverlap(cfg.ChunkOverlap), WithSeparators([]string{"\n\n", "\n", ".", "。", "?", "!", "；"})), nil
	case "nosplitter":
		return NoSplitterCharacter{}, nil
	default:
		return nil, fmt.Errorf("unknown text splitter type: %s", cfg.Provider)
	}
}
