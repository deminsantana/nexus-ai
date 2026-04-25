package nlp

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// IngestDocument lee un archivo o URL, lo divide en chunks y lo guarda en PostgreSQL.
// - filePath: ruta al archivo o URL.
// - clear: si es true, borra toda la base de datos.
// - tag: categoría para filtrar (ej: "ventas", "soporte").
// - summarize: si es true, usa la IA para resumir cada chunk antes de vectorizarlo.
func (b *Brain) IngestDocument(filePath string, clear bool, tag string, summarize bool) error {
	if clear {
		fmt.Println("🧹 Limpiando TODA la base de conocimientos...")
		b.DB.Exec("DELETE FROM knowledge_chunks")
	} else {
		fmt.Printf("🔄 Actualizando conocimiento del origen: %s (Tag: %s)\n", filePath, tag)
		b.DB.Exec("DELETE FROM knowledge_chunks WHERE source = $1", filePath)
	}

	var content string
	var err error

	// Detectar si es una URL
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		fmt.Println("🌐 Descargando contenido de URL...")
		content, err = ScrapeURL(filePath)
	} else {
		contentBytes, errR := os.ReadFile(filePath)
		err = errR
		content = string(contentBytes)
	}

	if err != nil {
		return fmt.Errorf("error obteniendo contenido: %v", err)
	}

	// Chunking
	chunks := strings.Split(content, "\n\n")

	for i, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}

		processText := chunk
		if summarize {
			fmt.Printf("📝 Resumiendo fragmento %d/%d...\n", i+1, len(chunks))
			summary, sErr := b.Provider.Ask(fmt.Sprintf("Resume brevemente este texto para que sea fácil de encontrar en una búsqueda semántica. Mantén los datos clave (nombres, fechas, precios):\n\n%s", chunk))
			if sErr == nil {
				processText = summary
			}
		}

		// 1. Generar embedding (del resumen si existe, o del original)
		embValues, err := b.Provider.Embed(processText)
		if err != nil {
			log.Printf("Error al generar embedding para el chunk: %v", err)
			continue
		}

		embStr := formatFloat32SliceForVector(embValues)

		// 2. Insertar con tag y source
		query := `INSERT INTO knowledge_chunks (content, embedding, source, category) VALUES ($1, $2, $3, $4)`
		_, err = b.DB.Exec(query, chunk, embStr, filePath, tag)
		if err != nil {
			log.Printf("Error insertando chunk en db: %v", err)
			continue
		}
	}

	return nil
}

// formatFloat32SliceForVector toma un slice y entrega el formato requerido por el tipo de dato PostgreSQL vector
func formatFloat32SliceForVector(values []float32) string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}

// SearchKnowledgeBase toma la pregunta del usuario, busca en pgvector y filtra por categoría si se provee.
func (b *Brain) SearchKnowledgeBase(query string, limit int, tag string) (string, error) {
	if b.DB == nil {
		return "", fmt.Errorf("DB no está instanciada en Brain")
	}

	embValues, err := b.Provider.Embed(query)
	if err != nil {
		return "", err
	}

	embStr := formatFloat32SliceForVector(embValues)

	// Construir query dinámica para filtrar por tag si no está vacío
	sqlQuery := `
		SELECT content 
		FROM knowledge_chunks 
		WHERE ($2 = '' OR category = $2)
		ORDER BY embedding <=> $1 
		LIMIT $3
	`
	
	rows, err := b.DB.Query(sqlQuery, embStr, tag, limit)
	if err != nil {
		return "", fmt.Errorf("error haciendo query a knowledge_chunks: %v", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		results = append(results, content)
	}

	if len(results) == 0 {
		return "", nil
	}
	
	return strings.Join(results, "\n... \n"), nil
}
