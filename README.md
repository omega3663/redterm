# REDTERM - An Intelligent Terminal for Red Teaming

**Features**

Redterm wraps your shell in a PTY and overlays LLM intelligence on top of your normal terminal workflow. Press `Ctrl+G` at any point to open the command bar without interrupting your session.

- Real-time command suggestions, attack path brainstorming, and situational awareness reports
- Structured finding extraction from nmap, NetExec, and BloodHound output
- Cross-session context sharing across multiple redterm instances on the same host
- Engagement scoping injected into every prompt (scope, type, objective, notes)
- Supports OpenAI, Anthropic, and Ollama (local) as LLM backends

> **Warning:** When using OpenAI or Anthropic as the LLM provider, terminal output — including hostnames, IP addresses, credentials, and tool output — is sent to third-party cloud APIs. Use Ollama with a local model for air-gapped or sensitive engagements.

**Install**

Requires Go 1.26+.

```
git clone <repo>
cd redterm
go build -o redterm .
```

Copy the example config to `~/.config/redterm/config.yaml`:

```
cp config.example.yaml ~/.config/redterm/config.yaml
```

Config fields:

| Field | Default | Description |
|---|---|---|
| `provider` | `ollama` | `openai`, `anthropic`, or `ollama` |
| `model` | `llama3.2` | Model name for the chosen provider |
| `api_key` | — | API key (or set `REDTERM_API_KEY`) |
| `base_url` | `http://localhost:11434` | Override for Ollama or OpenAI-compatible endpoints |
| `context_lines` | `500` | Rolling buffer size (lines of terminal output) |
| `shell` | `/bin/bash` | Shell to spawn |
| `trigger_key` | `ctrl+g` | Key chord that opens the command bar |

Environment variable overrides: `REDTERM_PROVIDER`, `REDTERM_MODEL`, `REDTERM_API_KEY`, `REDTERM_BASE_URL`.

**Usage**

```
./redterm [--config ~/.config/redterm/config.yaml]
```

Press `Ctrl+G` to open the command bar, then type a slash command:

| Command | Description |
|---|---|
| `/suggest` | Recommend up to 3 next commands with kill-chain reasoning |
| `/attack` | Brainstorm 2–3 attack paths (steps, tools, likelihood) |
| `/sitrep` | Situational awareness: hosts, credentials, access, findings, gaps |
| `/engage` | Set engagement scope, type, objective, and notes |
| `/context <file>` | Inject a file into the analysis context |
| `/prompt <text>` | Manually add text to the context buffer |
| `/clear` | Clear the context buffer |

Engagement context (set via `/engage`) is prepended to every LLM prompt:

| Field | Example |
|---|---|
| Scope | `10.0.0.0/24, corp.local` |
| Type | `internal`, `external`, `assumed-breach`, `purple` |
| Objective | `Establish persistence on file server` |
| Notes | `No EDR detected on initial scan` |
