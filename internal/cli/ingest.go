package cli

import (
	"context"
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/database"
	"nexus-core/internal/nlp"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

var clearFlag bool
var tagFlag string
var summarizeFlag bool

var ingestCmd = &cobra.Command{
	Use:   "ingest [archivo]",
	Short: "Ingresa un archivo Markdown a la Base de Conocimientos",
	Long:  `Procesa un archivo de texto, lo divide en partes, calcula sus embeddings y lo inserta en Postgres.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		fmt.Printf("📦 Procesando archivo: %s\n", filePath)

		cfg := config.LoadConfig()
		
		dbDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Pass, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.Sslmode)
			
		// Aseguramos que existan las tablas antes de insertar
		conn, errConn := pgx.Connect(context.Background(), dbDSN)
		if errConn == nil {
			database.RunMigrations(conn)
			conn.Close(context.Background())
		}
			
		db, err := sql.Open("pgx", dbDSN)
		if err != nil {
			fmt.Printf("❌ Error conectando a DB: %v\n", err)
			return
		}

		brain, err := nlp.NewBrain(cfg, db)
		if err != nil {
			fmt.Printf("❌ Error inicializando Engine IA para Ingesta: %v\n", err)
			return
		}

		fmt.Println("🚀 Iniciando ingesta de documentos...")
		err = brain.IngestDocument(filePath, clearFlag, tagFlag, summarizeFlag)
		if err != nil {
			fmt.Printf("❌ Error en Ingesta: %v\n", err)
		} else {
			fmt.Println("✅ Base de Conocimientos alimentada satisfactoriamente.")
		}
	},
}

func init() {
	ingestCmd.Flags().BoolVarP(&clearFlag, "clear", "c", false, "Borrar toda la base de conocimientos antes de la ingesta")
	ingestCmd.Flags().StringVarP(&tagFlag, "tag", "t", "", "Etiqueta para categorizar este conocimiento (ej: 'ventas')")
	ingestCmd.Flags().BoolVarP(&summarizeFlag, "summarize", "s", false, "Usar IA para resumir fragmentos antes de vectorizar (mejora la búsqueda)")
	rootCmd.AddCommand(ingestCmd)
}
