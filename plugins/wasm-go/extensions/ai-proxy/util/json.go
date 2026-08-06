package util

import (
	"strconv"
	"strings"
)

func EscapeStringForJson(s string) string {
	var builder strings.Builder
	for _, c := range s { //iterate through rune
		switch c {
		case '"':
			builder.WriteRune('\\')
			builder.WriteRune(c)
			break
		default:
			quoted := strconv.QuoteRune(c)
			builder.WriteString(quoted[1 : len(quoted)-1])
		}
	}
	return builder.String()
}
