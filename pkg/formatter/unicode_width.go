package formatter

import (
	"unicode"
)

// RuneWidth returns the display width of a rune
// ASCII characters have width 1, CJK characters have width 2
func RuneWidth(r rune) int {
	if r == '\t' {
		return 1
	}

	// ASCII is width 1
	if r < 128 {
		return 1
	}

	// CJK characters (한글, 한자, 일본어 등) are width 2
	if unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) {
		return 2
	}

	// Default for other unicode characters
	return 1
}

// StringWidth returns the display width of a string
func StringWidth(s string) int {
	width := 0
	for _, r := range s {
		width += RuneWidth(r)
	}
	return width
}

// PadString right-pads a string to the specified display width
func PadString(s string, width int) string {
	currentWidth := StringWidth(s)
	if currentWidth >= width {
		return s
	}

	// Add space padding
	return s + string(make([]rune, width-currentWidth))
}
