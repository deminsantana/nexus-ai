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

type MetaConfig struct {
	Token         string `yaml:"token"`
	PhoneNumberId string `yaml:"phone_number_id"`
	VerifyToken   string `yaml:"verify_token"`
}

type WhatsAppConfig struct {
	Provider string     `yaml:"provider"`
	Meta     MetaConfig `yaml:"meta"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	AI       AIConfig       `yaml:"ai"`
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
