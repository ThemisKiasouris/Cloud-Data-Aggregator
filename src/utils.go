package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var outputMu sync.Mutex

type Keyword struct {
	Raw string
	Re  *regexp.Regexp
}

// parseKeywords splits the raw comma-separated input and compiles each
// keyword into a case-insensitive, whole-word regex.
func parseKeywords(raw string) ([]Keyword, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	patterns := make([]Keyword, 0, len(parts))
	for _, kw := range parts {
		trimmed := strings.TrimSpace(kw)
		if trimmed == "" {
			continue
		}
		pattern := `(?i)\b` + regexp.QuoteMeta(trimmed) + `\b`
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid keyword %q: %w", trimmed, err)
		}
		patterns = append(patterns, Keyword{Raw: trimmed, Re: re})
	}

	return patterns, nil
}

// Thread-safe logging function
func logf(format string, args ...any) {
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Printf(format, args...)
}
