package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	APIKey  string `yaml:"api_key"`
}

type DatabaseConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	Pass    string `yaml:"pass"`
	Name    string `yaml:"name"`
	Sslmode string `yaml:"sslmode"`
}

type AIConfig struct {
	Provider string `yaml:"provider"` // "openai" | "gemini"
	APIKey   string `yaml:"api_key"`  // Se mantiene para OpenAI
	Model    string `yaml:"model"`
}

type RedisConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// --- Configuraciones por plataforma ---

type MetaConfig struct {
	Token         string `yaml:"token"`
	PhoneNumberId string `yaml:"phone_number_id"`
	VerifyToken   string `yaml:"verify_token"`
}

type WhatsAppProviderConfig struct {
	Meta        MetaConfig `yaml:"meta"`
	SessionPath string     `yaml:"session_path"`
	AllowGroups bool       `yaml:"allow_groups"` // responder en grupos
	HandleMedia bool       `yaml:"handle_media"` // reaccionar a stickers/fotos
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
}

type DiscordConfig struct {
	BotToken string `yaml:"bot_token"`
	GuildID  string `yaml:"guild_id"` // opcional, para restricción de servidor
}

type SlackConfig struct {
	BotToken      string `yaml:"bot_token"`      // xoxb-...
	AppToken      string `yaml:"app_token"`      // xapp-... (Socket Mode)
	SigningSecret string `yaml:"signing_secret"` // para validar payloads
}

type InstagramConfig struct {
	PageAccessToken string `yaml:"page_access_token"`
	VerifyToken     string `yaml:"verify_token"`
	IGID            string `yaml:"ig_id"` // Instagram Business Account ID
}

type MessengerConfig struct {
	PageAccessToken string `yaml:"page_access_token"`
	VerifyToken     string `yaml:"verify_token"`
	PageID          string `yaml:"page_id"`
}

type TwilioConfig struct {
	AccountSID  string `yaml:"account_sid"`
	AuthToken   string `yaml:"auth_token"`
	FromNumber  string `yaml:"from_number"`   // +1XXXXXXXXXX
	WebhookPort int    `yaml:"webhook_port"` // puerto para recibir SMS entrantes
}

type EmailConfig struct {
	IMAPHost     string `yaml:"imap_host"`
	IMAPPort     int    `yaml:"imap_port"`
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	PollInterval int    `yaml:"poll_interval_seconds"` // cada cuántos segundos revisar el inbox
}

// MessagingConfig agrupa la configuración de todas las plataformas de mensajería.
// El campo 'Provider' determina cuál se activa al iniciar Nexus.
//
// Valores de Provider:
//   - "mau"       → WhatsApp no-oficial (whatsmeow), se vincula con QR
//   - "meta"      → WhatsApp Business API oficial de Meta
//   - "telegram"  → Bot de Telegram
//   - "discord"   → Bot de Discord (Gateway WebSocket)
//   - "slack"     → App de Slack (Socket Mode, sin URL pública)
//   - "instagram" → Instagram DM via Meta Graph API
//   - "messenger" → Facebook Messenger via Meta Graph API
//   - "twilio"    → SMS via Twilio REST API
//   - "email"     → Correo electrónico via IMAP/SMTP
type MessagingConfig struct {
	Provider  string                 `yaml:"provider"`
	WhatsApp  WhatsAppProviderConfig `yaml:"whatsapp"`
	Telegram  TelegramConfig         `yaml:"telegram"`
	Discord   DiscordConfig          `yaml:"discord"`
	Slack     SlackConfig            `yaml:"slack"`
	Instagram InstagramConfig        `yaml:"instagram"`
	Messenger MessengerConfig        `yaml:"messenger"`
	Twilio    TwilioConfig           `yaml:"twilio"`
	Email     EmailConfig            `yaml:"email"`
}

// --- Voice Agent ---

type VoiceTwilioConfig struct {
	AccountSID  string `yaml:"account_sid"`
	AuthToken   string `yaml:"auth_token"`
	FromNumber  string `yaml:"from_number"`
	TwiMLBinURL string `yaml:"twiml_bin_url"` // URL TwiML para llamadas outbound
}

type VoiceConfig struct {
	Provider     string            `yaml:"provider"`      // "twilio" | "google" | "none"
	ResponseMode string            `yaml:"response_mode"` // "text" | "voice" | "both"
	Twilio       VoiceTwilioConfig `yaml:"twilio"`
}

type GoogleConfig struct {
	APIKey          string  `yaml:"api_key"`          // Para Gemini (AI Studio)
	CredentialsFile string  `yaml:"credentials_file"` // Para Cloud TTS (JSON)
	Language        string  `yaml:"language"`
	VoiceName       string  `yaml:"voice_name"`
	Gender          string  `yaml:"gender"`
	Pitch           float64 `yaml:"pitch"`
	SpeakingRate    float64 `yaml:"speaking_rate"`
}

// --- Scheduler (llamadas programadas) ---

type ScheduledJob struct {
	Name    string `yaml:"name"`
	Cron    string `yaml:"cron"`    // expresión cron, ej: "0 9 * * 1" o "@every 1m"
	Type    string `yaml:"type"`    // "call" | "voice_message" | "text_message"
	To      string `yaml:"to"`     // número E.164 o chat_id
	Message string `yaml:"message"` // texto que se convierte en voz o se envía
}

type SchedulerConfig struct {
	Enabled bool           `yaml:"enabled"`
	Jobs    []ScheduledJob `yaml:"jobs"`
}

// --- Sales Agent FSM ---

type SalesStateConfig struct {
	Prompt   string `yaml:"prompt"`
	MaxTurns int    `yaml:"max_turns"`
}

type SalesAgentConfig struct {
	Enabled     bool                        `yaml:"enabled"`
	ProductName string                      `yaml:"product_name"`
	States      map[string]SalesStateConfig `yaml:"states"`
}

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Messaging  MessagingConfig  `yaml:"messaging"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	AI         AIConfig         `yaml:"ai"`
	Voice      VoiceConfig      `yaml:"voice"`
	Google     GoogleConfig     `yaml:"google"` // <-- Nueva sección unificada
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	SalesAgent SalesAgentConfig `yaml:"sales_agent"`
}

func LoadConfig() *Config {
	dir, _ := os.Getwd()
	fmt.Printf("📂 Buscando config.yaml en: %s\n", dir)

	f, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("No se pudo abrir config.yaml: %v", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalf("Error al decodificar YAML: %v", err)
	}

	return &cfg
}
