package discord

import (
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"

	"github.com/bwmarrin/discordgo"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)) {
	handleMsg = h
}

// DiscordProvider implementa la interfaz messaging.Provider para Discord Bot Gateway.
type DiscordProvider struct {
	BotToken string
	GuildID  string
	session  *discordgo.Session
	db       *sql.DB
	brain    *nlp.Brain
}

func (d *DiscordProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	d.db = db
	d.brain = brain

	dg, err := discordgo.New("Bot " + d.BotToken)
	if err != nil {
		return fmt.Errorf("❌ Error creando sesión de Discord: %v", err)
	}
	d.session = dg

	// Necesitamos el intent de mensajes de guilds y DMs
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	// Registrar handler de mensajes
	dg.AddHandler(d.onMessageCreate)

	// Conectar al Gateway de Discord
	if err := dg.Open(); err != nil {
		return fmt.Errorf("❌ Error conectando al Gateway de Discord: %v", err)
	}

	fmt.Printf("✅ Nexus (Discord): Bot conectado como %s#%s\n", dg.State.User.Username, dg.State.User.Discriminator)
	return nil
}

// onMessageCreate se dispara cada vez que llega un mensaje a Discord.
func (d *DiscordProvider) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignorar mensajes del propio bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	senderID := m.Author.ID
	pushName := m.Author.Username
	if m.Author.GlobalName != "" {
		pushName = m.Author.GlobalName
	}

	channelID := m.ChannelID

	if handleMsg != nil {
		handleMsg("discord", m.Content, senderID, pushName, d.db, d.brain, func(targetID, text string) error {
			return d.sendToChannel(channelID, text)
		})
	}
}

// SendMessage envía un mensaje a un canal de Discord.
// 'target' debe ser el channelID como string.
func (d *DiscordProvider) SendMessage(target string, text string) error {
	return d.sendToChannel(target, text)
}

func (d *DiscordProvider) sendToChannel(channelID, text string) error {
	if d.session == nil {
		return fmt.Errorf("sesión de Discord no inicializada")
	}
	_, err := d.session.ChannelMessageSend(channelID, text)
	return err
}
