package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"nexus-core/internal/api"
	"nexus-core/internal/config"
	"nexus-core/internal/database"
	"nexus-core/internal/messaging"
	"nexus-core/internal/nlp"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Inicia el núcleo de Nexus y los servicios de mensajería",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadConfig()

		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Pass, cfg.Database.Host,
			cfg.Database.Port, cfg.Database.Name, cfg.Database.Sslmode)

		// 1. Migraciones
		conn, err := pgx.Connect(context.Background(), dsn)
		if err != nil {
			log.Fatalf("Error de conexión: %v", err)
		}
		database.RunMigrations(conn)
		conn.Close(context.Background())

		fmt.Println("🚀 Nexus Core activado...")

		// 2. Inicializar Cerebro (NLP)
		dbConn, _ := sql.Open("pgx", dsn)
		brain, err := nlp.NewBrain(cfg, dbConn)
		if err != nil {
			fmt.Printf("❌ Error Cerebro: %v\n", err)
		}

		// 3. Iniciar Proveedor de Mensajería (Telegram, WhatsApp Mau, WhatsApp Meta)
		provider, err := messaging.InitProvider(cfg)
		if err != nil {
			log.Fatalf("❌ Error inicializando proveedor: %v", err)
		}

		err = provider.Start(cfg, dsn, dbConn, brain)
		if err != nil {
			log.Fatalf("❌ Error iniciando proveedor: %v", err)
		}

		// 4. Iniciar Servidor API y Webhooks centralizado
		// Esto permite que el endpoint genérico y los webhooks de Meta/IG funcionen juntos
		http.HandleFunc("/api/webhook/ai", api.NewAIHandler(brain, cfg))

		port := fmt.Sprintf(":%d", cfg.Server.Port)
		fmt.Printf("🌐 Servidor HTTP iniciado en puerto %d (API disponible en /api/webhook/ai)\n", cfg.Server.Port)

		go func() {
			if err := http.ListenAndServe(port, nil); err != nil {
				fmt.Printf("❌ Error en Servidor HTTP Global: %v\n", err)
			}
		}()

		// 5. BLOQUEO PARA MANTENER EL COMANDO VIVO
		// Escuchamos señales de interrupción del sistema para cerrar elegantemente
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		fmt.Println("📌 Nexus está escuchando mensajes... (Presiona Ctrl+C para detener)")

		<-c // El programa se detiene aquí hasta que recibe una señal
		fmt.Println("\nTerminando Nexus Core...")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
