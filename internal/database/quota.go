package database

import (
	"database/sql"
	"fmt"
)

// CheckQuota verifica si un usuario puede enviar más mensajes.
// Inicializa la cuota en la BD si el usuario no existe.
func CheckQuota(db *sql.DB, senderID string) (bool, error) {
	var messagesSent, messageLimit int

	// Intentar obtener la cuota actual
	err := db.QueryRow("SELECT messages_sent, message_limit FROM usage_quotas WHERE sender_id = $1", senderID).Scan(&messagesSent, &messageLimit)
	
	if err == sql.ErrNoRows {
		// Inicializar usuario con límite de 1000 mensajes si no existe
		_, err := db.Exec("INSERT INTO usage_quotas (sender_id, messages_sent, message_limit) VALUES ($1, 0, 1000)", senderID)
		if err != nil {
			return false, fmt.Errorf("error al inicializar cuota de uso: %v", err)
		}
		return true, nil // Recién creado, tiene cuota
	} else if err != nil {
		return false, fmt.Errorf("error obteniendo cuota de uso: %v", err)
	}

	// Puede enviar el mensaje si no ha excedido su límite
	return messagesSent < messageLimit, nil
}

// IncrementQuota suma 1 contador de mensaje a la cuota del usuario
func IncrementQuota(db *sql.DB, senderID string) error {
	_, err := db.Exec("UPDATE usage_quotas SET messages_sent = messages_sent + 1, updated_at = CURRENT_TIMESTAMP WHERE sender_id = $1", senderID)
	return err
}
