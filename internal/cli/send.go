package cli

import (
	"fmt"
	"log"
	"nexus-core/internal/config"
	"nexus-core/internal/whatsapp"

	"github.com/spf13/cobra"
)

var (
	recipient string
	message   string
)

var sendCmd = &cobra.Command{
	Use:     "send",
	Short:   "Envía un mensaje de WhatsApp a un número específico",
	Example: `  nexus send --to 584121234567 --msg "Hola desde la terminal"`,
	Run: func(cmd *cobra.Command, args []string) {
		if recipient == "" || message == "" {
			fmt.Println("❌ Error: Debes especificar un destinatario (--to) y un mensaje (--msg)")
			return
		}

		cfg := config.LoadConfig()
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Pass, cfg.Database.Host,
			cfg.Database.Port, cfg.Database.Name, cfg.Database.Sslmode)

		fmt.Printf("📤 Enviando mensaje a %s...\n", recipient)

		// Llamamos a una nueva función que crearemos en el paquete whatsapp
		err := whatsapp.SendMessage(dsn, recipient, message)
		if err != nil {
			log.Fatalf("❌ Error al enviar mensaje: %v", err)
		}

		fmt.Println("✅ Mensaje enviado con éxito.")
	},
}

func init() {
	sendCmd.Flags().StringVarP(&recipient, "to", "t", "", "Número de teléfono (ej: 584121234567)")
	sendCmd.Flags().StringVarP(&message, "msg", "m", "", "Contenido del mensaje")
	rootCmd.AddCommand(sendCmd)
}
