package telegram

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strconv"
	"time"

	tele "gopkg.in/telebot.v3"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)) {
	handleMsg = h
}

// TelegramProvider implementa la interfaz messaging.Provider para Telegram Bot API.
type TelegramProvider struct {
	BotToken string
	bot      *tele.Bot
}

func (t *TelegramProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	pref := tele.Settings{
		Token:  t.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return fmt.Errorf("❌ Error iniciando bot de Telegram: %v", err)
	}
	t.bot = bot

	fmt.Printf("✅ Nexus (Telegram): Bot conectado como @%s\n", bot.Me.Username)

	// Handler para todos los mensajes de texto
	bot.Handle(tele.OnText, func(c tele.Context) error {
		msg := c.Message()
		senderID := strconv.FormatInt(msg.Sender.ID, 10)
		pushName := msg.Sender.FirstName
		if msg.Sender.LastName != "" {
			pushName += " " + msg.Sender.LastName
		}
		if msg.Sender.Username != "" {
			pushName += " (@" + msg.Sender.Username + ")"
		}

		chatID := strconv.FormatInt(msg.Chat.ID, 10)

		if handleMsg != nil {
			handleMsg("telegram", msg.Text, senderID, pushName, db, brain, func(targetID, text string) error {
				return t.sendToChat(chatID, text)
			})
		}
		return nil
	})

	// Iniciar el bot en una goroutine para no bloquear
	go bot.Start()

	return nil
}

// SendMessage envía un mensaje a un chat de Telegram.
// 'target' debe ser el chat_id como string (ej: "123456789").
func (t *TelegramProvider) SendMessage(target string, text string) error {
	return t.sendToChat(target, text)
}

func (t *TelegramProvider) sendToChat(chatID string, text string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("chat_id inválido %q: %v", chatID, err)
	}

	chat := &tele.Chat{ID: id}
	_, err = t.bot.Send(chat, text)
	return err
}
