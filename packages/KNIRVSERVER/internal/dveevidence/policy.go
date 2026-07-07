package dveevidence

import "strings"

type Policy struct {
	AllowedCommands    []string `json:"allowed_commands,omitempty"`
	DeniedCommands     []string `json:"denied_commands,omitempty"`
	AllowedCredentials []string `json:"allowed_credentials,omitempty"`
	AllowNetwork       bool     `json:"allow_network"`
}

type PolicyViolation struct {
	EventType string
	Expected  string
	Actual    string
}

func ReplayPolicy(b *Bundle, policy *Policy) []PolicyViolation {
	if policy == nil || b == nil {
		return nil
	}
	var violations []PolicyViolation
	for _, d := range b.PermissionDecisions {
		input := strings.TrimSpace(d.Input)
		for _, denied := range policy.DeniedCommands {
			if denied != "" && strings.HasPrefix(input, denied) && d.Action != "denied" && !d.Denied {
				violations = append(violations, PolicyViolation{
					EventType: d.EventType,
					Expected:  "denied",
					Actual:    d.Action,
				})
				break
			}
		}
		if strings.Contains(strings.ToLower(d.EventType), "network") && !policy.AllowNetwork && !d.Denied {
			violations = append(violations, PolicyViolation{
				EventType: d.EventType,
				Expected:  "network denied",
				Actual:    "allowed",
			})
		}
		if len(policy.AllowedCommands) > 0 && input != "" {
			allowed := false
			for _, prefix := range policy.AllowedCommands {
				if strings.HasPrefix(input, prefix) {
					allowed = true
					break
				}
			}
			if !allowed && d.Action == "approved" && !d.Denied {
				violations = append(violations, PolicyViolation{
					EventType: d.EventType,
					Expected:  "command allowed by policy",
					Actual:    input,
				})
			}
		}
	}
	return violations
}
