package cli

import (
	"fmt"
	"net"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verifica el estado de los servicios (Postgres y Redis)",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadConfig()
		services := map[string]string{
			"Postgres (Nexus DB)": fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
			"Redis (Cache)":       fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		}

		fmt.Println("🔍 Verificando infraestructura de Nexus...")
		for name, addr := range services {
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				fmt.Printf("❌ %s [%s]: FUERA DE LÍNEA\n", name, addr)
			} else {
				fmt.Printf("✅ %s [%s]: ACTIVO\n", name, addr)
				conn.Close()
			}
		}

		fmt.Println("\n🧠 Verificando conexión con Gemini AI...")

		if cfg.AI.APIKey == "" {
			fmt.Println("❌ Error: La API Key cargada desde el YAML está VACÍA.")
			return
		} else {
			fmt.Printf("ℹ️ API Key detectada (comienza por: %s...)\n", cfg.AI.APIKey[:5])
		}

		brain, err := nlp.NewBrain(cfg, nil)
		if err != nil {
			fmt.Printf("❌ IA: Error de configuración: %v\n", err)
			return
		}

		defer brain.Provider.Close()

		respuesta, err := brain.Provider.Ask("Hola, responde con la palabra 'CONECTADO' si puedes leerme.")
		if err != nil {
			fmt.Printf("❌ IA: Error de comunicación: %v\n", err)
		} else {
			fmt.Printf("✅ IA: Respuesta del modelo: %s\n", respuesta)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
