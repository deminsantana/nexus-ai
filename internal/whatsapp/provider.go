package whatsapp

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
)

type Provider interface {
	// Start inicializa el cliente (y/o el servidor web para webhooks)
	Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error
	
	// SendMessage envía un mensaje de texto plano a un objetivo
	// target puede ser JID o número de teléfono.
	SendMessage(target string, text string) error
}

func InitProvider(cfg *config.Config) (Provider, error) {
	if cfg.WhatsApp.Provider == "meta" {
		return &MetaProvider{
			token:         cfg.WhatsApp.Meta.Token,
			phoneNumberId: cfg.WhatsApp.Meta.PhoneNumberId,
			verifyToken:   cfg.WhatsApp.Meta.VerifyToken,
		}, nil
	}
	
	// Default Mau / whatsmeow
	if cfg.WhatsApp.Provider == "mau" || cfg.WhatsApp.Provider == "" {
		return &MauProvider{}, nil
	}

	return nil, fmt.Errorf("proveedor de WhatsApp desconocido: %s", cfg.WhatsApp.Provider)
}
