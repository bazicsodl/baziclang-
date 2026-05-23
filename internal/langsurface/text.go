package langsurface

import "regexp"

type LineCol struct {
	Line      int
	Character int
}

func IsIdentChar(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func WordAt(text string, pos LineCol) string {
	lines := splitLines(text)
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if pos.Character < 0 || pos.Character > len(line) {
		return ""
	}
	start := pos.Character
	for start > 0 && IsIdentChar(rune(line[start-1])) {
		start--
	}
	end := pos.Character
	for end < len(line) && IsIdentChar(rune(line[end])) {
		end++
	}
	if start == end {
		return ""
	}
	return line[start:end]
}

func FindWordRanges(text, word string) [][2]int {
	if word == "" {
		return nil
	}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	matches := pattern.FindAllStringIndex(text, -1)
	out := make([][2]int, 0, len(matches))
	for _, m := range matches {
		out = append(out, [2]int{m[0], m[1]})
	}
	return out
}

func splitLines(text string) []string {
	lines := make([]string, 0, 16)
	start := 0
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	lines = append(lines, text[start:])
	return lines
}
