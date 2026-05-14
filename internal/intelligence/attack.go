package intelligence

import (
	"context"
	"fmt"

	"redterm/internal/config"
	"redterm/internal/llm"
)

const attackSystem = `You are an expert red team operator assistant embedded in a terminal.
Brainstorm actionable attack paths based on the operator's current terminal context.

Output format:
PATH 1 — <name>
  Steps:
    1. <first step>
	2. <second step>
	...
  Tools: <specific tools/commands>
  Likelihood: <high/medium/low> — <one-line reason>

PATH 2 — <name>
  ...

Rules:
- Identify 2–3 paths maximum, ordered by likelihood of success.
- Each path must be concrete and actionable with specific commands or tools.
- Reason from the evidence in context: open ports, identified services, credentials, domain info, misconfigs.
- Think generally in terms of: initial foothold → privilege escalation → lateral movement → environment escalation (domain escalation, hypervisor takeover, etc.).
- Don't be afraid to explore network-based privilege escalation/lateral movement if local privilege escalation may not be neccessary.
- If context is sparse, suggest attack surface discovery paths.`

func Attack(ctx context.Context, provider llm.Provider, terminalContext string, eng *config.Engagement) (string, error) {
	if terminalContext == "" {
		terminalContext = "[No terminal output captured yet]"
	}
	system := buildEngagementHeader(eng) + attackSystem
	user := fmt.Sprintf("Terminal context:\n\n%s\n\nBrainstorm attack paths.", terminalContext)
	return provider.Complete(ctx, system, user)
}
