package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"nexus-core/internal/config"
	"nexus-core/internal/nlp"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// handleMsg es el callback para llamar al handler centralizado.
var handleMsg func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)

// SetHandler permite al paquete messaging inyectar el handler centralizado.
func SetHandler(h func(platform, msgText, senderStr, pushName string, db *sql.DB, brain *nlp.Brain, sendMsg func(string, string) error, sendAudio func(string, []byte) error)) {
	handleMsg = h
}

type MauProvider struct {
	client *whatsmeow.Client
}

func (m *MauProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	// Silenciar logs de la base de datos y del cliente para evitar verbosidad
	dbLog := waLog.Stdout("Database", "ERROR", true)
	if cfg.Messaging.WhatsApp.SessionPath == "" {
		dbLog = waLog.Stdout("Database", "NONE", true)
	}

	container := sqlstore.NewWithDB(db, "postgres", dbLog)
	err := container.Upgrade(context.Background())
	if err != nil {
		return fmt.Errorf("error Upgrade: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("error DeviceStore: %v", err)
	}

	// Forzar silencio absoluto en whatsmeow
	nullLog := waLog.Stdout("NULL", "ERROR", true)
	m.client = whatsmeow.NewClient(deviceStore, nullLog)
	m.client.Log = waLog.Stdout("Socket", "ERROR", true)
	
	// Si quieres ver un poco de actividad del cliente ponlo en "WARN", si no "ERROR"
	waLog.Stdout("Client", "ERROR", true)

	m.client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// ── FILTRO 1: Ignorar mensajes propios ──────────────────────────
			if v.Info.IsFromMe {
				// Log interno para que sepas por qué no contestó
				// fmt.Printf("ℹ️ Mensaje de '%s' ignorado (es tu propio número)\n", v.Info.Sender.User)
				return
			}

			// ── FILTRO 2: Ignorar mensajes antiguos (ventana de 2 minutos) ──
			if time.Since(v.Info.Timestamp) > 120*time.Second {
				return
			}

			// ── FILTRO 3: Ignorar mensajes de grupos si no está habilitado ──
			if v.Info.IsGroup && !cfg.Messaging.WhatsApp.AllowGroups {
				return
			}

			// ── PROCESAR NOTA DE VOZ (STT) ──────────────────────────────────
			if v.Message.GetAudioMessage() != nil {
				audioMsg := v.Message.GetAudioMessage()
				fmt.Println("🎤 Nota de voz recibida. Descargando...")
				data, err := m.client.Download(context.Background(), audioMsg)
				if err != nil {
					fmt.Printf("❌ Error descargando audio: %v\n", err)
					return
				}
				fmt.Println("🧠 Nexus está escuchando la nota de voz...")
				reply, err := brain.Provider.ProcessAudio(data, "audio/ogg")
				if err != nil {
					fmt.Printf("⚠️ Error STT (posible cuota excedida): %v\n", err)
					m.client.SendMessage(context.Background(), v.Info.Sender, &waProto.Message{
						Conversation: proto.String("⚠️ No pude transcribir tu nota de voz en este momento. Por favor escríbeme tu mensaje."),
					})
					return
				}
				m.client.SendMessage(context.Background(), v.Info.Sender, &waProto.Message{
					Conversation: proto.String("🤖 " + reply),
				})
				return
			}

			// ── PROCESAR CONTENIDO MULTIMEDIA (OPCIONAL) ───────────────────
			msgText := ""
			if v.Message.GetConversation() != "" {
				msgText = v.Message.GetConversation()
			} else if v.Message.GetExtendedTextMessage().GetText() != "" {
				msgText = v.Message.GetExtendedTextMessage().GetText()
			}

			// Si es multimedia y queremos que la IA reaccione
			if msgText == "" && cfg.Messaging.WhatsApp.HandleMedia {
				if v.Message.GetStickerMessage() != nil {
					msgText = "[El usuario envió un Sticker]"
				} else if v.Message.GetImageMessage() != nil {
					msgText = "[El usuario envió una Imagen]"
				} else if v.Message.GetVideoMessage() != nil {
					msgText = "[El usuario envió un Video]"
				} else if v.Message.GetDocumentMessage() != nil {
					msgText = "[El usuario envió un Documento]"
				}
			}

			if msgText == "" {
				return // ignorar si no hay texto ni queremos procesar media
			}

			senderStr := v.Info.Sender.String()
			fmt.Printf("\n📩 [%s | %s]: %s\n", "whatsapp_mau", v.Info.PushName, msgText)

			if handleMsg != nil {
				handleMsg("whatsapp_mau", msgText, senderStr, v.Info.PushName, db, brain, func(targetID, text string) error {
					return m.SendMessage(targetID, text)
				}, func(targetID string, audioBytes []byte) error {
					return m.SendAudio(targetID, audioBytes)
				})
			}

		case *events.StreamReplaced:
			fmt.Println("\n⚠️ Conexión reemplazada por otra instancia. Deteniendo este nodo...")
		}
	})

	if m.client.Store.ID == nil {
		m.renderQR()
	} else {
		err = m.client.Connect()
		if err != nil {
			return fmt.Errorf("error al conectar whatsmeow: %v", err)
		}
		fmt.Println("✅ Nexus (Mau): Conexión estable y sesión recuperada.")
	}

	return nil
}

func (m *MauProvider) SendMessage(target string, text string) error {
	cleanTarget := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, target)

	jid := types.NewJID(cleanTarget, types.DefaultUserServer)

	idx := strings.Index(target, "@")
	server := types.DefaultUserServer
	if idx != -1 {
		server = target[idx+1:]
		cleanTarget = target[:idx]
	}
	jid = types.NewJID(cleanTarget, server)

	_, err := m.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(text),
	})
	return err
}

// SendAudio sube el audio a los servidores de WhatsApp y lo envía como nota de voz (PTT).
// Los bytes deben ser audio en formato OGG/Opus (generado por Google TTS con OGG_OPUS).
func (m *MauProvider) SendAudio(target string, audioBytes []byte) error {
	if m.client == nil {
		return fmt.Errorf("cliente WhatsApp no inicializado")
	}

	// 1. Subir el audio a los servidores de WhatsApp
	uploaded, err := m.client.Upload(context.Background(), audioBytes, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("error subiendo audio a WhatsApp: %v", err)
	}

	// 2. Parsear el JID del destinatario
	idx := strings.Index(target, "@")
	server := types.DefaultUserServer
	user := target
	if idx != -1 {
		user = target[:idx]
		server = target[idx+1:]
	}
	jid := types.NewJID(user, server)

	// 3. Construir el AudioMessage (PTT: true = nota de voz con forma de onda)
	fileLen := uint64(len(audioBytes))
	ptt := true
	mimetype := "audio/ogg; codecs=opus"
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &mimetype,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &fileLen,
			PTT:           &ptt,
		},
	}

	// 4. Enviar el mensaje
	_, err = m.client.SendMessage(context.Background(), jid, audioMsg)
	if err != nil {
		return fmt.Errorf("error enviando audio a WhatsApp: %v", err)
	}

	fmt.Printf("🔊 Audio enviado como nota de voz a %s (%d bytes)\n", target, len(audioBytes))
	return nil
}

func (m *MauProvider) renderQR() {
	qrChan, _ := m.client.GetQRChannel(context.Background())
	err := m.client.Connect()
	if err != nil {
		return
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			q, _ := qrcode.New(evt.Code, qrcode.Medium)
			fmt.Println(q.ToSmallString(false))
			fmt.Println("👉 Escanea para vincular Nexus (Mau API).")
		} else if evt.Event == "success" {
			fmt.Println("✅ ¡Vinculación exitosa!")
		}
	}
}

// SendMessageStatic para retrocompatibilidad con el comando 'send'.
func SendMessageStatic(dbDSN, target, text string) error {
	db, _ := sql.Open("pgx", dbDSN)
	defer db.Close()
	container := sqlstore.NewWithDB(db, "postgres", nil)
	deviceStore, _ := container.GetFirstDevice(context.Background())

	c := whatsmeow.NewClient(deviceStore, nil)
	err := c.Connect()
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)

	cleanTarget := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, target)
	jid := types.NewJID(cleanTarget, types.DefaultUserServer)
	_, err = c.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(text),
	})
	time.Sleep(1 * time.Second)
	c.Disconnect()
	return err
}
