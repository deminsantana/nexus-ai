// Package scheduler implementa el motor de jobs programados de Nexus.
// Lee los jobs definidos en config.yaml bajo el bloque 'scheduler.jobs'
// y los ejecuta según la expresión cron definida.
//
// Tipos de jobs soportados:
//   - "call"          → Llamada telefónica outbound via Twilio Voice
//   - "voice_message" → Nota de voz enviada por WhatsApp/Telegram (Google TTS)
//   - "text_message"  → Mensaje de texto plano enviado por el provider activo
//
// Expresiones cron soportadas:
//   - Estándar 5 campos: "0 9 * * 1" (Lunes a las 9am)
//   - Macros: "@every 30m", "@daily", "@hourly", "@weekly"
package scheduler

import (
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/messaging"
	"nexus-core/internal/voice"

	"github.com/robfig/cron/v3"
)

// Scheduler gestiona todos los jobs programados de Nexus.
type Scheduler struct {
	cron          *cron.Cron
	jobs          []config.ScheduledJob
	voiceProvider voice.Provider
	msgProvider   messaging.Provider
}

// New crea un nuevo Scheduler con los proveedores inyectados.
// 'voiceProvider' puede ser nil si voice.provider = "none".
// 'msgProvider' es el proveedor de mensajería activo (telegram, twilio, etc.).
func New(cfg *config.Config, vp voice.Provider, mp messaging.Provider) *Scheduler {
	return &Scheduler{
		cron:          cron.New(cron.WithSeconds()),
		jobs:          cfg.Scheduler.Jobs,
		voiceProvider: vp,
		msgProvider:   mp,
	}
}

// Start registra todos los jobs y arranca el scheduler en background.
// Devuelve error si alguna expresión cron es inválida.
func (s *Scheduler) Start() error {
	if len(s.jobs) == 0 {
		fmt.Println("📅 Scheduler: no hay jobs configurados. Omitiendo.")
		return nil
	}

	for _, job := range s.jobs {
		job := job // capturar para el closure
		_, err := s.cron.AddFunc(job.Cron, func() {
			s.runJob(job)
		})
		if err != nil {
			return fmt.Errorf("expresión cron inválida para job '%s': %v", job.Name, err)
		}
		fmt.Printf("📅 Job programado: [%s] %s → %s\n", job.Cron, job.Name, job.Type)
	}

	s.cron.Start()
	fmt.Printf("✅ Scheduler iniciado con %d job(s).\n", len(s.jobs))
	return nil
}

// Stop detiene el scheduler de forma ordenada.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	fmt.Println("📅 Scheduler detenido.")
}

// runJob ejecuta un job según su tipo.
func (s *Scheduler) runJob(job config.ScheduledJob) {
	fmt.Printf("⚡ Ejecutando job: %s (tipo: %s) → %s\n", job.Name, job.Type, job.To)

	switch job.Type {
	case "call":
		s.runCallJob(job)
	case "voice_message":
		s.runVoiceMessageJob(job)
	case "text_message":
		s.runTextMessageJob(job)
	default:
		fmt.Printf("❌ Tipo de job desconocido: %s (job: %s)\n", job.Type, job.Name)
	}
}

// runCallJob inicia una llamada telefónica outbound.
func (s *Scheduler) runCallJob(job config.ScheduledJob) {
	if s.voiceProvider == nil {
		fmt.Printf("❌ Job '%s': voice provider no configurado. Activa voice.provider en config.yaml\n", job.Name)
		return
	}

	if err := s.voiceProvider.MakeCall(job.To, job.Message); err != nil {
		fmt.Printf("❌ Job '%s' error en llamada: %v\n", job.Name, err)
		return
	}
	fmt.Printf("✅ Job '%s': llamada iniciada a %s\n", job.Name, job.To)
}

// runVoiceMessageJob genera un audio TTS y lo envía como nota de voz.
func (s *Scheduler) runVoiceMessageJob(job config.ScheduledJob) {
	if s.voiceProvider == nil {
		fmt.Printf("❌ Job '%s': voice provider no configurado. Activa voice.provider en config.yaml\n", job.Name)
		// Fallback: enviar como texto
		s.runTextMessageJob(job)
		return
	}

	// Google TTS puede generar audio; Twilio no soporta bytes directos
	audioBytes, err := s.voiceProvider.TextToSpeech(job.Message, "")
	if err != nil {
		fmt.Printf("⚠️ Job '%s': TTS falló (%v). Enviando como texto...\n", job.Name, err)
		s.runTextMessageJob(job)
		return
	}

	// TODO: enviar audioBytes como nota de voz por el provider de mensajería.
	if err := s.msgProvider.SendAudio(job.To, audioBytes); err != nil {
		fmt.Printf("❌ Job '%s' error enviando audio: %v. Fallback a texto...\n", job.Name, err)
		s.runTextMessageJob(job)
		return
	}

	fmt.Printf("✅ Job '%s': nota de voz enviada a %s\n", job.Name, job.To)
}

// runTextMessageJob envía un mensaje de texto usando el provider de mensajería activo.
func (s *Scheduler) runTextMessageJob(job config.ScheduledJob) {
	if s.msgProvider == nil {
		fmt.Printf("❌ Job '%s': messaging provider no disponible\n", job.Name)
		return
	}

	if err := s.msgProvider.SendMessage(job.To, job.Message); err != nil {
		fmt.Printf("❌ Job '%s' error enviando mensaje: %v\n", job.Name, err)
		return
	}
	fmt.Printf("✅ Job '%s': mensaje enviado a %s\n", job.Name, job.To)
}
