package llm

import (
	"context"
	"fmt"

	"redterm/internal/config"
)

// Provider is implemented by all LLM backends.
type Provider interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// New returns the Provider configured by cfg.
func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return newOpenAI(cfg), nil
	case "anthropic":
		return newAnthropic(cfg), nil
	case "ollama":
		return newOllama(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (choices: openai, anthropic, ollama)", cfg.Provider)
	}
}
