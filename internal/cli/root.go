package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nexus",
	Short: "Nexus - Tu asistente personal inteligente",
	Long:  `Nexus es un núcleo centralizado para mensajería, automatización e IA.`,
}

// Execute es el punto de entrada que llamará main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
