package view

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeTerminal returns s with terminal control sequences that could
// corrupt the user's terminal stripped out. C0/C1 control bytes other
// than \t, \n, \r are removed. Escape sequences are dropped unless they
// are SGR (CSI ... 'm') sequences used for plain colour, which are
// passed through unchanged.
func SanitizeTerminal(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		c := s[i]

		// ESC: parse the escape sequence and either pass through (SGR) or drop.
		if c == 0x1b {
			i = handleEscape(s, i, &b)
			continue
		}

		// Fast path for ASCII bytes.
		if c < 0x80 {
			if c == '\t' || c == '\n' || c == '\r' {
				b.WriteByte(c)
			} else if c < 0x20 || c == 0x7f {
				// drop C0 control + DEL
			} else {
				b.WriteByte(c)
			}
			i++
			continue
		}

		// Multi-byte: decode rune and drop only if it is a control rune.
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte — drop it.
			i++
			continue
		}
		if unicode.IsControl(r) {
			// C1 controls (\x80..\x9f) and other control runes — drop.
			i += size
			continue
		}
		b.WriteString(s[i : i+size])
		i += size
	}

	return b.String()
}

// handleEscape consumes an escape sequence starting at s[i] (s[i] == ESC)
// and returns the index immediately after the consumed bytes. If the
// sequence is an SGR (CSI ... 'm'), it is written to b unchanged.
func handleEscape(s string, i int, b *strings.Builder) int {
	// Lone trailing ESC — drop.
	if i+1 >= len(s) {
		return len(s)
	}

	next := s[i+1]
	switch next {
	case ']': // OSC: drop until BEL or ST (ESC \).
		j := i + 2
		for j < len(s) {
			if s[j] == 0x07 {
				return j + 1
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
			j++
		}
		return len(s)
	case '[': // CSI: parameters, intermediates, final byte.
		j := i + 2
		// Parameter bytes 0x30-0x3F.
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3f {
			j++
		}
		// Intermediate bytes 0x20-0x2F.
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f {
			j++
		}
		if j >= len(s) {
			return len(s)
		}
		final := s[j]
		if final == 'm' {
			// SGR: pass through unchanged.
			b.WriteString(s[i : j+1])
		}
		return j + 1
	case 'P', '_', '^': // DCS / APC / PM: drop until ST.
		j := i + 2
		for j < len(s) {
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
			if s[j] == 0x07 {
				return j + 1
			}
			j++
		}
		return len(s)
	default:
		// Other 2-char ESC sequences (e.g. ESC M) — drop both bytes.
		return i + 2
	}
}
