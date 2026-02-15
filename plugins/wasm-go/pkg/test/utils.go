package test

import (
	"strings"
)

// HasHeader checks if the headers contain a header with the specified name (case-insensitive)
func HasHeader(headers [][2]string, headerName string) bool {
	for _, header := range headers {
		if strings.EqualFold(header[0], headerName) {
			return true
		}
	}
	return false
}

// GetHeaderValue returns the value of the specified header (case-insensitive)
// Returns empty string and false if header is not found
func GetHeaderValue(headers [][2]string, headerName string) (string, bool) {
	for _, header := range headers {
		if strings.EqualFold(header[0], headerName) {
			return header[1], true
		}
	}
	return "", false
}

// HasHeaderWithValue checks if the headers contain a header with the specified name and value (case-insensitive)
func HasHeaderWithValue(headers [][2]string, headerName, expectedValue string) bool {
	for _, header := range headers {
		if strings.EqualFold(header[0], headerName) {
			return strings.EqualFold(header[1], expectedValue)
		}
	}
	return false
}
