package render

import (
	"strings"
	"text/template"
	"time"
	"unicode"
)

// defaultFuncs returns the helper functions every template can rely on.
// They are deliberately few: naming conversions, indentation and the current
// year cover the needs of scaffolding templates.
func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"lower":  strings.ToLower,
		"upper":  strings.ToUpper,
		"trim":   strings.TrimSpace,
		"kebab":  Kebab,
		"snake":  Snake,
		"pascal": Pascal,
		"camel":  Camel,
		"title":  Title,
		"indent": Indent,
		"year":   func() int { return time.Now().Year() },
		"join":   strings.Join,
		"default": func(fallback, value string) string {
			if strings.TrimSpace(value) == "" {
				return fallback
			}
			return value
		},
	}
}

// words splits an identifier of any common casing into its lowercase parts.
func words(s string) []string {
	var out []string
	var cur []rune

	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = nil
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			flush()
		case unicode.IsUpper(r):
			// Start a new word at a lower-to-upper transition, and at the
			// end of an acronym such as HTTPServer.
			prevLower := i > 0 && unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (len(cur) > 0 && nextLower) {
				flush()
			}
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return out
}

// Kebab converts a name to kebab-case.
func Kebab(s string) string { return strings.Join(words(s), "-") }

// Snake converts a name to snake_case.
func Snake(s string) string { return strings.Join(words(s), "_") }

// Pascal converts a name to PascalCase.
func Pascal(s string) string {
	parts := words(s)
	for i, p := range parts {
		parts[i] = capitalise(p)
	}
	return strings.Join(parts, "")
}

// Camel converts a name to camelCase.
func Camel(s string) string {
	parts := words(s)
	for i, p := range parts {
		if i == 0 {
			continue
		}
		parts[i] = capitalise(p)
	}
	return strings.Join(parts, "")
}

// Title converts a name to a space separated title, suitable for headings.
func Title(s string) string {
	parts := words(s)
	for i, p := range parts {
		parts[i] = capitalise(p)
	}
	return strings.Join(parts, " ")
}

// Indent prefixes every non-empty line of s with n spaces.
func Indent(n int, s string) string {
	if n <= 0 || s == "" {
		return s
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
