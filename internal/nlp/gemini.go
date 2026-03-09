package nlp

import (
	"context"
	"fmt"
	"nexus-core/internal/config"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	Client *genai.Client
	Model  *genai.GenerativeModel
}

func NewGeminiProvider(cfg *config.Config) (*GeminiProvider, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.AI.APIKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(cfg.AI.Model)
	model.SetTemperature(0.7)

	return &GeminiProvider{
		Client: client,
		Model:  model,
	}, nil
}

func (g *GeminiProvider) Ask(prompt string) (string, error) {
	ctx := context.Background()
	resp, err := g.Model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 {
		return "Sin respuesta del modelo.", nil
	}

	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%v", part)
	}
	return result, nil
}

func (g *GeminiProvider) Close() error {
	// Mantiene una conexión persistente tipo gRPC,
	// es necesario cerrarla.
	if g.Client != nil {
		return g.Client.Close()
	}
	return nil
}
