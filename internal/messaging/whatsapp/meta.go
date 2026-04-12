package whatsapp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strings"
)

type MetaProvider struct {
	Token         string
	PhoneNumberId string
	VerifyToken   string
	db            *sql.DB
	brain         *nlp.Brain
}

func (m *MetaProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	m.db = db
	m.brain = brain

	http.HandleFunc("/webhook", m.webhookHandler)

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("✅ Nexus (Meta): Iniciando servidor web para Webhooks en %s\n", port)

	go func() {
		err := http.ListenAndServe(port, nil)
		if err != nil {
			fmt.Printf("❌ Error en Servidor Webhook Meta: %v\n", err)
		}
	}()

	return nil
}

func (m *MetaProvider) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode != "" && token != "" {
			if mode == "subscribe" && token == m.VerifyToken {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}
	} else if r.Method == "POST" {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)

		if entries, ok := payload["entry"].([]interface{}); ok {
			for _, entry := range entries {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					if changes, ok := entryMap["changes"].([]interface{}); ok {
						for _, change := range changes {
							if changeMap, ok := change.(map[string]interface{}); ok {
								if valueMap, ok := changeMap["value"].(map[string]interface{}); ok {
									if messages, ok := valueMap["messages"].([]interface{}); ok {
										for _, msgInterface := range messages {
											m.processWebhookMessage(msgInterface.(map[string]interface{}), valueMap)
										}
									}
								}
							}
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (m *MetaProvider) processWebhookMessage(msg map[string]interface{}, valueMap map[string]interface{}) {
	if msgType, _ := msg["type"].(string); msgType != "text" {
		return
	}

	from, _ := msg["from"].(string)

	pushName := from
	if contacts, ok := valueMap["contacts"].([]interface{}); ok && len(contacts) > 0 {
		if contact, ok := contacts[0].(map[string]interface{}); ok {
			if profile, ok := contact["profile"].(map[string]interface{}); ok {
				if name, ok := profile["name"].(string); ok {
					pushName = name
				}
			}
		}
	}

	textMap, ok := msg["text"].(map[string]interface{})
	if !ok {
		return
	}
	msgText, _ := textMap["body"].(string)

	if handleMsg != nil {
		handleMsg("whatsapp_meta", msgText, from, pushName, m.db, m.brain, func(targetID, text string) error {
			return m.SendMessage(targetID, text)
		})
	}
}

func (m *MetaProvider) SendMessage(target string, text string) error {
	idx := strings.Index(target, "@")
	if idx != -1 {
		target = target[:idx]
	}

	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", m.PhoneNumberId)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                target,
		"type":              "text",
		"text": map[string]interface{}{
			"preview_url": false,
			"body":        text,
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+m.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error API Meta: [%d] %s", resp.StatusCode, string(respBody))
	}

	return nil
}
