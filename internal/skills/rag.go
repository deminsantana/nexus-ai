package skills

import (
	"context"
	"fmt"
	"nexus-core/internal/nlp"
)

// RetrieveKnowledge busca información relevante en la base de datos vectorial filtrando por tag si se especifica.
func RetrieveKnowledge(ctx context.Context, brain *nlp.Brain, query string, tag string) (string, error) {
	// Buscamos los 3 fragmentos más relevantes filtrando por tag
	context, err := brain.SearchKnowledgeBase(query, 3, tag)
	if err != nil {
		return "", fmt.Errorf("error recuperando conocimiento: %v", err)
	}

	if context == "" {
		return "", nil
	}

	return fmt.Sprintf("\n--- INFORMACIÓN DE RESPALDO (Usa esto para responder) ---\n%s\n------------------------------------------------------\n", context), nil
}
