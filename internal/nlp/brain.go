package nlp

import (
	"context"
	"fmt"
	"nexus-core/internal/config"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Provider define el contrato que cualquier IA debe cumplir
type Provider interface {
	Ask(prompt string) (string, error)
	ProcessAudio(data []byte, mimeType string) (string, error)
	Close() error
}

// Brain es el orquestador que usa un proveedor específico
type Brain struct {
	Provider Provider
	RDB      *redis.Client
	Ctx      context.Context
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

	// Inicializar cliente de Redis (ajusta según tu config.yaml)
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Database.Host, 6380), // Puerto estándar Redis
	})

	return &Brain{
		Provider: p,
		RDB:      rdb,
		Ctx:      context.Background(),
	}, nil
}

func (b *Brain) ProcessMessage(text string) (string, error) {
	systemPrompt := "Eres Nexus, un asistente personal inteligente. Responde de forma breve y profesional."
	fullPrompt := fmt.Sprintf("%s\n\nUsuario: %s\nNexus:", systemPrompt, text)

	return b.Provider.Ask(fullPrompt)
}

func (b *Brain) ProcessMessageWithContext(senderID, text string) (string, error) {
	// 1. Obtener lo que hablamos hace poco
	pastTalk := b.GetContext(senderID)

	systemPrompt := "Eres Nexus, un asistente inteligente. " +
		"A continuación se muestra el contexto reciente de la conversación:\n" + pastTalk

	fullPrompt := fmt.Sprintf("%s\nUsuario actual: %s\nNexus:", systemPrompt, text)

	// 2. Preguntar a la IA
	reply, err := b.Provider.Ask(fullPrompt)
	if err != nil {
		return "", err
	}

	// 3. Guardar este intercambio en la memoria
	b.SaveContext(senderID, "Usuario: "+text)
	b.SaveContext(senderID, "Nexus: "+reply)

	return reply, nil
}

// GetContext recupera los últimos mensajes de un usuario
func (b *Brain) GetContext(senderID string) string {
	key := "ctx:" + senderID
	// Recuperamos la lista de mensajes (LRU)
	history, _ := b.RDB.LRange(b.Ctx, key, 0, 5).Result()

	// Redis guarda del más nuevo al más viejo, invertimos para la IA
	var sb strings.Builder
	for i := len(history) - 1; i >= 0; i-- {
		sb.WriteString(history[i] + "\n")
	}
	return sb.String()
}

// SaveContext guarda un nuevo mensaje y refresca el TTL
func (b *Brain) SaveContext(senderID, message string) {
	key := "ctx:" + senderID
	b.RDB.LPush(b.Ctx, key, message)
	b.RDB.LTrim(b.Ctx, key, 0, 10)          // Mantener solo los últimos 10
	b.RDB.Expire(b.Ctx, key, 5*time.Minute) // Expira en 5 minutos
}
