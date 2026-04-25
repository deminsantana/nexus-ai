package messaging

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/agent"
	"nexus-core/internal/config"
	"nexus-core/internal/database"
	"nexus-core/internal/nlp"
	"nexus-core/internal/voice"
	"strings"
)

// SenderFunc es el callback que cada proveedor pasa para enviar respuestas.
type SenderFunc = func(targetID, text string) error

// AudioSenderFunc es el callback para enviar notas de voz.
type AudioSenderFunc = func(targetID string, audioBytes []byte) error

// globalCfg almacena la configuración global para acceder al sales_agent en el handler.
var globalCfg *config.Config

// globalSalesAgent es la instancia singleton del agente de ventas (nil si no está activado).
var globalSalesAgent *agent.SalesAgent

// globalVoiceProvider es el proveedor de voz activo para respuestas TTS.
var globalVoiceProvider voice.Provider

// SetConfig inyecta la configuración global y prepara los agentes.
func SetConfig(cfg *config.Config, brain *nlp.Brain, vp voice.Provider) {
	globalCfg = cfg
	globalVoiceProvider = vp
	if cfg.SalesAgent.Enabled {
		globalSalesAgent = agent.NewSalesAgent(cfg.SalesAgent, brain.RDB)
		fmt.Printf("🤝 Sales Agent FSM activado para producto: %s\n", cfg.SalesAgent.ProductName)
	}
}

// HandleIncomingMessage centraliza la lógica de negocio para todos los proveedores.
// Recibe el texto del mensaje, quien lo envía, dependencias, y un callback para responder.
// 'platform' indica la plataforma de origen (para el log en BD).
//
// Flujo de decisión:
//  1. Si sales_agent.enabled → usar FSM de ventas (responde a TODOS los mensajes)
//  2. Si no → flujo original: solo responde si el mensaje empieza con "nexus"
func HandleIncomingMessage(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg SenderFunc, sendAudio AudioSenderFunc) {
	if msgText == "" {
		return
	}

	fmt.Printf("\n📩 [%s | %s]: %s\n", platform, pushName, msgText)

	// Persistencia en Postgres
	query := `INSERT INTO messages (source, sender_id, content, is_from_nexus) VALUES ($1, $2, $3, $4)`
	db.Exec(query, platform, senderStr, msgText, false)

	// ── MODO SALES AGENT FSM ──────────────────────────────────────────────────
	if globalSalesAgent != nil {
		// 1. Validar límite de tasa por Redis (anti-spam)
		if brain.IsRateLimited(senderStr) {
			fmt.Printf("⚠️ Usuario %s excedió el límite de tasa en Redis. Ignorando...\n", pushName)
			sendMsg(senderStr, "⚠️ Estás enviando mensajes muy rápido. Por favor, espera un momento.")
			return
		}

		// 2. Validar cuota mensual por base de datos
		canSend, err := database.CheckQuota(db, senderStr)
		if err != nil {
			fmt.Printf("❌ Error verificando cuota de BD para %s: %v\n", senderStr, err)
			return
		}
		if !canSend {
			fmt.Printf("⛔ Usuario %s excedió su límite de cuota (Rate Limit DB).\n", pushName)
			sendMsg(senderStr, "⛔ Has alcanzado el límite de mensajes. Por favor contáctanos directamente.")
			return
		}

		// 3. Procesar con FSM de ventas
		reply, err := globalSalesAgent.ProcessWithFSM(senderStr, msgText, brain)
		if err != nil {
			fmt.Printf("❌ Error en Sales FSM: %v\n", err)
			return
		}

		if err := sendMsg(senderStr, reply); err == nil {
			database.IncrementQuota(db, senderStr)
			
			// OPCIONAL: Si hay voz activa, podríamos enviar el audio aquí también.
			// Por ahora, el FSM se mantiene principalmente en texto por velocidad.
		} else {
			fmt.Printf("❌ Error al enviar respuesta FSM: %v\n", err)
		}
		return
	}

	// ── MODO NORMAL (trigger "nexus") ─────────────────────────────────────────
	if strings.HasPrefix(strings.ToLower(msgText), "nexus") {
		// ... validaciones de rate limit y cuota ...
		if brain.IsRateLimited(senderStr) {
			sendMsg(senderStr, "⚠️ Estás enviando mensajes muy rápido.")
			return
		}
		
		canSend, _ := database.CheckQuota(db, senderStr)
		if !canSend {
			sendMsg(senderStr, "⛔ Has alcanzado tu límite de mensajes.")
			return
		}

		fmt.Println("🧠 Procesando con IA...")
		cleanInput := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(msgText), "nexus"))

		reply, err := brain.ProcessMessageWithContext(senderStr, cleanInput)
		if err != nil {
			fmt.Printf("❌ Error IA: %v\n", err)
			return
		}

		// Decidir qué enviar según ResponseMode
		mode := strings.ToLower(globalCfg.Voice.ResponseMode)
		if mode == "" {
			mode = "text"
		}

		fmt.Printf("🔍 DEBUG: Mode='%s', VoiceProviderOK=%v, ProviderCfg='%s'\n", 
			mode, globalVoiceProvider != nil, globalCfg.Voice.Provider)

		// 1. Enviar Texto (si el modo es 'text' o 'both')
		if mode == "text" || mode == "both" {
			err = sendMsg(senderStr, reply)
		}

		// 2. Enviar Voz (si el modo es 'voice' o 'both' Y hay proveedor de voz activo)
		if mode == "voice" || mode == "both" {
			if globalVoiceProvider != nil && (globalCfg.Voice.Provider == "google" || globalCfg.Voice.Provider == "twilio") {
				fmt.Println("🎙️ Generando respuesta de voz...")
				audio, vErr := globalVoiceProvider.TextToSpeech(reply, "")
				if vErr == nil {
					fmt.Printf("🔊 Voz generada (%d bytes). Enviando...\n", len(audio))
					if errAudio := sendAudio(senderStr, audio); errAudio != nil {
						fmt.Printf("❌ Error enviando audio a la plataforma: %v\n", errAudio)
					}
				} else {
					fmt.Printf("❌ Error en Voice Provider: %v. Aplicando fallback a texto.\n", vErr)
					if mode == "voice" {
						sendMsg(senderStr, reply)
					}
				}
			} else {
				fmt.Printf("⚠️ No se puede enviar voz. Provider=%s, ProviderNil=%v\n", globalCfg.Voice.Provider, globalVoiceProvider == nil)
				if mode == "voice" {
					sendMsg(senderStr, reply)
				}
			}
		}

		if err == nil {
			fmt.Printf("🤖 Nexus dice: %s\n", reply)
			database.IncrementQuota(db, senderStr)
		}
	}
}
