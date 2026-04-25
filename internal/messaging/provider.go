package messaging

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/messaging/discord"
	"nexus-core/internal/messaging/email"
	"nexus-core/internal/messaging/instagram"
	"nexus-core/internal/messaging/messenger"
	"nexus-core/internal/messaging/slack"
	"nexus-core/internal/messaging/telegram"
	"nexus-core/internal/messaging/twilio"
	msgwa "nexus-core/internal/messaging/whatsapp"
	"nexus-core/internal/nlp"
)

// Provider define la interfaz genérica para cualquier plataforma de mensajería.
// Cada plataforma (WhatsApp, Telegram, Discord, etc.) implementa esta interfaz.
type Provider interface {
	// Start inicializa el cliente y/o servidor web para recibir mensajes.
	Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error

	// SendMessage envía un mensaje de texto a un destinatario.
	SendMessage(target string, text string) error

	// SendAudio envía un archivo de audio (nota de voz).
	SendAudio(target string, audioBytes []byte) error
}

// InitProvider detecta el proveedor configurado, inyecta el handler centralizado
// y retorna la implementación correcta.
func InitProvider(cfg *config.Config) (Provider, error) {
	// Inyectar el handler centralizado en todos los subpaquetes
	msgwa.SetHandler(HandleIncomingMessage)
	telegram.SetHandler(HandleIncomingMessage)
	discord.SetHandler(HandleIncomingMessage)
	slack.SetHandler(HandleIncomingMessage)
	instagram.SetHandler(HandleIncomingMessage)
	messenger.SetHandler(HandleIncomingMessage)
	twilio.SetHandler(HandleIncomingMessage)
	email.SetHandler(HandleIncomingMessage)

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

	case "discord":
		return &discord.DiscordProvider{
			BotToken: cfg.Messaging.Discord.BotToken,
			GuildID:  cfg.Messaging.Discord.GuildID,
		}, nil

	case "slack":
		return &slack.SlackProvider{
			BotToken:      cfg.Messaging.Slack.BotToken,
			AppToken:      cfg.Messaging.Slack.AppToken,
			SigningSecret: cfg.Messaging.Slack.SigningSecret,
		}, nil

	case "instagram":
		return &instagram.InstagramProvider{
			PageAccessToken: cfg.Messaging.Instagram.PageAccessToken,
			VerifyToken:     cfg.Messaging.Instagram.VerifyToken,
			IGID:            cfg.Messaging.Instagram.IGID,
		}, nil

	case "messenger":
		return &messenger.MessengerProvider{
			PageAccessToken: cfg.Messaging.Messenger.PageAccessToken,
			VerifyToken:     cfg.Messaging.Messenger.VerifyToken,
			PageID:          cfg.Messaging.Messenger.PageID,
		}, nil

	case "twilio":
		return &twilio.TwilioProvider{
			AccountSID:  cfg.Messaging.Twilio.AccountSID,
			AuthToken:   cfg.Messaging.Twilio.AuthToken,
			FromNumber:  cfg.Messaging.Twilio.FromNumber,
			WebhookPort: cfg.Messaging.Twilio.WebhookPort,
		}, nil

	case "email":
		return &email.EmailProvider{
			IMAPHost:     cfg.Messaging.Email.IMAPHost,
			IMAPPort:     cfg.Messaging.Email.IMAPPort,
			SMTPHost:     cfg.Messaging.Email.SMTPHost,
			SMTPPort:     cfg.Messaging.Email.SMTPPort,
			User:         cfg.Messaging.Email.User,
			Password:     cfg.Messaging.Email.Password,
			PollInterval: cfg.Messaging.Email.PollInterval,
		}, nil

	case "mau", "":
		return &msgwa.MauProvider{}, nil

	default:
		return nil, fmt.Errorf("proveedor de mensajería desconocido: %q (opciones: mau, meta, telegram, discord, slack, instagram, messenger, twilio, email)", cfg.Messaging.Provider)
	}
}
