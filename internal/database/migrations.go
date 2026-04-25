package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

// Define el esquema inicial de Nexus
const schema = `
-- Cargar extensión pgvector
CREATE EXTENSION IF NOT EXISTS vector;

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

-- Tabla RAG para Base de Conocimientos
CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(3072),
    source TEXT                       -- Columna añadida para rastrear el origen del archivo
);

-- Tabla de cuotas de uso (Rate Limiting por BD)
CREATE TABLE IF NOT EXISTS usage_quotas (
    sender_id TEXT PRIMARY KEY,
    messages_sent INT DEFAULT 0,
    message_limit INT DEFAULT 1000,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Forzamos la actualización de la columna por si se había creado con 768 en el intento anterior
ALTER TABLE knowledge_chunks ALTER COLUMN embedding TYPE vector(3072);
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS category TEXT;
`

func RunMigrations(conn *pgx.Conn) {
	_, err := conn.Exec(context.Background(), schema)
	if err != nil {
		log.Fatalf("Error ejecutando auto-migración: %v", err)
	}
	log.Println("Estructura de base de datos actualizada correctamente.")
}
