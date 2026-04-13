package messenger

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)) {
	handleMsg = h
}

// MessengerProvider implementa la interfaz messaging.Provider para Facebook Messenger via Meta Graph API.
// Permisos necesarios: pages_messaging, pages_read_engagement
type MessengerProvider struct {
	PageAccessToken string
	VerifyToken     string
	PageID          string
	db              *sql.DB
	brain           *nlp.Brain
}

func (p *MessengerProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	p.db = db
	p.brain = brain

	http.HandleFunc("/webhook/messenger", p.webhookHandler)

	fmt.Printf("✅ Nexus (Messenger): Webhook registrado en /webhook/messenger\n")

	return nil
}

func (p *MessengerProvider) webhookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Verificación de webhook de Meta
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == p.VerifyToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		w.WriteHeader(http.StatusForbidden)

	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verificar que es página de Messenger (object = "page")
		if obj, _ := payload["object"].(string); obj != "page" {
			w.WriteHeader(http.StatusOK)
			return
		}

		p.processPayload(payload)
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (p *MessengerProvider) processPayload(payload map[string]interface{}) {
	entries, _ := payload["entry"].([]interface{})
	for _, entryRaw := range entries {
		entry, _ := entryRaw.(map[string]interface{})
		messagings, _ := entry["messaging"].([]interface{})
		for _, msgRaw := range messagings {
			msgEvent, _ := msgRaw.(map[string]interface{})

			sender, _ := msgEvent["sender"].(map[string]interface{})
			senderID, _ := sender["id"].(string)

			// Ignorar mensajes de la propia página
			if senderID == p.PageID {
				continue
			}

			msgData, _ := msgEvent["message"].(map[string]interface{})
			if msgData == nil {
				continue
			}

			// Ignorar mensajes eco (enviados por la página)
			if isEcho, _ := msgData["is_echo"].(bool); isEcho {
				continue
			}

			text, _ := msgData["text"].(string)
			if text == "" {
				continue
			}

			// Intentar obtener el nombre del usuario via Graph API
			pushName := p.getUserName(senderID)

			if handleMsg != nil {
				handleMsg("messenger", text, senderID, pushName, p.db, p.brain, func(targetID, replyText string) error {
					return p.SendMessage(targetID, replyText)
				})
			}
		}
	}
}

// getUserName obtiene el nombre del usuario via Meta Graph API.
func (p *MessengerProvider) getUserName(userID string) string {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=name&access_token=%s", userID, p.PageAccessToken)
	resp, err := http.Get(url)
	if err != nil {
		return userID
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return userID
	}

	if name, ok := result["name"].(string); ok {
		return name
	}
	return userID
}

// SendMessage envía un mensaje via Facebook Messenger.
// 'target' debe ser el PSID (Page-Scoped ID) del destinatario.
func (p *MessengerProvider) SendMessage(target string, text string) error {
	url := "https://graph.facebook.com/v19.0/me/messages"

	payload := map[string]interface{}{
		"recipient":      map[string]string{"id": target},
		"message":        map[string]string{"text": text},
		"messaging_type": "RESPONSE",
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.PageAccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error API Messenger [%d]: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
