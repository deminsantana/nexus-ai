package messaging

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/database"
	"nexus-core/internal/nlp"
	"strings"
)

// SenderFunc es el callback que cada proveedor pasa para enviar respuestas.
type SenderFunc = func(targetID, text string) error

// HandleIncomingMessage centraliza la lógica de negocio para todos los proveedores.
// Recibe el texto del mensaje, quien lo envía, dependencias, y un callback para responder.
// 'platform' indica la plataforma de origen (para el log en BD).
func HandleIncomingMessage(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg SenderFunc) {
	if msgText == "" {
		return
	}

	fmt.Printf("\n📩 [%s | %s]: %s\n", platform, pushName, msgText)

	// Persistencia en Postgres
	query := `INSERT INTO messages (source, sender_id, content, is_from_nexus) VALUES ($1, $2, $3, $4)`
	db.Exec(query, platform, senderStr, msgText, false)

	// Lógica de respuesta IA con filtro "nexus"
	if strings.HasPrefix(strings.ToLower(msgText), "nexus") {

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
			sendMsg(senderStr, "⛔ Has alcanzado el límite de mensajes automáticos asignados a tu cuenta. Nexus no puede procesar más solicitudes temporalmente.")
			return
		}

		fmt.Println("🧠 Procesando con IA...")
		cleanInput := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(msgText), "nexus"))

		reply, err := brain.ProcessMessageWithContext(senderStr, cleanInput)
		if err != nil {
			fmt.Printf("❌ Error IA: %v\n", err)
			return
		}

		err = sendMsg(senderStr, reply)
		if err == nil {
			fmt.Printf("🤖 Nexus dice: %s\n", reply)
			// 3. Contabilizar mensaje enviado en la BD
			database.IncrementQuota(db, senderStr)
		} else {
			fmt.Printf("❌ Error al enviar respuesta: %v\n", err)
		}
	}
}
