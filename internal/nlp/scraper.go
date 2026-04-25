package nlp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ScrapeURL descarga el contenido de una URL y extrae el texto "limpio".
// Nota: Esta es una implementación básica. Para sitios complejos se recomienda
// usar librerías como goquery.
func ScrapeURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error al acceder a la URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error de estado HTTP: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer el cuerpo de la respuesta: %v", err)
	}

	// Limpieza básica de HTML (esto es muy simple, idealmente usar goquery)
	html := string(body)
	text := cleanHTML(html)

	return text, nil
}

// cleanHTML realiza una limpieza rudimentaria de etiquetas HTML.
func cleanHTML(html string) string {
	// 1. Eliminar scripts y estilos
	// (En una implementación real usaríamos regex o un parser)
	
	// Por ahora, una limpieza muy simple para el MVP:
	text := html
	tags := []string{"<script", "<style", "<nav", "<footer", "<header"}
	for _, tag := range tags {
		start := strings.Index(text, tag)
		for start != -1 {
			end := strings.Index(text[start:], ">")
			if end != -1 {
				// Buscar cierre de etiqueta </... >
				closingTag := "</" + tag[1:] + ">"
				closeIdx := strings.Index(text[start:], closingTag)
				if closeIdx != -1 {
					text = text[:start] + text[start+closeIdx+len(closingTag):]
				} else {
					text = text[:start] + text[start+end+1:]
				}
			} else {
				break
			}
			start = strings.Index(text, tag)
		}
	}

	// Eliminar todas las etiquetas restantes <...>
	var sb strings.Builder
	inTag := false
	for _, r := range text {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			sb.WriteRune(r)
		}
	}

	// Limpiar espacios en blanco excesivos
	result := sb.String()
	lines := strings.Split(result, "\n")
	var finalLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			finalLines = append(finalLines, trimmed)
		}
	}

	return strings.Join(finalLines, "\n")
}
