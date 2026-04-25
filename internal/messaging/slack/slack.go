package slack

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)) {
	handleMsg = h
}

// SlackProvider implementa la interfaz messaging.Provider para Slack via Socket Mode.
// Socket Mode no requiere URL pública: usa WebSocket para recibir eventos.
type SlackProvider struct {
	BotToken      string
	AppToken      string // xapp-... requerido para Socket Mode
	SigningSecret string
	client        *slack.Client
	db            *sql.DB
	brain         *nlp.Brain
}

func (s *SlackProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	s.db = db
	s.brain = brain

	// Crear cliente de Slack con el App Token para Socket Mode
	s.client = slack.New(
		s.BotToken,
		slack.OptionAppLevelToken(s.AppToken),
	)

	// Obtener info del bot para saber su ID
	authResp, err := s.client.AuthTest()
	if err != nil {
		return fmt.Errorf("❌ Error autenticando con Slack: %v", err)
	}
	botID := authResp.UserID
	fmt.Printf("✅ Nexus (Slack): Bot conectado como @%s (ID: %s)\n", authResp.User, botID)

	// Crear cliente de Socket Mode
	smClient := socketmode.New(s.client)

	go func() {
		for evt := range smClient.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				smClient.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						// Ignorar mensajes del propio bot o de bots
						if ev.User == "" || ev.BotID != "" || ev.SubType == "bot_message" {
							continue
						}
						if ev.User == botID {
							continue
						}

						channelID := ev.Channel
						userID := ev.User
						msgText := ev.Text

						// Resolver nombre del usuario
						pushName := userID
						if userInfo, err := s.client.GetUserInfo(userID); err == nil {
							pushName = userInfo.Profile.DisplayName
							if pushName == "" {
								pushName = userInfo.Profile.RealName
							}
						}

						if handleMsg != nil {
							handleMsg("slack", msgText, userID, pushName, db, brain, func(targetID, text string) error {
								return s.sendToChannel(channelID, text)
							}, func(targetID string, audioBytes []byte) error {
								return s.SendAudio(channelID, audioBytes)
							})
						}
					}
				}
			}
		}
	}()

	// Iniciar Socket Mode en goroutine
	go func() {
		if err := smClient.Run(); err != nil {
			fmt.Printf("❌ Error en Socket Mode de Slack: %v\n", err)
		}
	}()

	return nil
}

// SendMessage envía un mensaje a un canal o usuario de Slack.
// 'target' debe ser el channelID (C...) o userID (U...) como string.
func (s *SlackProvider) SendMessage(target string, text string) error {
	return s.sendToChannel(target, text)
}

func (s *SlackProvider) sendToChannel(channelID, text string) error {
	if s.client == nil {
		return fmt.Errorf("cliente de Slack no inicializado")
	}
	_, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(text, false))
	return err
}

// SendAudio envía un audio a Slack. Actualmente placeholder.
func (s *SlackProvider) SendAudio(target string, audioBytes []byte) error {
	fmt.Printf("🎙️ Slack: Intento de envío de audio (%d bytes) a %s. Funcionalidad en desarrollo.\n", len(audioBytes), target)
	return nil
}
