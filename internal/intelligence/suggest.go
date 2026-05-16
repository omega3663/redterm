package intelligence

import (
	"context"
	"fmt"
	"strings"

	"redterm/internal/config"
	"redterm/internal/llm"
)

const suggestSystem = `You are an expert red team operator assistant embedded in a terminal.
Suggest the single most tactically valuable next command given the current terminal context.

Output format (strictly):
COMMAND [1]: <exact command to run>
REASON:  <one sentence explaining why>

COMMAND [2]: <exact command to run>
REASON:  <one sentence explaining why>

...

Rules:
- Recommend up to three commands. No preamble, no alternatives.
- Prefer commands that directly build on evidence in the context: hosts, ports, services, credentials, file paths.
- Cover the full kill chain where relevant: enumeration, exploitation, credential abuse, lateral movement, privilege escalation, persistence, exfiltration.
- If the context is empty or minimal, recommend an appropriate initial enumeration command.
- Use real tool names (nmap, netexec, impacket, bloodhound-python, etc.).`

// ParseSuggestCommands extracts the command strings from a /suggest LLM response.
// Matches lines of the form "COMMAND1: <cmd>", "COMMAND2: <cmd>", "COMMAND3: <cmd>".
func ParseSuggestCommands(result string) []string {
	var cmds []string
	for _, line := range strings.Split(result, "\n") {
		upper := strings.ToUpper(strings.TrimSpace(line))
		for _, prefix := range []string{"COMMAND [1]:", "COMMAND [2]:", "COMMAND [3]:"} {
			if strings.HasPrefix(upper, prefix) {
				// Extract from the original line to preserve casing
				idx := strings.Index(strings.ToUpper(line), prefix)
				cmd := strings.TrimSpace(line[idx+len(prefix):])
				if cmd != "" {
					cmds = append(cmds, cmd)
				}
				break
			}
		}
	}
	return cmds
}

func Suggest(ctx context.Context, provider llm.Provider, terminalContext string, eng *config.Engagement) (string, error) {
	if terminalContext == "" {
		terminalContext = "[No terminal output captured yet]"
	}
	system := buildEngagementHeader(eng) + suggestSystem
	user := fmt.Sprintf("Terminal context:\n\n%s\n\nSuggest the next command.", terminalContext)
	return provider.Complete(ctx, system, user)
}
