// Package config resolves environment variables in the alertmanager YAML template
// at startup, substituting real Slack/PagerDuty/SMTP credentials before writing the
// final config to disk.
//
// Usage:
//   alertmanager-config /path/to/alertmanager.template.yml /etc/alertmanager/alertmanager.yml
//
// Env vars resolved:
//   SLACK_WEBHOOK_URL         — Slack incoming webhook URL (all channels)
//   PAGERDUTY_INTEGRATION_KEY — PagerDuty events API integration key
//   SMTP_PASSWORD             — SMTP auth password for alerts@knirv.com
//
// Variables use the Bash-like ${VAR:-default} syntax.
package config

import (
	"fmt"
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}:-]+)(?::-([^}]*))?}`)

// ResolveEnvVars replaces ${VAR} and ${VAR:-default} placeholders in the input
// with values from the environment. Returns an error only if a required variable
// (one without a :-default) is unset or empty.
func ResolveEnvVars(input string) (string, error) {
	return resolveVars(input, os.Getenv)
}

func resolveVars(input string, lookup func(string) string) (string, error) {
	var lastErr error
	result := envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		name := parts[1]
		defaultVal := ""
		if len(parts) >= 3 {
			defaultVal = parts[2]
		}

		val := lookup(name)
		if val == "" {
			if defaultVal != "" {
				return defaultVal
			}
			lastErr = fmt.Errorf("required env var %s is unset or empty", name)
			return match
		}
		return val
	})
	if lastErr != nil {
		return result, lastErr
	}
	return result, nil
}

// WriteResolvedConfig reads a template file, resolves ${VAR:-default} placeholders
// against the environment, and writes the result to the output path.
func WriteResolvedConfig(templatePath, outputPath string) error {
	input, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", templatePath, err)
	}

	resolved, err := ResolveEnvVars(string(input))
	if err != nil {
		return fmt.Errorf("resolve env vars: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(resolved), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	return nil
}
