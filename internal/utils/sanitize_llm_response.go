package utils

import "strings"

func SanitizeResponse(s string) string {
	s = strings.TrimSpace(s)

	// remove BOM
	s = strings.TrimPrefix(s, "\uFEFF")

	// normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// remove zero-width characters
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			return -1
		}
		return r
	}, s)

	// prefer explicit json fence extraction
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+len("```json"):]
		s = strings.TrimLeft(s, "\n\t ")

		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
		return strings.TrimSpace(s)
	}

	// remove markdown code fences
	if strings.HasPrefix(s, "```") {
		s = s[3:]

		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}

		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}

	// remove leading "Here is ..." junk
	l := strings.ToLower(s)
	for _, p := range []string{
		"here is the result:",
		"here is the output:",
		"here is the json:",
		"result:",
		"output:",
	} {
		if strings.HasPrefix(l, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}
	return strings.TrimSpace(s)
}

func SanitizeResponseJSON(s string) string {
	objStart := strings.Index(s, "{")
	objEnd := strings.LastIndex(s, "}")

	arrStart := strings.Index(s, "[")
	arrEnd := strings.LastIndex(s, "]")

	switch {
	case objStart == -1 && arrStart == -1:
		return s
	case arrStart >= 0 && (objStart == -1 || arrStart < objStart):
		if arrEnd > arrStart {
			return s[arrStart : arrEnd+1]
		}
		return s
	case objStart >= 0:
		if objEnd > objStart {
			return s[objStart : objEnd+1]
		}
		return s
	default:
		return s
	}
}
