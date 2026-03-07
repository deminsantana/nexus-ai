package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
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
	dbLog := waLog.Stdout("Database", "INFO", true)

	// 1. Abrir conexión con pgx
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		fmt.Printf("Error al abrir la base de datos: %v\n", err)
		return
	}

	// 2. Validar conexión
	if err := db.Ping(); err != nil {
		fmt.Printf("\n[ERROR CRÍTICO] El ping falló: %v\n", err)
		return
	}

	// 3. Inicializar Store y EJECUTAR UPGRADE PRIMERO
	container := sqlstore.NewWithDB(db, "postgres", dbLog)

	// Esto crea las tablas whatsmeow_device, etc.
	err = container.Upgrade(context.Background())
	if err != nil {
		fmt.Printf("Error al ejecutar migraciones de WhatsApp: %v\n", err)
		return
	}

	// 4. Ahora sí, obtener el dispositivo
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		fmt.Printf("Error al obtener el dispositivo: %v\n", err)
		return
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// 5. Manejador de eventos (Captura mensajes de cualquier formato)
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Intentamos extraer el texto de varias fuentes posibles en el objeto de la mensaje
			var msgText string

			// Prioridad 1: Texto simple
			if v.Message.GetConversation() != "" {
				msgText = v.Message.GetConversation()
			} else if v.Message.GetExtendedTextMessage().GetText() != "" {
				// Prioridad 2: Mensajes con formato o respuestas
				msgText = v.Message.GetExtendedTextMessage().GetText()
			} else if v.Message.GetImageMessage().GetCaption() != "" {
				// Prioridad 3: Subtítulos de imágenes
				msgText = "[Imagen]: " + v.Message.GetImageMessage().GetCaption()
			}

			// Si logramos extraer texto, procesamos la exhibición y la base de datos
			if msgText != "" {
				sender := v.Info.Sender.User
				fmt.Printf("\n📩 WhatsApp de %s: %s\n", sender, msgText)

				// GUARDAR EN LA BASE DE DATOS (Persistencia)
				query := `INSERT INTO messages (source, sender_id, content, is_from_nexus) VALUES ($1, $2, $3, $4)`
				_, err := db.Exec(query, "whatsapp", sender, msgText, false)
				if err != nil {
					fmt.Printf("❌ Error SQL al guardar mensaje: %v\n", err)
				} else {
					fmt.Println("💾 Mensaje guardado en la memoria de Nexus.")
				}
			}
		}
	})

	// 6. Conexión y QR
	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			fmt.Printf("Error al conectar para QR: %v\n", err)
			return
		}

		fmt.Println("\n--- AUTENTICACIÓN NEXUS ---")
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				q, _ := qrcode.New(evt.Code, qrcode.Medium)
				fmt.Println(q.ToSmallString(false))
				fmt.Println("Nexus: Escanea el código arriba con tu WhatsApp para vincular.")
			case "success":
				fmt.Println("Nexus: ¡Vinculación exitosa!")
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			fmt.Printf("Error al conectar: %v\n", err)
			return
		}
		fmt.Println("Nexus: Sesión recuperada de Postgres. Conectado a WhatsApp.")
	}
}

// SendMessage conecta a WhatsApp, envía un mensaje y se cierra.
func SendMessage(dbDSN, target, text string) error {
	db, _ := sql.Open("pgx", dbDSN)
	container := sqlstore.NewWithDB(db, "postgres", nil)
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil || deviceStore == nil {
		return fmt.Errorf("no se encontró una sesión activa. Ejecuta 'nexus serve' primero")
	}

	client := whatsmeow.NewClient(deviceStore, nil)
	err = client.Connect()
	if err != nil {
		return err
	}

	// Esperar un momento a que la conexión se estabilice
	fmt.Println("⏳ Sincronizando sesión...")
	time.Sleep(3 * time.Second)

	// Limpiar el número de +, [ ] y espacios
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

	if err == nil {
		fmt.Println("🚀 Comando de envío procesado.")
	}

	// Darle un segundo extra para asegurar que el paquete salió
	time.Sleep(1 * time.Second)
	client.Disconnect()
	return err
}
