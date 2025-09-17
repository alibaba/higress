package textsplitter

import (
	"regexp"
	"strings"
)

// ChineseRecursiveCharacter is a text splitter that will split Chinese texts recursively by different
// characters with regex support.
type ChineseRecursiveCharacter struct {
	Separators       []string
	ChunkSize        int
	ChunkOverlap     int
	LenFunc          func(string) int
	KeepSeparator    bool
	IsSeparatorRegex bool
}

// NewChineseRecursiveCharacter creates a new Chinese recursive character splitter with default values.
// By default, the separators used are Chinese punctuation marks and the chunk size is set to 4000
// and chunk overlap is set to 200.
func NewChineseRecursiveCharacter(opts ...Option) ChineseRecursiveCharacter {
	options := DefaultOptions()
	for _, o := range opts {
		o(&options)
	}

	// Default Chinese separators
	defaultSeparators := []string{
		"\n\n",
		"\n",
		"。|！|？",
		"\\.\\s|\\!\\s|\\?\\s",
		"；|;\\s",
		"，|,\\s",
	}

	if len(options.Separators) == 0 {
		options.Separators = defaultSeparators
	}

	s := ChineseRecursiveCharacter{
		Separators:       options.Separators,
		ChunkSize:        options.ChunkSize,
		ChunkOverlap:     options.ChunkOverlap,
		LenFunc:          options.LenFunc,
		KeepSeparator:    options.KeepSeparator,
		IsSeparatorRegex: true,
	}

	return s
}

// SplitText splits a text into multiple text.
func (s ChineseRecursiveCharacter) SplitText(text string) ([]string, error) {
	return s.splitText(text, s.Separators)
}

// splitTextWithRegexFromEnd splits text with regex and optionally keeps separator
func (s ChineseRecursiveCharacter) splitTextWithRegexFromEnd(text, separator string, keepSeparator bool) []string {
	if separator == "" {
		// Split into individual characters
		runes := []rune(text)
		result := make([]string, len(runes))
		for i, r := range runes {
			result[i] = string(r)
		}
		return filterEmptyStrings(result)
	}

	if keepSeparator {
		// Split with regex and keep separator
		re := regexp.MustCompile("(" + separator + ")")
		splits := re.Split(text, -1)
		separators := re.FindAllString(text, -1)

		result := make([]string, 0)
		for i, split := range splits {
			if i < len(separators) {
				result = append(result, split+separators[i])
			} else {
				result = append(result, split)
			}
		}
		return filterEmptyStrings(result)
	} else {
		re := regexp.MustCompile(separator)
		splits := re.Split(text, -1)
		return filterEmptyStrings(splits)
	}
}

func (s ChineseRecursiveCharacter) splitText(text string, separators []string) ([]string, error) {
	finalChunks := make([]string, 0)

	// Find the appropriate separator
	separator := separators[len(separators)-1]
	newSeparators := []string{}
	for i, sep := range separators {
		regexSep := sep
		if !s.IsSeparatorRegex {
			regexSep = regexp.QuoteMeta(sep)
		}

		if sep == "" {
			separator = sep
			break
		}

		if matched, _ := regexp.MatchString(regexSep, text); matched {
			separator = sep
			newSeparators = separators[i+1:]
			break
		}
	}

	regexSeparator := separator
	if !s.IsSeparatorRegex {
		regexSeparator = regexp.QuoteMeta(separator)
	}

	splits := s.splitTextWithRegexFromEnd(text, regexSeparator, s.KeepSeparator)

	goodSplits := make([]string, 0)
	mergeSeparator := ""
	if !s.KeepSeparator {
		mergeSeparator = separator
	}

	// Merge the splits, recursively splitting larger texts
	for _, split := range splits {
		if s.LenFunc(split) < s.ChunkSize {
			goodSplits = append(goodSplits, split)
			continue
		}

		if len(goodSplits) > 0 {
			mergedText := mergeSplits(goodSplits, mergeSeparator, s.ChunkSize, s.ChunkOverlap, s.LenFunc)
			finalChunks = append(finalChunks, mergedText...)
			goodSplits = make([]string, 0)
		}

		if len(newSeparators) == 0 {
			finalChunks = append(finalChunks, split)
		} else {
			otherInfo, err := s.splitText(split, newSeparators)
			if err != nil {
				return nil, err
			}
			finalChunks = append(finalChunks, otherInfo...)
		}
	}

	if len(goodSplits) > 0 {
		mergedText := mergeSplits(goodSplits, mergeSeparator, s.ChunkSize, s.ChunkOverlap, s.LenFunc)
		finalChunks = append(finalChunks, mergedText...)
	}

	// Clean up chunks: remove multiple newlines and empty chunks
	cleanedChunks := make([]string, 0)
	for _, chunk := range finalChunks {
		// Replace multiple newlines with single newline
		re := regexp.MustCompile(`\n{2,}`)
		cleanedChunk := re.ReplaceAllString(strings.TrimSpace(chunk), "\n")
		if cleanedChunk != "" {
			cleanedChunks = append(cleanedChunks, cleanedChunk)
		}
	}

	return cleanedChunks, nil
}

// filterEmptyStrings removes empty strings from slice
func filterEmptyStrings(strs []string) []string {
	result := make([]string, 0)
	for _, s := range strs {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
