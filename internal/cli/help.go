package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help-me",
	Short: "Muestra la guía completa de comandos de Nexus",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`
🌟 NEXUS CORE - GUÍA DE USUARIO 🌟

Nexus es tu asistente personal integrado con WhatsApp e Inteligencia Artificial.

COMANDOS DISPONIBLES:
----------------------------------------------------------------------
1. nexus serve
   Explicación: Inicia el servidor de escucha de mensajes.
   Uso: Debe estar corriendo para que Nexus pueda guardar chats y responder.
   Ejemplo: nexus serve

2. nexus status
   Explicación: Verifica la conexión con Postgres, Redis y la IA.
   Uso: Úsalo para diagnosticar problemas de infraestructura.

3. nexus send --to [numero] --msg "[mensaje]"
   Explicación: Envía un mensaje manual a cualquier número.
   Ejemplo: nexus send --to 584128833155 --msg "Hola desde la terminal"

4. nexus summarize
   Explicación: La IA analiza los últimos 20 mensajes y genera un resumen.
   Uso: Ideal para ponerse al día rápidamente.

5. nexus help-me
   Explicación: Muestra esta pantalla de ayuda.

NOTAS:
- Para que la IA responda automáticamente en WhatsApp, el mensaje
  debe empezar con la palabra "Nexus".
----------------------------------------------------------------------`)
	},
}

func init() {
	rootCmd.AddCommand(helpCmd)
}
