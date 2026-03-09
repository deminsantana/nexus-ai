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

func StartClient(dbDSN string) {
	// 1. Configuración de Logs y DB
	dbLog := waLog.Stdout("Database", "ERROR", true)
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		fmt.Printf("❌ Error DB: %v\n", err)
		return
	}

	// 2. Inicializar Almacén de Sesiones
	container := sqlstore.NewWithDB(db, "postgres", dbLog)
	err = container.Upgrade(context.Background())
	if err != nil {
		fmt.Printf("❌ Error Upgrade: %v\n", err)
		return
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		fmt.Printf("❌ Error DeviceStore: %v\n", err)
		return
	}

	// 3. Configurar Cliente
	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// 4. Inicializar Cerebro (NLP)
	cfg := config.LoadConfig()
	brain, err := nlp.NewBrain(cfg)
	if err != nil {
		fmt.Printf("❌ Error Cerebro: %v\n", err)
	}

	// 5. Manejador de Eventos
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			handleIncomingMessage(client, v, db, brain)

			if v.Message.GetAudioMessage() != nil {
				audioMsg := v.Message.GetAudioMessage()
				fmt.Println("🎤 Nota de voz recibida. Descargando...")

				// Descargar los bytes del audio desde los servidores de WhatsApp
				data, err := client.Download(context.Background(), audioMsg)
				if err != nil {
					fmt.Printf("❌ Error descargando audio: %v\n", err)
					return
				}

				// Procesar con el cerebro
				fmt.Println("🧠 Nexus está escuchando la nota de voz...")
				// WhatsApp usa "audio/ogg; codecs=opus"
				reply, err := brain.Provider.ProcessAudio(data, "audio/ogg")
				if err != nil {
					fmt.Printf("❌ Error de IA en audio: %v\n", err)
					return
				}

				// Responder por texto (por ahora)
				client.SendMessage(context.Background(), v.Info.Sender, &waProto.Message{
					Conversation: proto.String("🤖 Escuché tu nota de voz: " + reply),
				})
			}

		case *events.StreamReplaced:
			// Manejo de colisión manual
			fmt.Println("\n⚠️ Conexión reemplazada por otra instancia. Deteniendo este nodo...")
		}
	})

	// 6. Lógica de Conexión / QR
	if client.Store.ID == nil {
		renderQR(client)
	} else {
		err = client.Connect()
		if err != nil {
			fmt.Printf("❌ Error al conectar: %v\n", err)
			return
		}
		fmt.Println("✅ Nexus: Conexión estable y sesión recuperada.")
	}
}

func handleIncomingMessage(client *whatsmeow.Client, v *events.Message, db *sql.DB, brain *nlp.Brain) {
	msgText := ""
	if v.Message.GetConversation() != "" {
		msgText = v.Message.GetConversation()
	} else if v.Message.GetExtendedTextMessage().GetText() != "" {
		msgText = v.Message.GetExtendedTextMessage().GetText()
	}

	if msgText == "" {
		return
	}

	senderStr := v.Info.Sender.String()
	// Usamos PushName para ver quién escribe en consola
	fmt.Printf("\n📩 [%s]: %s\n", v.Info.PushName, msgText)

	// Persistencia en Postgres
	query := `INSERT INTO messages (source, sender_id, content, is_from_nexus) VALUES ($1, $2, $3, $4)`
	db.Exec(query, "whatsapp", senderStr, msgText, false)

	// Lógica de Respuesta IA con filtro "Nexus"
	if strings.HasPrefix(strings.ToLower(msgText), "nexus") {
		fmt.Println("🧠 Procesando con Gemini...")
		cleanInput := strings.TrimPrefix(strings.ToLower(msgText), "nexus")

		reply, err := brain.ProcessMessage(cleanInput)
		if err != nil {
			fmt.Printf("❌ Error IA: %v\n", err)
			return
		}

		// Enviar respuesta al JID original
		_, err = client.SendMessage(context.Background(), v.Info.Sender, &waProto.Message{
			Conversation: proto.String(reply),
		})

		if err == nil {
			fmt.Printf("🤖 Nexus dice: %s\n", reply)
		} else {
			fmt.Printf("❌ Error al enviar respuesta: %v\n", err)
		}
	}
}

func renderQR(client *whatsmeow.Client) {
	qrChan, _ := client.GetQRChannel(context.Background())
	err := client.Connect()
	if err != nil {
		return
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			q, _ := qrcode.New(evt.Code, qrcode.Medium)
			fmt.Println(q.ToSmallString(false))
			fmt.Println("👉 Escanea para vincular Nexus.")
		} else if evt.Event == "success" {
			fmt.Println("✅ ¡Vinculación exitosa!")
		}
	}
}

func SendMessage(dbDSN, target, text string) error {
	db, _ := sql.Open("pgx", dbDSN)
	container := sqlstore.NewWithDB(db, "postgres", nil)
	deviceStore, _ := container.GetFirstDevice(context.Background())

	client := whatsmeow.NewClient(deviceStore, nil)
	err := client.Connect()
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
	_, err = client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(text),
	})

	time.Sleep(1 * time.Second)
	client.Disconnect()
	return err
}
