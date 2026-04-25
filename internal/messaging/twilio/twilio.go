package twilio

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strings"

	twilioApi "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)) {
	handleMsg = h
}

// TwilioProvider implementa la interfaz messaging.Provider para SMS via Twilio.
// Twilio envía los SMS entrantes a tu webhook como HTTP POST.
// Para desarrollo local, usa ngrok: ngrok http <webhook_port>
type TwilioProvider struct {
	AccountSID  string
	AuthToken   string
	FromNumber  string
	WebhookPort int
	client      *twilioApi.RestClient
	db          *sql.DB
	brain       *nlp.Brain
}

func (t *TwilioProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	t.db = db
	t.brain = brain

	// Inicializar cliente de Twilio
	t.client = twilioApi.NewRestClientWithParams(twilioApi.ClientParams{
		Username: t.AccountSID,
		Password: t.AuthToken,
	})

	// Puerto del webhook (independiente del servidor principal)
	port := t.WebhookPort
	if port == 0 {
		port = 18790 // puerto por defecto para el webhook de Twilio
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/sms", t.smsHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("✅ Nexus (Twilio): Servidor SMS webhook iniciado en %s/webhook/sms\n", addr)
	fmt.Printf("⚠️  Configura este URL en Twilio Console → Phone Numbers → Messaging Webhook\n")

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("❌ Error en servidor webhook de Twilio: %v\n", err)
		}
	}()

	return nil
}

// smsHandler recibe los SMS entrantes enviados por Twilio.
// Twilio envía: From, To, Body, MessageSid, etc. como application/x-www-form-urlencoded
func (t *TwilioProvider) smsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	from := r.FormValue("From")   // +1234567890
	body := r.FormValue("Body")   // texto del SMS

	if from == "" || body == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Formatear número como pushName legible
	pushName := from

	if handleMsg != nil {
		handleMsg("sms", body, from, pushName, t.db, t.brain, func(targetID, text string) error {
			return t.SendMessage(targetID, text)
		}, func(targetID string, audioBytes []byte) error {
			return t.SendAudio(targetID, audioBytes)
		})
	}

	// Twilio espera respuesta TwiML o 200 vacío
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response></Response>`))
}

// SendMessage envía un SMS via Twilio REST API.
// 'target' debe ser el número de teléfono en formato E.164 (ej: +584121234567).
func (t *TwilioProvider) SendMessage(target string, text string) error {
	if t.client == nil {
		return fmt.Errorf("cliente de Twilio no inicializado")
	}

	// Dividir mensajes largos (SMS tiene límite de 160 caracteres por segmento)
	chunks := splitSMS(text, 1600) // Twilio concatena hasta 10 partes
	for _, chunk := range chunks {
		params := &openapi.CreateMessageParams{}
		params.SetTo(target)
		params.SetFrom(t.FromNumber)
		params.SetBody(chunk)

		_, err := t.client.Api.CreateMessage(params)
		if err != nil {
			return fmt.Errorf("error enviando SMS via Twilio: %v", err)
		}
	}
	return nil
}

// SendAudio envía un audio via Twilio. Actualmente placeholder para SMS.
func (t *TwilioProvider) SendAudio(target string, audioBytes []byte) error {
	fmt.Printf("🎙️ Twilio SMS: Intento de envío de audio (%d bytes) a %s. Los SMS no soportan audio, usa Twilio Voice para llamadas.\n", len(audioBytes), target)
	return nil
}

// splitSMS divide un texto largo en fragmentos para SMS.
func splitSMS(text string, maxLen int) []string {
	_ = url.QueryEscape // suprimir import no usado
	if len(text) <= maxLen {
		return []string{text}
	}
	var chunks []string
	for len(text) > maxLen {
		chunks = append(chunks, text[:maxLen])
		text = text[maxLen:]
	}
	if len(strings.TrimSpace(text)) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}
