package util

import (
	"regexp"
	"strconv"
	"strings"
)

func StripPrefix(s string, prefix string) string {
	if len(prefix) != 0 && len(s) >= len(prefix) && s[0:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func MatchStatus(status string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, status)
		if matched {
			return true
		}
	}
	return false
}

// unicodeEscapeRegex matches Unicode escape sequences like \uXXXX
var unicodeEscapeRegex = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// DecodeUnicodeEscapes decodes Unicode escape sequences (\uXXXX) in a string to UTF-8 characters.
// This is useful when a JSON response contains ASCII-safe encoded non-ASCII characters.
func DecodeUnicodeEscapes(input []byte) []byte {
	result := unicodeEscapeRegex.ReplaceAllFunc(input, func(match []byte) []byte {
		// match is like \uXXXX, extract the hex part (XXXX)
		hexStr := string(match[2:6])
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			return match // return original if parse fails
		}
		return []byte(string(rune(codePoint)))
	})
	return result
}

// DecodeUnicodeEscapesInSSE decodes Unicode escape sequences in SSE formatted data.
// It processes each line that starts with "data: " and decodes Unicode escapes in the JSON payload.
func DecodeUnicodeEscapesInSSE(input []byte) []byte {
	lines := strings.Split(string(input), "\n")
	var result strings.Builder
	for i, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			// Decode Unicode escapes in the JSON payload
			jsonData := line[6:]
			decodedData := DecodeUnicodeEscapes([]byte(jsonData))
			result.WriteString("data: ")
			result.Write(decodedData)
		} else {
			result.WriteString(line)
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	return []byte(result.String())
}
