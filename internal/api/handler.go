package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
)

// Request es la estructura para la petición entrante al webhook de AI.
type Request struct {
	UserID  string                 `json:"user_id"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context"` // Datos adicionales para el futuro
}

// Response es la estructura de la respuesta de la IA.
type Response struct {
	Reply     string `json:"reply"`
	SessionID string `json:"session_id"`
}

// NewAIHandler crea un controlador para procesar mensajes genéricos con Nexus.
func NewAIHandler(brain *nlp.Brain, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validación básica de API Key
		if cfg.Server.APIKey != "" {
			apiKey := r.Header.Get("X-Nexus-API-Key")
			if apiKey != cfg.Server.APIKey {
				fmt.Printf("⚠️  [API] Intento de acceso no autorizado desde %s\n", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "API Key inválida o ausente"})
				return
			}
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "El campo 'message' es obligatorio", http.StatusBadRequest)
			return
		}

		// Si no hay user_id, usamos un identificador genérico
		userID := req.UserID
		if userID == "" {
			userID = "webhook_anonymous"
		}

		fmt.Printf("🌐 [Webhook API]: Mensaje de %s: %s\n", userID, req.Message)

		// Procesar con el cerebro de Nexus (IA + RAG + Memoria Redis)
		reply, err := brain.ProcessMessageWithContext(userID, req.Message)
		if err != nil {
			fmt.Printf("❌ Error API AI: %v\n", err)
			http.Error(w, "Error interno procesando IA", http.StatusInternalServerError)
			return
		}

		resp := Response{
			Reply:     reply,
			SessionID: userID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
