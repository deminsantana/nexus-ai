package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"nexus-core/internal/api"
	"nexus-core/internal/config"
	"nexus-core/internal/database"
	"nexus-core/internal/messaging"
	"nexus-core/internal/nlp"
	"nexus-core/internal/scheduler"
	"nexus-core/internal/voice"
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

		// 2. Iniciar Cerebro (Gemini o OpenAI)
		dbConn, _ := sql.Open("pgx", dsn)
		var brain *nlp.Brain
		brain, err = nlp.NewBrain(cfg, dbConn)
		if err != nil {
			fmt.Printf("❌ Error Cerebro: %v\n", err)
		}

		// 3. Iniciar Voice Provider
		voiceProvider, err := voice.InitProvider(cfg)
		if err != nil {
			fmt.Printf("⚠️ Voice provider no disponible: %v\n", err)
		}

		// 4. Inyectar config y preparar Sales Agent FSM
		messaging.SetConfig(cfg, brain, voiceProvider)

		// 5. Iniciar Proveedor de Mensajería (Telegram, WhatsApp Mau, WhatsApp Meta, etc.)
		provider, err := messaging.InitProvider(cfg)
		if err != nil {
			log.Fatalf("❌ Error inicializando proveedor: %v", err)
		}

		err = provider.Start(cfg, dsn, dbConn, brain)
		if err != nil {
			log.Fatalf("❌ Error iniciando proveedor: %v", err)
		}
		
		if voiceProvider != nil {
			fmt.Printf("🎙️ Voice provider activo: %s\n", cfg.Voice.Provider)
		}

		// 6. Iniciar Scheduler (llamadas y mensajes programados)
		if cfg.Scheduler.Enabled {
			sched := scheduler.New(cfg, voiceProvider, provider)
			if err := sched.Start(); err != nil {
				fmt.Printf("❌ Error iniciando scheduler: %v\n", err)
			} else {
				// Detener scheduler al cerrar
				defer sched.Stop()
			}
		}

		// 7. Iniciar Servidor API y Webhooks centralizado (SOLO SI ESTÁ HABILITADO)
		if cfg.Server.Enabled {
			http.HandleFunc("/api/webhook/ai", api.NewAIHandler(brain, cfg))

			// Endpoint TwiML local para llamadas Twilio (si se usa)
			if cfg.Voice.Provider == "twilio" && cfg.Voice.Twilio.TwiMLBinURL == "" {
				http.HandleFunc("/voice/twiml", func(w http.ResponseWriter, r *http.Request) {
					message := r.URL.Query().Get("message")
					if message == "" {
						message = "Hola, soy Nexus, tu asistente inteligente de ventas."
					}
					w.Header().Set("Content-Type", "text/xml")
					fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Response><Say language="es-MX" voice="Polly.Lupe">%s</Say></Response>`, message)
				})
				fmt.Printf("🎤 TwiML local disponible en /voice/twiml\n")
			}

			port := fmt.Sprintf(":%d", cfg.Server.Port)
			fmt.Printf("🌐 Servidor HTTP iniciado en puerto %d (API disponible en /api/webhook/ai)\n", cfg.Server.Port)

			go func() {
				if err := http.ListenAndServe(port, nil); err != nil {
					fmt.Printf("❌ Error en Servidor HTTP Global: %v\n", err)
				}
			}()
		} else {
			fmt.Println("🌐 Servidor HTTP desactivado (según config.yaml)")
		}

		// 8. BLOQUEO PARA MANTENER EL COMANDO VIVO
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
