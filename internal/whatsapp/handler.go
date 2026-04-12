package whatsapp

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/database"
	"nexus-core/internal/nlp"
	"strings"
)

// SenderFunc es una firma para que el proveedor sepa cómo enviar un mensaje de vuelta
type SenderFunc func(targetID, text string) error

// HandleIncomingMessage unifica la lógica de negocio para cualquier proveedor.
// Recibe el texto, el identificador del usuario (senderStr), dependencias y una función para responder.
func HandleIncomingMessage(msgText string, senderStr string, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg SenderFunc) {
	if msgText == "" {
		return
	}

	// Usamos PushName para ver quién escribe en consola
	fmt.Printf("\n📩 [%s]: %s\n", pushName, msgText)

	// Persistencia en Postgres
	query := `INSERT INTO messages (source, sender_id, content, is_from_nexus) VALUES ($1, $2, $3, $4)`
	db.Exec(query, "whatsapp", senderStr, msgText, false)

	// Lógica de Respuesta IA con filtro "Nexus"
	if strings.HasPrefix(strings.ToLower(msgText), "nexus") {
		
		// 1. Validar Límite de Tasa por Redis (Spam de Segundos)
		if brain.IsRateLimited(senderStr) {
			fmt.Printf("⚠️ Usuario %s excedió el límite de tasa en Redis. Ignorando...\n", pushName)
			sendMsg(senderStr, "⚠️ Estás enviando mensajes muy rápido. Por favor, espera un momento.")
			return
		}

		// 2. Validar Cuota Mensual por Base de Datos
		canSend, err := database.CheckQuota(db, senderStr)
		if err != nil {
			fmt.Printf("❌ Error verificando cuota de BD para %s: %v\n", senderStr, err)
			return // Fallar de forma segura (no enviar)
		}

		if !canSend {
			fmt.Printf("⛔ Usuario %s excedió su límite de cuota (Rate Limit DB).\n", pushName)
			// Opcional: Enviar mensaje informando del límite
			sendMsg(senderStr, "⛔ Has alcanzado el límite de mensajes automáticos asignados a tu cuenta. Nexus no puede procesar más solicitudes temporalmente.")
			return
		}

		fmt.Println("🧠 Procesando con IA...")
		cleanInput := strings.TrimPrefix(strings.ToLower(msgText), "nexus")

		// Pasamos el senderID para que Redis sepa de quién es el contexto
		reply, err := brain.ProcessMessageWithContext(senderStr, cleanInput)
		if err != nil {
			fmt.Printf("❌ Error IA: %v\n", err)
			return
		}

		// Enviar respuesta al JID/Número original
		err = sendMsg(senderStr, reply)

		if err == nil {
			fmt.Printf("🤖 Nexus dice: %s\n", reply)
			// 3. Descontar/Contar mensaje enviado en la BD
			database.IncrementQuota(db, senderStr)
		} else {
			fmt.Printf("❌ Error al enviar respuesta: %v\n", err)
		}
	}
}
