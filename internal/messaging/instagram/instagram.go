package instagram

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
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)) {
	handleMsg = h
}

// InstagramProvider implementa la interfaz messaging.Provider para Instagram DM via Meta Graph API.
// Requiere una cuenta de Instagram Business/Creator vinculada a una página de Facebook.
// Permisos necesarios: instagram_manage_messages, pages_messaging
type InstagramProvider struct {
	PageAccessToken string
	VerifyToken     string
	IGID            string
	db              *sql.DB
	brain           *nlp.Brain
}

func (p *InstagramProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	p.db = db
	p.brain = brain

	http.HandleFunc("/webhook/instagram", p.webhookHandler)

	fmt.Printf("✅ Nexus (Instagram): Webhook registrado en /webhook/instagram\n")

	return nil
}

func (p *InstagramProvider) webhookHandler(w http.ResponseWriter, r *http.Request) {
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

		p.processPayload(payload)
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (p *InstagramProvider) processPayload(payload map[string]interface{}) {
	entries, _ := payload["entry"].([]interface{})
	for _, entryRaw := range entries {
		entry, _ := entryRaw.(map[string]interface{})
		messagings, _ := entry["messaging"].([]interface{})
		for _, msgRaw := range messagings {
			msgEvent, _ := msgRaw.(map[string]interface{})

			sender, _ := msgEvent["sender"].(map[string]interface{})
			senderID, _ := sender["id"].(string)

			msgData, _ := msgEvent["message"].(map[string]interface{})
			if msgData == nil {
				continue
			}
			text, _ := msgData["text"].(string)
			if text == "" {
				continue
			}

			// Instagram no provee pushName en el webhook, usamos el IGSID
			pushName := senderID

			if handleMsg != nil {
				handleMsg("instagram", text, senderID, pushName, p.db, p.brain, func(targetID, replyText string) error {
					return p.SendMessage(targetID, replyText)
				}, func(targetID string, audioBytes []byte) error {
					return p.SendAudio(targetID, audioBytes)
				})
			}
		}
	}
}

// SendMessage envía un mensaje DM de Instagram.
// 'target' debe ser el IGSID (Instagram-Scoped ID) del destinatario.
func (p *InstagramProvider) SendMessage(target string, text string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", p.IGID)

	payload := map[string]interface{}{
		"recipient": map[string]string{"id": target},
		"message":   map[string]string{"text": text},
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
		return fmt.Errorf("error API Instagram [%d]: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendAudio envía un audio a Instagram. Actualmente placeholder.
func (p *InstagramProvider) SendAudio(target string, audioBytes []byte) error {
	fmt.Printf("🎙️ Instagram: Intento de envío de audio (%d bytes) a %s. Funcionalidad en desarrollo.\n", len(audioBytes), target)
	return nil
}
