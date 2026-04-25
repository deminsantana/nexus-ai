// Package agent - sales_agent.go
// Implementa el orquestador principal del agente de ventas.
// ProcessWithFSM es el punto de entrada desde el handler de mensajería.
package agent

import (
	"fmt"
	"nexus-core/internal/nlp"
	"strings"
)

// ProcessWithFSM procesa un mensaje entrante usando la máquina de estados de ventas.
// Es el equivalente al Brain.ProcessMessageWithContext pero con FSM de ventas.
//
// Retorna:
//   - reply: la respuesta de la IA limpia (sin señales de control)
//   - err: error si algo falla
func (a *SalesAgent) ProcessWithFSM(senderID, userMessage string, brain *nlp.Brain) (string, error) {
	// 1. Cargar estado actual del usuario
	cd, err := a.stateStore.Get(senderID)
	if err != nil {
		return "", fmt.Errorf("error cargando estado de conversación: %v", err)
	}

	// 2. Si la conversación ya terminó, responder con mensaje de cierre
	if cd.State == StateDone {
		return "¡Gracias por tu tiempo! Si tienes más preguntas en el futuro, no dudes en escribirme. 😊", nil
	}

	// 3. Si está en IDLE, comenzar con GREETING
	if cd.State == StateIdle {
		cd.State = StateGreeting
		cd.Turns = 0
	}

	// 4. Detectar comandos especiales del usuario
	lowerMsg := strings.ToLower(strings.TrimSpace(userMessage))
	if lowerMsg == "reiniciar" || lowerMsg == "reset" || lowerMsg == "empezar de nuevo" {
		if err := a.stateStore.Reset(senderID); err != nil {
			fmt.Printf("⚠️ Error reseteando estado de %s: %v\n", senderID, err)
		}
		return "Claro, empecemos de nuevo. 😊 ¡Hola! Soy tu asesor de " + a.cfg.ProductName + ". ¿En qué puedo ayudarte hoy?", nil
	}

	// 5. Construir el prompt del sistema según el estado actual
	systemPrompt := a.BuildPrompt(cd)

	// 6. Obtener contexto de conversación reciente de Redis
	pastContext := brain.GetContext(senderID)

	// 7. Construir el prompt completo para la IA
	fullPrompt := fmt.Sprintf("%s\n\nHistorial reciente:\n%s\nCliente: %s\nAsesor:", 
		systemPrompt, pastContext, userMessage)

	// 8. Consultar a la IA
	rawReply, err := brain.Provider.Ask(fullPrompt)
	if err != nil {
		return "", fmt.Errorf("error consultando IA en FSM: %v", err)
	}

	// 9. Calcular siguiente estado antes de limpiar la respuesta
	maxTurns := a.GetMaxTurns(cd.State)
	nextState := a.NextState(cd.State, rawReply, cd.Turns, maxTurns)

	// 10. Limpiar la respuesta (eliminar señales de control)
	cleanReply := CleanResponse(rawReply)

	// 11. Guardar contexto en Redis (para historial de conversación)
	brain.SaveContext(senderID, "Cliente: "+userMessage)
	brain.SaveContext(senderID, "Asesor: "+cleanReply)

	// 12. Actualizar estado
	cd.Turns++
	cd.TotalTurns++
	if nextState != cd.State {
		fmt.Printf("🔄 FSM [%s]: %s → %s (turno %d)\n", senderID, cd.State, nextState, cd.TotalTurns)
		cd.State = nextState
		cd.Turns = 0 // resetear contador de turnos en el nuevo estado
	}

	// 13. Persistir estado actualizado
	if err := a.stateStore.Save(senderID, cd); err != nil {
		fmt.Printf("⚠️ Error guardando estado FSM para %s: %v\n", senderID, err)
	}

	fmt.Printf("🤝 FSM [%s | estado: %s | turno %d]: %s\n", senderID, cd.State, cd.TotalTurns, cleanReply)
	return cleanReply, nil
}
