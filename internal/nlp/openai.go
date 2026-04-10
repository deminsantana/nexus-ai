package nlp

import (
	"bytes"
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

func (o *OpenAIProvider) ProcessAudio(data []byte, mimeType string) (string, error) {
	ctx := context.Background()

	// 1. Transcripción con Whisper
	// Creamos un lector a partir de los bytes recibidos de WhatsApp
	audioReader := bytes.NewReader(data)

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   audioReader,
		FilePath: "voice_note.ogg", // Nombre ficticio para que la API detecte el formato
	}

	transcription, err := o.Client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error en Whisper (STT): %v", err)
	}

	if transcription.Text == "" {
		return "No se detectó audio claro en la nota de voz.", nil
	}

	fmt.Printf("📝 Transcripción de Whisper: %s\n", transcription.Text)

	// 2. Procesar el texto transcrito con el LLM (GPT)
	// Reutilizamos el método Ask para que la IA responda al contenido del audio
	return o.Ask("El usuario envió una nota de voz que dice: " + transcription.Text)
}

func (o *OpenAIProvider) Embed(text string) ([]float32, error) {
	ctx := context.Background()

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2, // Standard model for OpenAI text embeddings
	}

	resp, err := o.Client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error generando embedding en OpenAI: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("el modelo devolvió un embedding vacío")
	}

	return resp.Data[0].Embedding, nil
}

func (o *OpenAIProvider) Close() error {
	// No requiere cierre explícito en la librería actual
	// OpenAI usa HTTP simple y no mantiene una conexión persistente (como gRPC),
	// por lo que el método puede quedar vacío pero debe existir para cumplir con la interfaz.
	return nil
}
