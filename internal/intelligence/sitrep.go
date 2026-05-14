package intelligence

import (
	"context"
	"fmt"

	"redterm/internal/config"
	"redterm/internal/llm"
)

const sitrepSystem = `You are an expert red team operator assistant embedded in a terminal.
Produce a concise situational awareness report from the operator's terminal context.

Structure (omit sections with no evidence):
HOSTS
  • <ip>  <hostname if known>  <role/OS if identified>

SERVICES/APPLICATIONS
  • <ip>:<port>  <service>  <version if known>  <notable detail>
  • <application>  <version if known>  <notable detail>

CREDENTIALS
  • <type>  <value>  <source>

ACCESS
  • Current user, host, and privilege level

FINDINGS
  • Notable misconfigs, vulnerable software, interesting files, domain info, AV/EDR products, security measures

GAPS
  • What has not been enumerated yet

Rules:
- Be terse. Bullet points only. No prose.
- Report only what is directly evidenced in the context. No speculation.
- Flag high-value findings with [!].`

func Sitrep(ctx context.Context, provider llm.Provider, terminalContext string, eng *config.Engagement) (string, error) {
	if terminalContext == "" {
		terminalContext = "[No terminal output captured yet]"
	}
	system := buildEngagementHeader(eng) + sitrepSystem
	user := fmt.Sprintf("Terminal context:\n\n%s\n\nProvide a situational awareness report.", terminalContext)
	return provider.Complete(ctx, system, user)
}
