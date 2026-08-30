package main

import (
	"fmt"
	"strings"
	"sync"
)

var outputMu sync.Mutex

func parseKeywords(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	keywords := make([]string, 0, len(parts))
	for _, kw := range parts {
		trimmed := strings.TrimSpace(kw)
		if trimmed != "" {
			keywords = append(keywords, trimmed)
		}
	}

	return keywords
}

// Thread-safe logging function
func logf(format string, args ...any) {
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Printf(format, args...)
}
