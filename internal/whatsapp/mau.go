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

type MauProvider struct {
	client *whatsmeow.Client
}

func (m *MauProvider) Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error {
	dbLog := waLog.Stdout("Database", "ERROR", true)

	container := sqlstore.NewWithDB(db, "postgres", dbLog)
	err := container.Upgrade(context.Background())
	if err != nil {
		return fmt.Errorf("error Upgrade: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("error DeviceStore: %v", err)
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	m.client = whatsmeow.NewClient(deviceStore, clientLog)

	m.client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			msgText := ""
			if v.Message.GetConversation() != "" {
				msgText = v.Message.GetConversation()
			} else if v.Message.GetExtendedTextMessage().GetText() != "" {
				msgText = v.Message.GetExtendedTextMessage().GetText()
			}

			senderStr := v.Info.Sender.String()

			HandleIncomingMessage(msgText, senderStr, v.Info.PushName, db, brain, func(targetID, text string) error {
				return m.SendMessage(targetID, text) // Callback para enviar respuesta usando MauProvider
			})

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
					fmt.Printf("❌ Error de IA en audio: %v\n", err)
					return
				}
				m.client.SendMessage(context.Background(), v.Info.Sender, &waProto.Message{
					Conversation: proto.String("🤖 Escuché tu nota de voz: " + reply),
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
	
	// Si el target incluía ".net" o cosas de Jid original, extrae el numero.
	// Recompone JID
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

// Para retrocompatibilidad de uso aislado en send.go
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
