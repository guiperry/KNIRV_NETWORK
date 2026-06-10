package mcphardening

import (
	"regexp"
	"strings"
)

type InjectionDetector struct {
	patterns []*regexp.Regexp
}

var defaultInjectionPatterns = []string{
	`(?i)\bDROP\s+TABLE`,
	`(?i)\bDELETE\s+FROM`,
	`(?i)\bINSERT\s+INTO`,
	`(?i)\bEXEC\b`,
	`(?i)<script`,
	`(?i)javascript:`,
	`(?i)onerror\s*=`,
	`(?i)onload\s*=`,
	`(?i)\$\{.*\}`,
	`(?i)\|.*\bbash\b`,
	`(?i)\|.*\bsh\b`,
	`(?i)` + "`" + `[^` + "`" + `]*` + "`" + ``,
	`(?i)\brm\s+-rf`,
	`(?i)\bwget\s+`,
	`(?i)\bcurl\s+`,
}

func NewInjectionDetector() *InjectionDetector {
	patterns := make([]*regexp.Regexp, 0, len(defaultInjectionPatterns))
	for _, p := range defaultInjectionPatterns {
		patterns = append(patterns, regexp.MustCompile(p))
	}
	return &InjectionDetector{patterns: patterns}
}

func (id *InjectionDetector) Detect(value string) (bool, string) {
	for _, pat := range id.patterns {
		if pat.MatchString(value) {
			return true, pat.String()
		}
	}
	return false, ""
}

func (id *InjectionDetector) Sanitize(value string) string {
	for _, pat := range id.patterns {
		if pat.MatchString(value) {
			return strings.Repeat("*", len(value))
		}
	}
	return value
}

func (id *InjectionDetector) ScanArguments(args map[string]interface{}) []string {
	var flagged []string
	for key, val := range args {
		str, ok := val.(string)
		if !ok {
			continue
		}
		if detected, _ := id.Detect(str); detected {
			flagged = append(flagged, key)
		}
	}
	return flagged
}
