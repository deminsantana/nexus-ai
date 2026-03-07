package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

// Define el esquema inicial de Nexus
const schema = `
-- Tabla de configuraciones globales
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de integraciones (WhatsApp, Discord, etc.)
CREATE TABLE IF NOT EXISTS integrations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,       -- 'whatsapp', 'discord'
    status TEXT DEFAULT 'disconnected',
    credentials JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de conversaciones (El historial del chat)
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    source TEXT NOT NULL,            -- 'whatsapp', 'webchat', 'cli'
    sender_id TEXT NOT NULL,
    content TEXT NOT NULL,
    is_from_nexus BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla especial para que WhatsApp guarde su sesión de forma segura
CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    id SERIAL PRIMARY KEY,
    data BYTEA
);
`

func RunMigrations(conn *pgx.Conn) {
	_, err := conn.Exec(context.Background(), schema)
	if err != nil {
		log.Fatalf("Error ejecutando auto-migración: %v", err)
	}
	log.Println("Estructura de base de datos actualizada correctamente.")
}
