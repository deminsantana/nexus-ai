package voice

import (
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/voice/google"
	"nexus-core/internal/voice/twilio"
)

// Provider define el contrato para cualquier proveedor de voz.
// Permite TTS (texto a voz), STT (voz a texto) y llamadas outbound.
type Provider interface {
	// TextToSpeech genera audio MP3/OGG desde un texto dado.
	TextToSpeech(text, lang string) ([]byte, error)

	// MakeCall inicia una llamada telefónica outbound con TTS.
	// 'to' debe ser un número E.164 (ej: "+584121234567").
	MakeCall(to, message string) error
}

// InitProvider devuelve el VoiceProvider configurado en config.yaml.
// Retorna nil, nil si provider = "none" (voz desactivada).
func InitProvider(cfg *config.Config) (Provider, error) {
	switch cfg.Voice.Provider {
	case "twilio":
		return twilio.NewVoiceProvider(
			cfg.Voice.Twilio.AccountSID,
			cfg.Voice.Twilio.AuthToken,
			cfg.Voice.Twilio.FromNumber,
			cfg.Voice.Twilio.TwiMLBinURL,
		), nil

	case "google":
		return google.NewVoiceProvider(
			cfg.Google.CredentialsFile,
			cfg.Google.Language,
			cfg.Google.VoiceName,
			cfg.Google.Gender,
			cfg.Google.Pitch,
			cfg.Google.SpeakingRate,
		)

	case "none", "":
		return nil, nil // Voice desactivado

	default:
		return nil, fmt.Errorf("proveedor de voz desconocido: %q (opciones: twilio, google, none)", cfg.Voice.Provider)
	}
}
