package util

import re "regexp"

// IsValidBase64 checks if a string is a valid base64 encoded string
func IsValidBase64(s string) bool {
	return re.MustCompile(`^[a-zA-Z0-9+/=-]+$`).MatchString(s)
}
