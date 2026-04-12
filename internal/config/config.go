package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port int `yaml:"port"`
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
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
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
	Meta MetaConfig `yaml:"meta"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
}

// MessagingConfig agrupa la configuración de todas las plataformas de mensajería.
// El campo 'Provider' determina cuál se activa al iniciar Nexus.
//
// Valores de Provider:
//   - "mau"      → WhatsApp no-oficial (whatsmeow), se vincula con QR
//   - "meta"     → WhatsApp Business API oficial de Meta
//   - "telegram" → Bot de Telegram (solo necesitas bot_token de @BotFather)
type MessagingConfig struct {
	Provider string                 `yaml:"provider"`
	WhatsApp WhatsAppProviderConfig `yaml:"whatsapp"`
	Telegram TelegramConfig         `yaml:"telegram"`
}

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Messaging MessagingConfig `yaml:"messaging"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	AI        AIConfig        `yaml:"ai"`
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
