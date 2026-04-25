// Package google implementa el VoiceProvider usando Google Cloud Text-to-Speech.
// Genera archivos de audio MP3 directamente desde texto, sin necesidad de llamadas.
// Ideal para enviar notas de voz en WhatsApp o Telegram.
//
// Requiere credenciales de Google Cloud en JSON.
// Activar la API en: https://console.cloud.google.com/apis/library/texttospeech.googleapis.com
//
// Si no tienes credenciales GCP, usa el proveedor "twilio" para llamadas reales.
package google

import (
	"context"
	"fmt"
	"os"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
)

// GoogleVoiceProvider implementa voice.Provider usando Google Cloud TTS.
type GoogleVoiceProvider struct {
	credentialsFile string
	language        string
	voiceName       string
	gender          string
	pitch           float64
	speakingRate    float64
}

// NewVoiceProvider crea un GoogleVoiceProvider.
// Si credentialsFile está vacío, usa Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS).
func NewVoiceProvider(credentialsFile, language, voiceName, gender string, pitch, speakingRate float64) (*GoogleVoiceProvider, error) {
	if language == "" {
		language = "es-ES"
	}
	if voiceName == "" {
		voiceName = "es-ES-Standard-A"
	}
	if speakingRate == 0 {
		speakingRate = 1.0
	}

	// Validar que el archivo de credenciales existe si se proporcionó
	if credentialsFile != "" {
		if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("archivo de credenciales GCP no encontrado: %s", credentialsFile)
		}
	}

	return &GoogleVoiceProvider{
		credentialsFile: credentialsFile,
		language:        language,
		voiceName:       voiceName,
		gender:          gender,
		pitch:           pitch,
		speakingRate:    speakingRate,
	}, nil
}

// TextToSpeech convierte texto en audio MP3 usando Google Cloud TTS.
// Devuelve los bytes del archivo MP3 para enviar como nota de voz.
func (g *GoogleVoiceProvider) TextToSpeech(text, lang string) ([]byte, error) {
	ctx := context.Background()

	// Construir opciones de cliente
	var opts []option.ClientOption
	if g.credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(g.credentialsFile))
	}

	client, err := texttospeech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente Google TTS: %v\n💡 Activa la API en: https://console.cloud.google.com/apis/library/texttospeech.googleapis.com", err)
	}
	defer client.Close()

	// Usar el idioma del parámetro o el configurado por defecto
	language := lang
	if language == "" {
		language = g.language
	}

	// Mapear género
	gender := texttospeechpb.SsmlVoiceGender_NEUTRAL
	switch g.gender {
	case "male":
		gender = texttospeechpb.SsmlVoiceGender_MALE
	case "female":
		gender = texttospeechpb.SsmlVoiceGender_FEMALE
	}

	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: language,
			Name:         g.voiceName,
			SsmlGender:   gender,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_OGG_OPUS, // OGG_OPUS = formato nativo de notas de voz en WhatsApp
			SpeakingRate:  g.speakingRate,
			Pitch:         g.pitch,
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error sintetizando voz con Google TTS: %v", err)
	}

	return resp.AudioContent, nil
}

// MakeCall no está soportado en el proveedor Google TTS.
// Para llamadas telefónicas reales, usa el proveedor "twilio".
func (g *GoogleVoiceProvider) MakeCall(to, message string) error {
	return fmt.Errorf("google TTS no soporta llamadas telefónicas directas; usa el proveedor 'twilio' para llamadas outbound")
}
