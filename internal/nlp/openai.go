package nlp

import (
	"context"
	"fmt"
	"nexus-core/internal/config"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	Client *openai.Client
	Model  string
}

func NewOpenAIProvider(cfg *config.Config) (*OpenAIProvider, error) {
	client := openai.NewClient(cfg.AI.APIKey)
	return &OpenAIProvider{
		Client: client,
		Model:  cfg.AI.Model,
	}, nil
}

func (o *OpenAIProvider) Ask(prompt string) (string, error) {
	resp, err := o.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: o.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("error OpenAI: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (o *OpenAIProvider) Close() error {
	// No requiere cierre explícito en la librería actual
	// OpenAI usa HTTP simple y no mantiene una conexión persistente (como gRPC),
	// por lo que el método puede quedar vacío pero debe existir para cumplir con la interfaz.
	return nil
}
