package cli

import (
	"database/sql"
	"fmt"
	"io/fs"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [directorio]",
	Short: "Monitorea una carpeta y auto-ingesta cambios en archivos Markdown",
	Long:  `Revisa periódicamente una carpeta. Si detecta un archivo nuevo o modificado (.md), lo procesa automáticamente.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirPath := args[0]
		fmt.Printf("👀 Iniciando monitoreo en: %s (Polling cada 5s)\n", dirPath)

		cfg := config.LoadConfig()
		dbDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Pass, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.Sslmode)

		db, err := sql.Open("pgx", dbDSN)
		if err != nil {
			fmt.Printf("❌ Error DB: %v\n", err)
			return
		}

		brain, err := nlp.NewBrain(cfg, db)
		if err != nil {
			fmt.Printf("❌ Error IA: %v\n", err)
			return
		}

		// fileStates rastrea la última fecha de modificación de cada archivo
		fileStates := make(map[string]time.Time)

		for {
			err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				// Solo procesar archivos Markdown
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
					info, err := d.Info()
					if err != nil {
						return nil
					}

					modTime := info.ModTime()
					lastTime, seen := fileStates[path]

					// Si es nuevo o ha sido modificado
					if !seen || modTime.After(lastTime) {
						if !seen {
							fmt.Printf("📄 Nuevo archivo detectado: %s\n", d.Name())
						} else {
							fmt.Printf("✏️ Cambio detectado en: %s\n", d.Name())
						}

						// Ingesta automática (sin clear, sin summarize por defecto)
						// Usamos el nombre de la carpeta como tag por defecto
						tag := filepath.Base(dirPath)
						err := brain.IngestDocument(path, false, tag, false)
						if err != nil {
							fmt.Printf("❌ Falló auto-ingesta de %s: %v\n", d.Name(), err)
						} else {
							fileStates[path] = modTime
							fmt.Printf("✅ %s sincronizado correctamente.\n", d.Name())
						}
					}
				}
				return nil
			})

			if err != nil {
				fmt.Printf("⚠️ Error recorriendo directorio: %v\n", err)
			}

			time.Sleep(5 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
