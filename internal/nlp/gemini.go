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
	
	apiKey := cfg.AI.APIKey
	if cfg.Google.APIKey != "" {
		apiKey = cfg.Google.APIKey
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
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

func (g *GeminiProvider) ProcessAudio(data []byte, mimeType string) (string, error) {
	ctx := context.Background()

	// Prompt de sistema para dar contexto al audio
	// prompt := genai.Text("Soy tu dueño, Demin Santana. Escucha este audio y responde de forma inteligente.")
	prompt := genai.Text("Transcribe exactamente lo que dice este audio y luego responde a la petición. Formato: [Transcripción] | [Respuesta]")

	// Adjuntamos los datos del audio (WhatsApp suele enviar audio/ogg; codecs=opus)
	blob := genai.Blob{
		MIMEType: mimeType,
		Data:     data,
	}

	resp, err := g.Model.GenerateContent(ctx, prompt, blob)
	if err != nil {
		return "", fmt.Errorf("error procesando audio en Gemini: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "No pude entender el audio.", nil
	}

	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%v", part)
	}
	return result, nil
}

func (g *GeminiProvider) Embed(text string) ([]float32, error) {
	ctx := context.Background()
	// Usamos gemini-embedding-001 que es el modelo actual disponible y soportado.
	em := g.Client.EmbeddingModel("gemini-embedding-001")
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("error generando embedding: %v", err)
	}
	
	if len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("el modelo devolvió un embedding vacío")
	}
	return res.Embedding.Values, nil
}

func (g *GeminiProvider) Close() error {
	// Mantiene una conexión persistente tipo gRPC,
	// es necesario cerrarla.
	if g.Client != nil {
		return g.Client.Close()
	}
	return nil
}
