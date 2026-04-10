package nlp

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// IngestDocument lee un archivo, lo divide en chunks y lo guarda en PostgreSQL con sus embeddings
func (b *Brain) IngestDocument(filePath string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error leyendo archivo: %v", err)
	}

	content := string(contentBytes)
	// Chunking super simple: por doble salto de línea (párrafos)
	chunks := strings.Split(content, "\n\n")

	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}

		// 1. Generar embedding
		embValues, err := b.Provider.Embed(chunk)
		if err != nil {
			log.Printf("Error al generar embedding para el chunk: %v", err)
			continue
		}

		// 2. Formatear embedding como string para pgvector -> "[0.1, 0.2, ...]"
		embStr := formatFloat32SliceForVector(embValues)

		// 3. Insertar en base de datos
		query := `INSERT INTO knowledge_chunks (content, embedding) VALUES ($1, $2)`
		_, err = b.DB.Exec(query, chunk, embStr)
		if err != nil {
			log.Printf("Error insertando chunk en db: %v", err)
			continue
		}
		
		log.Println("✅ Chunk indexado correctamente.")
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

// SearchKnowledgeBase toma la pregunta del usuario, busca en pgvector y retorna el contexto
func (b *Brain) SearchKnowledgeBase(query string, limit int) (string, error) {
	if b.DB == nil {
		return "", fmt.Errorf("DB no está instanciada en Brain")
	}
	
	// Verificar si la base de datos ya tiene chunks
	var count int
	b.DB.QueryRow("SELECT COUNT(*) FROM knowledge_chunks").Scan(&count)
	if count == 0 {
		return "", nil // Base de conocimientos vacía, no usamos RAG
	}

	embValues, err := b.Provider.Embed(query)
	if err != nil {
		return "", err
	}

	embStr := formatFloat32SliceForVector(embValues)

	// <=> es la Distancia del Coseno, ideal para texto. ORDER BY ascendente (los más similares primero)
	sqlQuery := `
		SELECT content 
		FROM knowledge_chunks 
		ORDER BY embedding <=> $1 
		LIMIT $2
	`
	
	rows, err := b.DB.Query(sqlQuery, embStr, limit)
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
