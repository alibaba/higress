package streamxform

import (
	"strconv"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// appendJSONString 按 encoding/json 的规则（含 HTML 安全转义 < > &）编码字符串。
// 与官方 json.Marshal 的输出逐字节一致，差分比对时不产生噪音。
func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if safeSet[b] {
				i++
				continue
			}
			dst = append(dst, s[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				// 控制字符与 < > & 走 \u00XX
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			dst = append(dst, s[start:i]...)
			dst = append(dst, `�`...)
			i += size
			start = i
			continue
		}
		if c == ' ' || c == ' ' {
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', hexDigits[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, s[start:]...)
	return append(dst, '"')
}

// safeSet 与 encoding/json 的 htmlSafeSet 一致：这些 ASCII 字节原样输出。
var safeSet = func() [utf8.RuneSelf]bool {
	var s [utf8.RuneSelf]bool
	for b := 0x20; b < utf8.RuneSelf; b++ {
		s[b] = true
	}
	s['"'] = false
	s['\\'] = false
	s['<'] = false
	s['>'] = false
	s['&'] = false
	return s
}()

func appendInt(dst []byte, n int) []byte { return strconv.AppendInt(dst, int64(n), 10) }

// jsonUnquote 解码一个 JSON 字符串字面量（含两端引号）。
// 非法输入返回 ok=false。
func jsonUnquote(raw []byte) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	return unescapeInner(raw[1 : len(raw)-1])
}

// unescapeInner 解码不含引号的字符串内容。
func unescapeInner(b []byte) (string, bool) {
	// 快路径：无转义
	hasEsc := false
	for _, c := range b {
		if c == '\\' {
			hasEsc = true
			break
		}
	}
	if !hasEsc {
		return string(b), true
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		i++
		if i >= len(b) {
			return "", false
		}
		switch b[i] {
		case '"', '\\', '/':
			out = append(out, b[i])
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if i+4 >= len(b) {
				return "", false
			}
			r, ok := hex4(b[i+1 : i+5])
			if !ok {
				return "", false
			}
			i += 4
			if r >= 0xD800 && r < 0xDC00 { // 高代理，尝试配对
				if i+6 < len(b) && b[i+1] == '\\' && b[i+2] == 'u' {
					if r2, ok2 := hex4(b[i+3 : i+7]); ok2 && r2 >= 0xDC00 && r2 < 0xE000 {
						r = 0x10000 + (r-0xD800)<<10 + (r2 - 0xDC00)
						i += 6
					} else {
						r = utf8.RuneError
					}
				} else {
					r = utf8.RuneError
				}
			} else if r >= 0xDC00 && r < 0xE000 {
				r = utf8.RuneError
			}
			out = utf8.AppendRune(out, r)
		default:
			return "", false
		}
	}
	return string(out), true
}

func hex4(b []byte) (rune, bool) {
	var r rune
	for _, c := range b {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	return r, true
}

// unescapePrefix 解码字符串前缀，同时记录每个解码后字节对应的原始偏移，
// 用于"解码后判定、原始字节续传"的场景（如 data: URL 头）。
// 末尾不完整的转义序列不解码；rawOff 末尾多一个哨兵，指向解码停止处的原始偏移，
// 因此 rawOff[len(dec)] 之后的原始字节可以原样续传。
func unescapePrefix(b []byte) (dec []byte, rawOff []int) {
	dec = make([]byte, 0, len(b))
	rawOff = make([]int, 0, len(b)+1)
	i := 0
	for i < len(b) {
		c := b[i]
		if c != '\\' {
			dec = append(dec, c)
			rawOff = append(rawOff, i)
			i++
			continue
		}
		if i+1 >= len(b) {
			break // 不完整的转义：停在这里
		}
		start := i
		var r rune
		consumed := 2
		switch b[i+1] {
		case '"', '\\', '/':
			r = rune(b[i+1])
		case 'b':
			r = '\b'
		case 'f':
			r = '\f'
		case 'n':
			r = '\n'
		case 'r':
			r = '\r'
		case 't':
			r = '\t'
		case 'u':
			if i+6 > len(b) {
				i = len(b) + 1 // 标记：不完整
				break
			}
			v, ok := hex4(b[i+2 : i+6])
			if !ok {
				v = utf8.RuneError
			}
			r = v
			consumed = 6
		default:
			r = utf8.RuneError
		}
		if i > len(b) {
			i = start
			break
		}
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], r)
		for k := 0; k < n; k++ {
			dec = append(dec, tmp[k])
			rawOff = append(rawOff, start)
		}
		i = start + consumed
	}
	rawOff = append(rawOff, i) // 哨兵
	return dec, rawOff
}

// isZeroNum 判断数字字面量是否为零值（0 / 0.0 / 0e0 …）——复刻 omitempty 语义。
func isZeroNum(s []byte) bool {
	f, err := strconv.ParseFloat(string(s), 64)
	return err == nil && f == 0
}

// isIntLiteral 判断是否是 encoding/json 能解进 int 的字面量。
func isIntLiteral(s []byte) bool {
	_, err := strconv.ParseInt(string(s), 10, 64)
	return err == nil
}

func isNumLiteral(s []byte) bool {
	_, err := strconv.ParseFloat(string(s), 64)
	return err == nil && len(s) > 0 && s[0] != '+' && s[0] != '.'
}

// gjsonBool 复刻 gjson.Result.Bool()：true 字面量、非零数字、可被 ParseBool 的字符串。
func gjsonBool(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	switch raw[0] {
	case 't':
		return string(raw) == "true"
	case '"':
		s, ok := jsonUnquote(raw)
		if !ok {
			return false
		}
		b, err := strconv.ParseBool(lower(s))
		return err == nil && b
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		f, err := strconv.ParseFloat(string(raw), 64)
		return err == nil && f != 0
	}
	return false
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func isScalarByte(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case c == '-', c == '+', c == '.':
		return true
	}
	return false
}
