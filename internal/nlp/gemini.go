package nlp

import (
	"context"
	"fmt"
	"nexus-core/internal/config"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Brain struct {
	Client *genai.Client
	Model  *genai.GenerativeModel
}

func NewBrain(cfg *config.Config) (*Brain, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.AI.APIKey))
	if err != nil {
		return nil, fmt.Errorf("error creando cliente de Gemini: %v", err)
	}

	model := client.GenerativeModel(cfg.AI.Model)

	// Configuración de temperatura (0.7 es un buen equilibrio entre creatividad y precisión)
	model.SetTemperature(0.7)

	return &Brain{
		Client: client,
		Model:  model,
	}, nil
}

// Ask le hace una pregunta simple al modelo
func (b *Brain) Ask(prompt string) (string, error) {
	ctx := context.Background()
	resp, err := b.Model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 {
		return "Nexus no pudo generar una respuesta.", nil
	}

	// Extraer el texto de la respuesta
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%v", part)
	}

	return result, nil
}
