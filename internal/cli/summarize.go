package cli

import (
	"context"
	"fmt"
	"log"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Resume los últimos mensajes recibidos",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadConfig()
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Pass, cfg.Database.Host,
			cfg.Database.Port, cfg.Database.Name, cfg.Database.Sslmode)

		conn, err := pgx.Connect(context.Background(), dsn)
		if err != nil {
			log.Fatalf("❌ Error DB: %v", err)
		}
		defer conn.Close(context.Background())

		// 1. Obtener los últimos 20 mensajes
		rows, err := conn.Query(context.Background(),
			"SELECT sender_id, content FROM messages WHERE is_from_nexus = false ORDER BY created_at DESC LIMIT 20")
		if err != nil {
			log.Fatalf("❌ Error al leer mensajes: %v", err)
		}

		var history []string
		for rows.Next() {
			var sender, content string
			rows.Scan(&sender, &content)
			history = append(history, fmt.Sprintf("%s: %s", sender, content))
		}

		if len(history) == 0 {
			fmt.Println("📭 No hay mensajes recientes para resumir.")
			return
		}

		// 2. Preparar el prompt para la IA
		fmt.Println("🧠 Nexus está analizando el historial...")
		brain, _ := nlp.NewBrain(cfg)
		contextText := strings.Join(history, "\n")
		prompt := fmt.Sprintf("A continuación tienes los últimos mensajes recibidos en WhatsApp. "+
			"Por favor, haz un resumen ejecutivo muy breve de los temas tratados:\n\n%s", contextText)

		resumen, err := brain.Ask(prompt)
		if err != nil {
			fmt.Printf("❌ Error de IA: %v\n", err)
			return
		}

		fmt.Printf("\n📋 RESUMEN DE ACTIVIDAD:\n%s\n", resumen)
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)
}
