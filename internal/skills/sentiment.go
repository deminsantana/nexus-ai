package skills

import (
	"context"
	"fmt"
	"nexus-core/internal/nlp"
	"strings"
)

// SentimentResult representa el resultado del análisis de sentimiento.
type SentimentResult struct {
	Label string // POSITIVO, NEGATIVO, NEUTRAL, MOLESTO, ENTUSIASTA
	Score float64
}

// AnalyzeSentiment detecta el sentimiento del mensaje del usuario.
func AnalyzeSentiment(ctx context.Context, brain *nlp.Brain, text string) (string, error) {
	prompt := fmt.Sprintf(`Analiza el sentimiento del siguiente mensaje de un usuario. 
Responde ÚNICAMENTE con una de estas palabras en mayúsculas: POSITIVO, NEGATIVO, NEUTRAL, MOLESTO, ENTUSIASTA.

Mensaje: "%s"

Sentimiento:`, text)

	result, err := brain.Provider.Ask(prompt)
	if err != nil {
		return "NEUTRAL", err
	}

	cleanResult := strings.ToUpper(strings.TrimSpace(result))
	// Validar que el resultado sea uno de los esperados
	validLabels := []string{"POSITIVO", "NEGATIVO", "NEUTRAL", "MOLESTO", "ENTUSIASTA"}
	for _, label := range validLabels {
		if strings.Contains(cleanResult, label) {
			return label, nil
		}
	}

	return "NEUTRAL", nil
}
