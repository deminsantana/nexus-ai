package email

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/smtp"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// handleMsg es inyectado desde el paquete messaging para usar el handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error)) {
	handleMsg = h
}

// EmailProvider implementa la interfaz messaging.Provider usando IMAP (recepción) y SMTP (envío).
// Revisa el inbox periódicamente y responde a los correos que contengan "nexus" en el asunto o cuerpo.
type EmailProvider struct {
	IMAPHost     string
	IMAPPort     int
	SMTPHost     string
	SMTPPort     int
	User         string
	Password     string
	PollInterval int // segundos entre revisiones del inbox
	db           *sql.DB
	brain        *nlp.Brain
	// Control de mensajes ya procesados
	processedUIDs map[uint32]bool
}

func (e *EmailProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	e.db = db
	e.brain = brain
	e.processedUIDs = make(map[uint32]bool)

	if e.PollInterval <= 0 {
		e.PollInterval = 30 // 30 segundos por defecto
	}

	fmt.Printf("✅ Nexus (Email): Iniciando polling IMAP cada %ds → %s\n", e.PollInterval, e.User)

	go e.pollLoop()
	return nil
}

// pollLoop revisa el inbox periódicamente en background.
func (e *EmailProvider) pollLoop() {
	ticker := time.NewTicker(time.Duration(e.PollInterval) * time.Second)
	defer ticker.Stop()

	// Primera ejecución inmediata
	e.checkInbox()

	for range ticker.C {
		e.checkInbox()
	}
}

// checkInbox se conecta al servidor IMAP y procesa los mensajes no leídos.
func (e *EmailProvider) checkInbox() {
	addr := fmt.Sprintf("%s:%d", e.IMAPHost, e.IMAPPort)

	// Conexión IMAP con TLS
	tlsCfg := &tls.Config{ServerName: e.IMAPHost}
	c, err := client.DialTLS(addr, tlsCfg)
	if err != nil {
		fmt.Printf("❌ [Email] Error conectando al servidor IMAP: %v\n", err)
		return
	}
	defer c.Logout()

	// Autenticación
	if err := c.Login(e.User, e.Password); err != nil {
		fmt.Printf("❌ [Email] Error de autenticación IMAP: %v\n", err)
		return
	}

	// Seleccionar INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		fmt.Printf("❌ [Email] Error seleccionando INBOX: %v\n", err)
		return
	}

	if mbox.Messages == 0 {
		return
	}

	// Buscar mensajes no leídos
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	uids, err := c.Search(criteria)
	if err != nil || len(uids) == 0 {
		return
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid}

	messages := make(chan *imap.Message, 10)
	go func() {
		c.Fetch(seqSet, items, messages)
	}()

	for msg := range messages {
		if msg == nil {
			continue
		}
		// Evitar reprocesar el mismo UID
		if e.processedUIDs[msg.Uid] {
			continue
		}
		e.processedUIDs[msg.Uid] = true

		e.processEmail(msg, section)
	}
}

// processEmail extrae remitente y cuerpo de un mensaje IMAP y lo pasa al handler.
func (e *EmailProvider) processEmail(msg *imap.Message, section *imap.BodySectionName) {
	if msg.Envelope == nil {
		return
	}

	// Remitente
	from := ""
	pushName := ""
	if len(msg.Envelope.From) > 0 {
		from = msg.Envelope.From[0].Address()
		pushName = msg.Envelope.From[0].PersonalName
		if pushName == "" {
			pushName = from
		}
	}
	if from == "" {
		return
	}

	// Asunto como contexto
	subject := msg.Envelope.Subject

	// Leer el cuerpo del correo
	body := section.FetchItem()
	r := msg.GetBody(section)
	if r == nil {
		return
	}

	buf := new(strings.Builder)
	bodyBytes := make([]byte, 0, 4096)
	tmp := make([]byte, 512)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			bodyBytes = append(bodyBytes, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	_ = body
	_ = buf

	// Extraer texto plano del body (simplificado)
	rawBody := string(bodyBytes)
	textBody := extractPlainText(rawBody)

	// Combinar asunto + cuerpo para el handler
	msgText := fmt.Sprintf("[Asunto: %s] %s", subject, textBody)

	if handleMsg != nil {
		handleMsg("email", msgText, from, pushName, e.db, e.brain, func(targetID, replyText string) error {
			return e.SendMessage(targetID, replyText)
		})
	}
}

// extractPlainText intenta extraer texto legible de un raw email body.
func extractPlainText(raw string) string {
	// Separar headers del cuerpo (doble CRLF)
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	if len(parts) == 2 {
		body := parts[1]
		// Limpiar boundary markers de multipart
		if idx := strings.Index(body, "--"); idx > 0 {
			body = body[:idx]
		}
		return strings.TrimSpace(body)
	}
	return strings.TrimSpace(raw)
}

// SendMessage envía un correo electrónico via SMTP.
// 'target' debe ser la dirección de email del destinatario.
func (e *EmailProvider) SendMessage(target string, text string) error {
	addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)
	auth := smtp.PlainAuth("", e.User, e.Password, e.SMTPHost)

	subject := "Re: Respuesta de Nexus AI"
	body := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		subject, e.User, target, text)

	return smtp.SendMail(addr, auth, e.User, []string{target}, []byte(body))
}
