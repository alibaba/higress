package util

func StripPrefix(s string, prefix string) string {
	if len(prefix) != 0 && len(s) >= len(prefix) && s[0:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
