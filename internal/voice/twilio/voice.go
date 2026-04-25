// Package twilio implementa el VoiceProvider usando Twilio Voice API.
// Permite hacer llamadas telefónicas outbound donde un TTS lee un mensaje.
//
// Para llamadas outbound, Twilio necesita un TwiML que describa qué decir.
// Opciones:
//   - TwiML Bin (URL pública que devuelve XML con <Say>)
//   - URL de tu propio servidor webhook que devuelve TwiML dinámico
//
// Este provider usa el enfoque de TwiML dinámico: levanta un endpoint
// /voice/twiml en el servidor HTTP de Nexus para generar TwiML on-the-fly.
package twilio

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	twilioApi "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioVoiceProvider implementa voice.Provider usando Twilio Voice API.
type TwilioVoiceProvider struct {
	accountSID  string
	authToken   string
	fromNumber  string
	twimlBinURL string // URL TwiML externo (opcional)

	client *twilioApi.RestClient

	// pendingMessages guarda el mensaje TTS pendiente para la próxima llamada.
	// Se usa cuando no hay twimlBinURL externo y se sirve el TwiML localmente.
	mu              sync.Mutex
	pendingMessages map[string]string // callSID → message (temporal)
	lastMessage     string            // último mensaje para /voice/twiml
}

// NewVoiceProvider crea un TwilioVoiceProvider listo para usar.
func NewVoiceProvider(accountSID, authToken, fromNumber, twimlBinURL string) *TwilioVoiceProvider {
	client := twilioApi.NewRestClientWithParams(twilioApi.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &TwilioVoiceProvider{
		accountSID:      accountSID,
		authToken:       authToken,
		fromNumber:      fromNumber,
		twimlBinURL:     twimlBinURL,
		client:          client,
		pendingMessages: make(map[string]string),
	}
}

// TextToSpeech convierte texto en audio usando Twilio.
// Nota: Twilio no provee TTS como bytes directamente; esto hace MakeCall.
// Para audio bytes, usa el proveedor Google.
func (t *TwilioVoiceProvider) TextToSpeech(text, lang string) ([]byte, error) {
	return nil, fmt.Errorf("twilio voice provider no soporta TextToSpeech como bytes; usa MakeCall para llamadas o el proveedor google para audio")
}

// MakeCall inicia una llamada telefónica outbound.
// Twilio llama a 'to', y al contestar reproduce el mensaje via TTS.
// 'to' debe estar en formato E.164 (ej: "+584121234567").
func (t *TwilioVoiceProvider) MakeCall(to, message string) error {
	if t.client == nil {
		return fmt.Errorf("cliente Twilio no inicializado")
	}

	// Determinar la URL del TwiML
	twimlURL := t.twimlBinURL
	if twimlURL == "" {
		// Generar TwiML dinámico directamente (sin URL externa)
		// Twilio requiere una URL pública; si no hay, usamos URL codificada
		// alternativa: TwiML como string inline con Url parámetro no soportado.
		// La mejor opción real es proveer un twiml_bin_url en config.
		// Como fallback, generamos el TwiML y lo codificamos en URL de data (no soportado por Twilio).
		// En producción SIEMPRE configura twiml_bin_url o el webhook del scheduler.
		return fmt.Errorf("se requiere 'voice.twilio.twiml_bin_url' en config.yaml para llamadas outbound. Crea un TwiML Bin en https://www.twilio.com/console/twiml-bins con: <Response><Say language=\"es-MX\">%s</Say></Response>", message)
	}

	// Codificar el mensaje en la URL del TwiML bin como query param (si el bin lo soporta)
	fullURL := twimlURL
	if !strings.Contains(twimlURL, "?") {
		encodedMsg := url.QueryEscape(message)
		fullURL = fmt.Sprintf("%s?message=%s", twimlURL, encodedMsg)
	}

	params := &openapi.CreateCallParams{}
	params.SetTo(to)
	params.SetFrom(t.fromNumber)
	params.SetUrl(fullURL)

	call, err := t.client.Api.CreateCall(params)
	if err != nil {
		return fmt.Errorf("error iniciando llamada Twilio a %s: %v", to, err)
	}

	fmt.Printf("📞 Llamada iniciada → %s | SID: %s\n", to, *call.Sid)
	return nil
}

// TwiMLResponse es la estructura XML para respuestas TwiML.
type TwiMLResponse struct {
	XMLName xml.Name `xml:"Response"`
	Say     TwiMLSay `xml:"Say"`
}

type TwiMLSay struct {
	Language string `xml:"language,attr"`
	Voice    string `xml:"voice,attr"`
	Text     string `xml:",chardata"`
}

// ServeLocalTwiML devuelve un http.HandlerFunc que genera TwiML dinámico.
// Úsalo en el servidor HTTP de Nexus: mux.HandleFunc("/voice/twiml", provider.ServeLocalTwiML())
func (t *TwilioVoiceProvider) ServeLocalTwiML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		message := r.URL.Query().Get("message")
		if message == "" {
			t.mu.Lock()
			message = t.lastMessage
			t.mu.Unlock()
		}
		if message == "" {
			message = "Hola, soy Nexus, tu asistente inteligente."
		}

		resp := TwiMLResponse{
			Say: TwiMLSay{
				Language: "es-MX",
				Voice:    "Polly.Lupe",
				Text:     message,
			},
		}

		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}
