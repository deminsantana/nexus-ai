package nlp

import (
	"fmt"
	"nexus-core/internal/config"
)

// Provider define el contrato que cualquier IA debe cumplir
type Provider interface {
	Ask(prompt string) (string, error)
	Close() error
}

// Brain es el orquestador que usa un proveedor específico
type Brain struct {
	Provider Provider
}

// NewBrain instancia el proveedor configurado en el YAML
func NewBrain(cfg *config.Config) (*Brain, error) {
	var p Provider
	var err error

	switch cfg.AI.Provider {
	case "google":
		p, err = NewGeminiProvider(cfg)
	case "openai":
		p, err = NewOpenAIProvider(cfg)
	default:
		return nil, fmt.Errorf("proveedor de IA no soportado: %s", cfg.AI.Provider)
	}

	if err != nil {
		return nil, err
	}

	return &Brain{Provider: p}, nil
}

func (b *Brain) ProcessMessage(text string) (string, error) {
	systemPrompt := "Eres Nexus, un asistente personal inteligente. Responde de forma breve y profesional."
	fullPrompt := fmt.Sprintf("%s\n\nUsuario: %s\nNexus:", systemPrompt, text)

	return b.Provider.Ask(fullPrompt)
}
