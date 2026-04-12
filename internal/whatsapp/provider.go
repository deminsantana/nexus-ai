// Package whatsapp (legacy) es mantenido por retrocompatibilidad.
// El nuevo punto de entrada es internal/messaging.
package whatsapp

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
)

// Provider define la interfaz de un proveedor de mensajería.
type Provider interface {
	Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error
	SendMessage(target string, text string) error
}

// InitProvider mantiene retrocompatibilidad con el paquete legacy.
// Usa el campo Messaging de la nueva config.
func InitProvider(cfg *config.Config) (Provider, error) {
	if cfg.Messaging.Provider == "meta" {
		return &MetaProvider{
			Token:         cfg.Messaging.WhatsApp.Meta.Token,
			PhoneNumberId: cfg.Messaging.WhatsApp.Meta.PhoneNumberId,
			VerifyToken:   cfg.Messaging.WhatsApp.Meta.VerifyToken,
		}, nil
	}

	// Default: Mau / whatsmeow
	if cfg.Messaging.Provider == "mau" || cfg.Messaging.Provider == "" {
		return &MauProvider{}, nil
	}

	return nil, fmt.Errorf("proveedor desconocido: %s", cfg.Messaging.Provider)
}
