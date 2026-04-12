package messaging

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/messaging/telegram"
	msgwa "nexus-core/internal/messaging/whatsapp"
	"nexus-core/internal/nlp"
)

// Provider define la interfaz genérica para cualquier plataforma de mensajería.
// Cada plataforma (WhatsApp, Telegram, Discord, etc.) implementa esta interfaz.
type Provider interface {
	// Start inicializa el cliente y/o servidor web para recibir mensajes.
	Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error

	// SendMessage envía un mensaje de texto a un destinatario.
	// El formato de 'target' depende de la plataforma:
	//   - WhatsApp Mau: JID (ej: "5841234567@s.whatsapp.net")
	//   - WhatsApp Meta: número de teléfono (ej: "5841234567")
	//   - Telegram: chat_id como string (ej: "123456789")
	SendMessage(target string, text string) error
}

// InitProvider detecta el proveedor configurado, inyecta el handler centralizado
// y retorna la implementación correcta.
func InitProvider(cfg *config.Config) (Provider, error) {
	// Inyectar el handler centralizado en todos los subpaquetes
	msgwa.SetHandler(HandleIncomingMessage)
	telegram.SetHandler(HandleIncomingMessage)

	switch cfg.Messaging.Provider {
	case "telegram":
		return &telegram.TelegramProvider{
			BotToken: cfg.Messaging.Telegram.BotToken,
		}, nil

	case "meta":
		return &msgwa.MetaProvider{
			Token:         cfg.Messaging.WhatsApp.Meta.Token,
			PhoneNumberId: cfg.Messaging.WhatsApp.Meta.PhoneNumberId,
			VerifyToken:   cfg.Messaging.WhatsApp.Meta.VerifyToken,
		}, nil

	case "mau", "":
		return &msgwa.MauProvider{}, nil

	default:
		return nil, fmt.Errorf("proveedor de mensajería desconocido: %q (opciones: mau, meta, telegram)", cfg.Messaging.Provider)
	}
}
