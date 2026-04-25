// Package agent implementa el Agente de Ventas con máquina de estados (FSM).
// Cada usuario tiene su propio estado de conversación, persistido en Redis.
//
// Estados del funnel de ventas:
//
//	IDLE → GREETING → QUALIFY → PRESENT → OBJECTION → CLOSE → FOLLOW_UP → DONE
//	                                          ↑___________↓
//
// El FSM transiciona automáticamente entre estados basándose en:
//   - Número de turnos en el estado actual (max_turns)
//   - Palabras clave detectadas en el mensaje del usuario
//   - Señales de intención generadas por la IA
package agent

import (
	"encoding/json"
	"fmt"
	"nexus-core/internal/config"
	"strings"
	"time"

	"context"

	"github.com/redis/go-redis/v9"
)

// SalesState representa los estados posibles del agente de ventas.
type SalesState string

const (
	StateIdle       SalesState = "idle"
	StateGreeting   SalesState = "greeting"
	StateQualify    SalesState = "qualify"
	StatePresent    SalesState = "present"
	StateObjection  SalesState = "objection"
	StateClose      SalesState = "close"
	StateFollowUp   SalesState = "follow_up"
	StateDone       SalesState = "done"
)

// StateFlow define el orden de transición estándar de los estados.
var StateFlow = []SalesState{
	StateGreeting,
	StateQualify,
	StatePresent,
	StateObjection,
	StateClose,
	StateFollowUp,
	StateDone,
}

// ConversationData almacena el estado de conversación de un usuario en Redis.
type ConversationData struct {
	State      SalesState        `json:"state"`
	Turns      int               `json:"turns"`       // turnos en el estado actual
	TotalTurns int               `json:"total_turns"` // turnos totales en la conversación
	UserData   map[string]string `json:"user_data"`   // datos recopilados (nombre, empresa, etc.)
	UpdatedAt  time.Time         `json:"updated_at"`
}

// StateStore gestiona la persistencia de estados en Redis.
type StateStore struct {
	rdb *redis.Client
	ctx context.Context
	ttl time.Duration
}

// NewStateStore crea un StateStore conectado a Redis.
func NewStateStore(rdb *redis.Client) *StateStore {
	return &StateStore{
		rdb: rdb,
		ctx: context.Background(),
		ttl: 24 * time.Hour, // conversaciones activas por 24 horas
	}
}

// Get obtiene el estado actual de un usuario.
// Si no existe, devuelve un estado inicial IDLE.
func (s *StateStore) Get(senderID string) (*ConversationData, error) {
	key := "sales:state:" + senderID
	data, err := s.rdb.Get(s.ctx, key).Result()
	if err == redis.Nil {
		// Usuario nuevo: iniciar en IDLE
		return &ConversationData{
			State:    StateIdle,
			Turns:    0,
			UserData: make(map[string]string),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error leyendo estado Redis: %v", err)
	}

	var cd ConversationData
	if err := json.Unmarshal([]byte(data), &cd); err != nil {
		return nil, fmt.Errorf("error deserializando estado: %v", err)
	}
	return &cd, nil
}

// Save persiste el estado de un usuario en Redis.
func (s *StateStore) Save(senderID string, cd *ConversationData) error {
	cd.UpdatedAt = time.Now()
	key := "sales:state:" + senderID

	bytes, err := json.Marshal(cd)
	if err != nil {
		return fmt.Errorf("error serializando estado: %v", err)
	}

	return s.rdb.Set(s.ctx, key, bytes, s.ttl).Err()
}

// Reset elimina el estado de un usuario (reinicia la conversación).
func (s *StateStore) Reset(senderID string) error {
	key := "sales:state:" + senderID
	return s.rdb.Del(s.ctx, key).Err()
}

// SalesAgent es el agente de ventas con FSM.
// Tiene acceso a la configuración de estados y al store de Redis.
type SalesAgent struct {
	cfg        config.SalesAgentConfig
	stateStore *StateStore
}

// NewSalesAgent crea un SalesAgent con la configuración y Redis dados.
func NewSalesAgent(cfg config.SalesAgentConfig, rdb *redis.Client) *SalesAgent {
	return &SalesAgent{
		cfg:        cfg,
		stateStore: NewStateStore(rdb),
	}
}

// BuildPrompt construye el system prompt para la IA según el estado actual.
// Combina el prompt del estado con instrucciones genéricas de ventas.
func (a *SalesAgent) BuildPrompt(cd *ConversationData) string {
	stateCfg, ok := a.cfg.States[string(cd.State)]
	if !ok {
		// Estado sin configuración específica: usar prompt genérico
		return fmt.Sprintf(
			"Eres un asesor de ventas de %s. Estás en la etapa '%s'. "+
				"Sé amigable, profesional y guía al cliente hacia la siguiente etapa.",
			a.cfg.ProductName, cd.State,
		)
	}

	userData := formatUserData(cd.UserData)

	prompt := fmt.Sprintf(
		"Eres un asesor de ventas de %s.\n"+
			"ETAPA ACTUAL: %s\n"+
			"TU ROL EN ESTA ETAPA: %s\n",
		a.cfg.ProductName, cd.State, stateCfg.Prompt,
	)

	if userData != "" {
		prompt += "\nINFORMACIÓN DEL CLIENTE RECOPILADA:\n" + userData + "\n"
	}

	prompt += fmt.Sprintf(
		"\nINSTRUCCIONES:\n"+
			"- Responde de forma natural y conversacional (máx 3 oraciones).\n"+
			"- Escucha activamente y personaliza tu respuesta al contexto del cliente.\n"+
			"- Cuando hayas completado el objetivo de esta etapa, incluye al final: [AVANZAR]\n"+
			"- Si el cliente muestra interés explícito en comprar, incluye al final: [CERRAR]\n"+
			"- Si el cliente rechaza definitivamente, incluye al final: [FIN]\n"+
			"- Turno %d de máximo %d en esta etapa.\n",
		cd.Turns+1, stateCfg.MaxTurns,
	)

	return prompt
}

// NextState calcula el siguiente estado basándose en señales de la IA y turnos.
func (a *SalesAgent) NextState(current SalesState, aiResponse string, turns, maxTurns int) SalesState {
	// Señales explícitas de la IA en la respuesta
	if strings.Contains(aiResponse, "[CERRAR]") {
		return StateClose
	}
	if strings.Contains(aiResponse, "[FIN]") {
		return StateDone
	}
	if strings.Contains(aiResponse, "[AVANZAR]") || turns >= maxTurns-1 {
		return advanceState(current)
	}
	return current // mantener estado actual
}

// advanceState retorna el siguiente estado en el flujo.
func advanceState(current SalesState) SalesState {
	for i, s := range StateFlow {
		if s == current && i+1 < len(StateFlow) {
			return StateFlow[i+1]
		}
	}
	return StateDone
}

// CleanResponse elimina las señales de control de la respuesta antes de enviarla al usuario.
func CleanResponse(response string) string {
	response = strings.ReplaceAll(response, "[AVANZAR]", "")
	response = strings.ReplaceAll(response, "[CERRAR]", "")
	response = strings.ReplaceAll(response, "[FIN]", "")
	return strings.TrimSpace(response)
}

// formatUserData formatea los datos del usuario recopilados para el prompt.
func formatUserData(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}
	var sb strings.Builder
	for k, v := range data {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	return sb.String()
}

// GetMaxTurns obtiene el max_turns configurado para un estado.
// Devuelve 3 como fallback si no está configurado.
func (a *SalesAgent) GetMaxTurns(state SalesState) int {
	if stateCfg, ok := a.cfg.States[string(state)]; ok && stateCfg.MaxTurns > 0 {
		return stateCfg.MaxTurns
	}
	return 3 // fallback
}
