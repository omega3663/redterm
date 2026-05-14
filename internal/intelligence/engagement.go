package intelligence

import (
	"strings"

	"redterm/internal/config"
)

// buildEngagementHeader returns a formatted block prepended to every LLM system prompt
// when an engagement context is active. Returns "" if eng is nil or empty.
func buildEngagementHeader(eng *config.Engagement) string {
	if eng == nil {
		return ""
	}
	var parts []string
	if eng.Scope != "" {
		parts = append(parts, "Scope: "+eng.Scope)
	}
	if eng.Type != "" {
		parts = append(parts, "Type: "+eng.Type)
	}
	if eng.Objective != "" {
		parts = append(parts, "Objective: "+eng.Objective)
	}
	if eng.Notes != "" {
		parts = append(parts, "Notes: "+eng.Notes)
	}
	if len(parts) == 0 {
		return ""
	}
	return "[ENGAGEMENT]\n" + strings.Join(parts, "  |  ") + "\n---\n\n"
}
