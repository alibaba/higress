package utils

import "strings"

func TrimQuote(source string) string {
	return strings.Trim(source, `"`)
}
