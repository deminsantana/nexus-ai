# Nexus Core 🚀

## 🤖 Tu Agente de IA Multi-Plataforma con Voz y Ventas

Nexus es un agente de automatización desarrollado en **Go** que actúa como puente inteligente entre múltiples plataformas de mensajería y modelos de lenguaje (LLMs). Conecta **9 plataformas** desde un único binario, con capacidades de **voz (TTS/STT)**, **llamadas programadas** y un **agente de ventas con máquina de estados**.

---

## 🚀 Novedades: RAG Inteligente v2

Nexus ha evolucionado su sistema de gestión de conocimientos para ser más autónomo y preciso:

- **👀 Folder Watching:** Monitoreo de carpetas en tiempo real. Nexus auto-ingesta cualquier cambio en archivos `.md` automáticamente.
- **🌐 Web Scraping:** Ingesta directa desde URLs. Nexus extrae el contenido de texto limpio de sitios web públicos para aprender de ellos.
- **🛡️ Filtrado por Perfil:** Segmentación de conocimientos usando `rag_tag`. Cada agente (Ventas, Soporte, Médico) solo accede a la información que le corresponde.
- **📝 Resumen IA:** Optimización de búsqueda mediante resúmenes automáticos. La IA sintetiza fragmentos largos antes de vectorizarlos para mejorar la puntería de la búsqueda semántica.

---

## 🗺️ Plataformas de Mensajería

| Proveedor | Plataforma | Mecanismo | URL pública |
|---|---|---|---|
| `mau` | WhatsApp (no-oficial) | WebSocket — whatsmeow | ❌ No |
| `meta` | WhatsApp Business API | Webhook HTTP | ✅ Sí |
| `telegram` | Telegram | Long Polling | ❌ No |
| `discord` | Discord | Gateway WebSocket | ❌ No |
| `slack` | Slack | Socket Mode | ❌ No |
| `instagram` | Instagram DM | Meta Graph API Webhook | ✅ Sí |
| `messenger` | Facebook Messenger | Meta Graph API Webhook | ✅ Sí |
| `twilio` | SMS | Twilio REST API + Webhook | ✅ Sí |
| `email` | Email (IMAP/SMTP) | Polling IMAP | ❌ No |
| `api` | API Webhook Genérico | HTTP POST (X-Nexus-API-Key) | ✅ Sí |

> **Recomendación para empezar:** `telegram` — no requiere URL pública.

---

## 🏗️ Arquitectura General

```
┌──────────────────────────────────────────────────────────────────┐
│                        NEXUS CORE (Go)                           │
│                                                                  │
│  ┌─────────────────┐   ┌──────────────────────────────────────┐  │
│  │  messaging/     │   │           internal/nlp/              │  │
│  │  Provider       │──►│  Brain (Gemini / OpenAI)             │  │
│  │  Interface      │   │  + RAG (pgvector) + STT              │  │
│  └────────┬────────┘   └──────────────────────────────────────┘  │
│           │                                                      │
│  ┌────────▼──────────────────────────────────────────────────┐   │
│  │              messaging/handler.go (centralizado)          │   │
│  │                                                           │   │
│  │   ¿sales_agent.enabled?                                   │   │
│  │     ├── SÍ → agent/SalesAgent.ProcessWithFSM()           │   │
│  │     └── NO → trigger "nexus" → Brain.ProcessWithContext() │   │
│  │                                                           │   │
│  │   Rate Limit (Redis) → Quota (PostgreSQL) → sendMsg()    │   │
│  └───────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐  ┌───────────────┐  ┌──────────────────┐  │
│  │  internal/agent/ │  │internal/voice/│  │internal/scheduler│  │
│  │  FSM de Ventas   │  │  TTS / Calls  │  │  Cron Jobs       │  │
│  │  (Redis State)   │  │  (Twilio/GCP) │  │  (robfig/cron)   │  │
│  └──────────────────┘  └───────────────┘  └──────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
         │
    ┌────▼──────┐   ┌──────────┐   ┌──────────────┐
    │PostgreSQL │   │  Redis   │   │  Gemini /    │
    │(Historia  │   │(Rate Lim.│   │  OpenAI API  │
    │ + RAG)    │   │ + Estado)│   │              │
    └───────────┘   └──────────┘   └──────────────┘
```

---

## 🎙️ Voice Agent — TTS y STT

### ¿Qué es TTS y STT?

| Sigla | Significado | En Nexus |
|-------|-------------|----------|
| **TTS** | Text-To-Speech — convierte texto en audio hablado | Nexus genera audio MP3 u OGG desde su respuesta |
| **STT** | Speech-To-Text — transcribe audio a texto | Nexus entiende notas de voz enviadas por el usuario |
| **TwiML** | Twilio Markup Language — XML que controla llamadas | Dice a Twilio qué decir en una llamada |

### Proveedores de Voz

```
                    ┌─────────────────────────────────┐
                    │       voice/provider.go          │
                    │   Interfaz VoiceProvider         │
                    └──────────────┬──────────────────┘
                                   │
               ┌───────────────────┼───────────────────┐
               ▼                   ▼                   ▼
   ┌───────────────────┐ ┌─────────────────┐  ┌──────────────┐
   │  voice/twilio/    │ │ voice/google/   │  │    "none"    │
   │  Llamadas reales  │ │ Audio MP3/OGG   │  │  Desactivado │
   │  outbound (TwiML) │ │ (Cloud TTS API) │  │  (default)   │
   └───────────────────┘ └─────────────────┘  └──────────────┘
           │                     │
    Twilio Voice API      Google Cloud TTS
    Voz: Polly.Lupe        Voz: es-ES-Standard-A
    (Amazon Polly TTS)     (WaveNet opcional)
```

### Flujo: Llamada Outbound con Twilio

```
nexus serve
    │
    ├── Scheduler activa job "call" (cron)
    │       │
    │       ▼
    │   voice/twilio.MakeCall("+584121234567", "Hola, te recuerdo tu cita...")
    │       │
    │       ▼
    │   Twilio API: CreateCall
    │       │
    │       ▼
    │   Twilio llama al número destino
    │       │
    │       ▼
    │   Usuario contesta ← TwiML: <Say voice="Polly.Lupe">Hola, te recuerdo...</Say>
    │       │
    └── Log: 📞 Llamada iniciada → +584121234567 | SID: CAxxxx
```

### Flujo: STT — Nota de voz → IA → Respuesta texto

```
Usuario envía nota de voz (OGG/Opus)
    │
    ▼
WhatsApp/Telegram entrega audio bytes
    │
    ▼
nlp/gemini.ProcessAudio(data, "audio/ogg")
    │  Prompt: "Transcribe el audio y responde"
    ▼
Gemini Vision/Audio API
    │
    ▼
[Transcripción] | [Respuesta de la IA]
    │
    ▼
sendMsg(senderID, respuesta)
```

### Configuración de Voz

```yaml
voice:
  provider: "twilio"       # twilio | google | none

  twilio:
    account_sid: "ACxxxx"
    auth_token: "xxxx"
    from_number: "+1XXXXXXXXXX"
    twiml_bin_url: "https://handler.twilio.com/twiml/XXXX"
    # Si twiml_bin_url está vacío, Nexus sirve /voice/twiml localmente

  google:
    credentials_file: "gcp-key.json"
    language: "es-ES"
    voice_name: "es-ES-Standard-A"
```

**Cómo crear un TwiML Bin en Twilio:**
1. Ve a [twilio.com/console/twiml-bins](https://www.twilio.com/console/twiml-bins)
2. Crea uno con este contenido:
   ```xml
   <Response>
     <Say language="es-MX" voice="Polly.Lupe">{{message}}</Say>
   </Response>
   ```
3. Copia la URL y pégala en `twiml_bin_url`

---

## 📅 Scheduler — Llamadas y Mensajes Programados

El scheduler usa expresiones **cron** para ejecutar tareas en horarios definidos.

### ¿Qué es una expresión cron?

```
┌──────────── segundo  (0-59)
│ ┌────────── minuto   (0-59)
│ │ ┌──────── hora     (0-23)
│ │ │ ┌────── día mes  (1-31)
│ │ │ │ ┌──── mes      (1-12)
│ │ │ │ │ ┌── día semana (0=Dom, 1=Lun ... 6=Sab)
│ │ │ │ │ │
0 0 9 * * 1   →  Lunes a las 9:00:00 AM
0 0 14 * * *  →  Todos los días a las 2:00 PM
@every 30m    →  Cada 30 minutos
@daily        →  Cada día a medianoche
```

### Tipos de Jobs

| Tipo | Requiere | Acción |
|------|----------|--------|
| `call` | `voice.provider: twilio` | Llamada telefónica real outbound |
| `voice_message` | `voice.provider: google` | Audio MP3 generado y enviado |
| `text_message` | Provider de mensajería activo | Texto plano al chat |

### Flujo del Scheduler

```
nexus serve
    │
    ├── scheduler.New(cfg, voiceProvider, msgProvider)
    │       │
    │       ├── Lee cfg.Scheduler.Jobs
    │       ├── Registra expresiones cron
    │       └── cron.Start() ← corre en background (goroutine)
    │
    │   [Al llegar la hora configurada]
    │
    ├── runJob(job) según tipo:
    │       ├── "call"          → voiceProvider.MakeCall(to, message)
    │       ├── "voice_message" → voiceProvider.TextToSpeech() → sendAudio()
    │       └── "text_message"  → msgProvider.SendMessage(to, message)
    │
    └── Log: ✅ Job 'recordatorio': mensaje enviado a +584121234567
```

### Configuración del Scheduler

```yaml
scheduler:
  enabled: true
  jobs:
    - name: "recordatorio_citas"
      cron: "0 0 9 * * 1"          # Lunes 9am
      type: "call"
      to: "+584121234567"
      message: "Hola! Te llamo para recordarte tu cita de hoy."

    - name: "seguimiento_leads"
      cron: "0 30 10 * * *"         # Diario 10:30am
      type: "text_message"
      to: "123456789"               # chat_id Telegram o JID WhatsApp
      message: "¿Tienes alguna pregunta sobre nuestros servicios?"
```

---

## 🤝 Sales Agent — Agente de Ventas con FSM

### ¿Qué es una FSM (Finite State Machine)?

Una **máquina de estados finita** es un modelo de comportamiento donde el sistema solo puede estar en **un estado a la vez**, y transiciona entre estados según reglas predefinidas. En Nexus, cada conversación de ventas tiene su propio estado persistido en Redis.

```
Estado actual + Evento → Siguiente estado
```

### Estados del Funnel de Ventas

```
  ┌─────────┐
  │  IDLE   │ ← Usuario nuevo o sin conversación activa
  └────┬────┘
       │ Primer mensaje
       ▼
  ┌──────────┐
  │ GREETING │ Saluda y pregunta cómo puede ayudar
  └────┬─────┘
       │ max_turns alcanzado o [AVANZAR]
       ▼
  ┌──────────┐
  │ QUALIFY  │ Identifica necesidades y pain points
  └────┬─────┘
       │
       ▼
  ┌─────────┐
  │ PRESENT │ Presenta el producto según las necesidades
  └────┬────┘
       │                    ┌──────────────┐
       ├───────────────────►│  OBJECTION   │ Maneja dudas y objeciones
       │                    └──────┬───────┘
       │                           │ Resuelto
       ▼                           │
  ┌───────┐ ◄─────────────────────┘
  │ CLOSE │ Propone próximo paso (demo, trial, contacto)
  └───┬───┘
      │ No cierra
      ▼
  ┌───────────┐
  │ FOLLOW_UP │ Deja la puerta abierta, datos de contacto
  └─────┬─────┘
        │
        ▼
  ┌──────┐
  │ DONE │ Conversación finalizada
  └──────┘
```

**Señales de control** — La IA incluye estas etiquetas al final de su respuesta para indicar transiciones:

| Señal | Acción |
|-------|--------|
| `[AVANZAR]` | Pasa al siguiente estado del flujo |
| `[CERRAR]` | Salta directamente a `CLOSE` |
| `[FIN]` | Va a `DONE` (conversación terminada) |

Estas señales se eliminan antes de enviar la respuesta al usuario.

### Flujo de un Mensaje en Modo Sales Agent

```
Usuario envía: "Hola, ¿qué servicios ofrecen?"
       │
       ▼
handler.go → globalSalesAgent != nil → modo FSM activo
       │
       ▼
stateStore.Get(senderID) → Estado actual: QUALIFY (turno 2/4)
       │
       ▼
SalesAgent.BuildPrompt(state) →
  "Estás en fase de calificación.
   Objetivo: identificar las necesidades...
   Turno 2 de máximo 4."
       │
       ▼
brain.GetContext(senderID) → Historial reciente de Redis
       │
       ▼
brain.Provider.Ask(fullPrompt + historial + mensaje)
       │
       ▼
Respuesta IA: "Entiendo que buscas optimizar tu soporte. 
               ¿Cuántos agentes tiene tu equipo actualmente? [AVANZAR]"
       │
       ▼
NextState: QUALIFY → PRESENT (detectó [AVANZAR])
       │
       ▼
CleanResponse: elimina "[AVANZAR]" del texto
       │
       ▼
stateStore.Save(senderID, {state: PRESENT, turns: 0})
       │
       ▼
sendMsg(senderID, "Entiendo que buscas optimizar tu soporte. 
                   ¿Cuántos agentes tiene tu equipo actualmente?")
```

### Persistencia de Estado en Redis

```
Clave:   sales:state:<senderID>
Valor:   JSON { "state": "qualify", "turns": 2, "total_turns": 5,
                "user_data": {}, "updated_at": "2026-04-23T..." }
TTL:     24 horas (conversación activa)
```

Para inspeccionar el estado de un usuario:
```bash
redis-cli GET "sales:state:123456789"
redis-cli DEL "sales:state:123456789"   # Reiniciar conversación
```

El usuario también puede escribir `reiniciar` o `reset` para empezar de nuevo.

## 🛡️ Perfiles y Habilidades (Arquitectura Modular)

Nexus ahora es modular. Puedes definir diferentes **Perfiles** y equiparlos con **Habilidades** (Skills) específicas según la necesidad del negocio.

### Habilidades Disponibles

| Habilidad | ID | Descripción |
|-----------|----|-------------|
| **Análisis de Sentimiento** | `sentiment` | Detecta la emoción del usuario (MOLESTO, ENTUSIASTA, etc.) y adapta la respuesta. |
| **Base de Conocimientos** | `rag` | Busca información precisa en tus documentos o webs subidas vía `ingest`. |

### Configuración de Perfiles y Tags

En el `config.yaml`, cada perfil puede filtrar la base de conocimientos usando `rag_tag`:

```yaml
profiles:
  active_profile: "medical"
  list:
    - id: "support"
      name: "Soporte Técnico"
      skills: ["sentiment", "rag"]
      rag_tag: "tecnico"      # Solo busca info etiquetada como 'tecnico'
      system_prompt: "Eres el soporte técnico..."
```

---

## 🤝 Sales Agent (FSM Clásico)
> **Nota:** Se mantiene por compatibilidad, pero se recomienda migrar a la nueva arquitectura de Perfiles para mayor flexibilidad.

---

## 🔄 Flujo Completo de Arranque (`nexus serve`)

```
nexus serve
    │
    ├── 1. config.LoadConfig()           Lee config.yaml
    │
    ├── 2. database.RunMigrations()      Crea tablas en PostgreSQL
    │
    ├── 3. nlp.NewBrain()               Inicia Gemini/OpenAI + Redis
    │
    ├── 4. messaging.SetConfig()         ← NUEVO
    │       └── Si sales_agent.enabled → NewSalesAgent() → FSM listo
    │
    ├── 5. messaging.InitProvider()      Inicia el canal activo
    │       └── telegram/whatsapp/discord/etc.
    │
    ├── 6. voice.InitProvider()          ← NUEVO
    │       └── twilio/google/none
    │
    ├── 7. scheduler.New().Start()       ← NUEVO
    │       └── Registra cron jobs en background
    │
    ├── 8. http.ListenAndServe()         API + TwiML endpoint
    │       ├── POST /api/webhook/ai
    │       └── GET  /voice/twiml       (si voice.provider = twilio)
    │
    └── 📌 Nexus escuchando... (Ctrl+C para detener)
```

---

## 📦 Estructura del Proyecto

```
nexus/
├── cmd/nexus/main.go
├── config.yaml
├── config.example.yaml
├── internal/
│   ├── agent/                         ← NUEVO
│   │   ├── fsm.go                     # FSM + StateStore (Redis)
│   │   └── sales_agent.go             # Orquestador del agente de ventas
│   ├── api/handler.go                 # API Webhook genérico
│   ├── cli/
│   │   ├── serve.go                   # nexus serve (actualizado)
│   │   ├── ingest.go                  # nexus ingest
│   │   ├── send.go                    # nexus send
│   │   ├── status.go                  # nexus status
│   │   └── summarize.go               # nexus summarize
│   ├── config/config.go               # Structs YAML (actualizado)
│   ├── database/                      # Migraciones y queries
│   ├── messaging/
│   │   ├── provider.go                # Factory de proveedores
│   │   ├── handler.go                 # Handler centralizado (actualizado)
│   │   ├── whatsapp/{mau,meta}.go
│   │   ├── telegram/telegram.go
│   │   ├── discord/discord.go
│   │   ├── slack/slack.go
│   │   ├── instagram/instagram.go
│   │   ├── messenger/messenger.go
│   │   ├── twilio/twilio.go
│   │   └── email/email.go
│   ├── nlp/
│   │   ├── brain.go                   # Orquestador IA
│   │   ├── gemini.go                  # Google Gemini (TTS/STT incluido)
│   │   ├── openai.go                  # OpenAI
│   │   └── rag.go                     # RAG con pgvector
│   ├── scheduler/                     ← NUEVO
│   │   └── scheduler.go               # Cron engine (robfig/cron/v3)
│   └── voice/                         ← NUEVO
│       ├── provider.go                # Interfaz VoiceProvider
│       ├── twilio/voice.go            # Llamadas outbound + TwiML
│       └── google/voice.go            # Google Cloud TTS → MP3
└── knowledge/                         # Archivos .md para RAG
```

---

## 🛠️ Instalación y Uso

### Requisitos

- **Go 1.21+**
- **Docker** (para PostgreSQL con pgvector y Redis)
- **API Key** de [Google AI Studio](https://aistudio.google.com/app/apikey) o OpenAI

### 1. Clonar y configurar

```bash
git clone https://github.com/tu-usuario/nexus.git
cd nexus
cp config.example.yaml config.yaml
# Editar config.yaml con tus credenciales
```

### 2. Levantar infraestructura

```bash
docker-compose up -d
```

### 3. Compilar y ejecutar

```bash
go build -o nexus.exe ./cmd/nexus
./nexus serve
```

### Comandos disponibles

```bash
nexus serve       # Inicia el agente (mensajería + voz + scheduler + FSM)
nexus status      # Verifica conexión con IA y base de datos
nexus ingest      # Carga archivo .md a la base de conocimientos RAG
nexus send        # Envía mensaje manual desde la CLI
nexus summarize   # Resume la conversación reciente
nexus help-me     # Ayuda interactiva
```

---

## 🤖 Modos de Respuesta

### Modo Normal (trigger `nexus`)

El agente solo responde cuando el mensaje comienza con la palabra `nexus`:

```
nexus ¿cuáles son los horarios de atención?
nexus necesito información sobre el producto X
```

### Modo Sales Agent (sin trigger)

Con `sales_agent.enabled: true`, el agente responde a **todos** los mensajes y guía la conversación por el funnel de ventas automáticamente.

---

## 🌐 API Genérica (Webhook de IA)

```
POST /api/webhook/ai
Header: X-Nexus-API-Key: <tu_api_key>
Body: { "user_id": "usuario_123", "message": "¿Cuál es el precio?" }
```

```json
{ "reply": "El plan básico es...", "session_id": "usuario_123" }
```

---

## 🧠 Sistema RAG (Retrieval-Augmented Generation)

Nexus usa **pgvector** para almacenar embeddings de documentos. Al recibir un mensaje, recupera los fragmentos más relevantes y los inyecta en el prompt antes de llamar al LLM. Esto evita alucinaciones y permite respuestas basadas en tu documentación.

```bash
### Ingesta de Conocimientos (RAG v2)

El comando `ingest` ahora soporta múltiples orígenes y optimizaciones:

```bash
# 📂 Ingesta de Archivo local con etiqueta
nexus ingest knowledge/faq.md --tag ventas

# 🌐 Ingesta desde URL (Web Scraping)
nexus ingest https://mi-sitio.com/precios --tag ventas

# 📝 Ingesta con Resumen por IA (Mejora la precisión de búsqueda)
nexus ingest knowledge/manual_largo.md --summarize

# 🧹 Limpieza total antes de subir
nexus ingest --clear knowledge/nuevo_manual.md
```

### Monitoreo Automático (Watcher)

Si no quieres ejecutar comandos manualmente, Nexus puede vigilar una carpeta:

```bash
nexus watch knowledge/
```
- Cualquier archivo `.md` que guardes o modifiques en esa carpeta se sincronizará automáticamente con la base de datos de Nexus.

### Resumen de Flags de `ingest`:
- `-c, --clear`: Borra toda la base de datos de conocimiento.
- `-t, --tag`: Etiqueta el contenido (para que solo ciertos perfiles lo usen).
- `-s, --summarize`: La IA resume cada parte antes de guardarla (más lento pero mucho más preciso).
```

---

## 🛡️ Rate Limiting y Cuotas

| Mecanismo | Tecnología | Límite |
|-----------|-----------|--------|
| **Rate Limit** | Redis (`ratelimit:<id>`) | 10 msg/segundo por usuario |
| **Cuota mensual** | PostgreSQL (`message_quotas`) | Configurable por cuenta |

---

## ⚙️ Configuración por Plataforma

### 🔵 Telegram (recomendado)

```yaml
messaging:
  provider: "telegram"
  telegram:
    bot_token: "1234567890:AAFxxx"
```
Obtén el token con **@BotFather** → `/newbot`.

### 🟣 Discord

```yaml
messaging:
  provider: "discord"
  discord:
    bot_token: "TU_TOKEN"
    guild_id: ""
```
[discord.com/developers](https://discord.com/developers/applications) → Bot → Reset Token. Activa **Message Content Intent**.

### 🟡 Slack (Socket Mode)

```yaml
messaging:
  provider: "slack"
  slack:
    bot_token: "xoxb-..."
    app_token: "xapp-..."
    signing_secret: "..."
```

### 🟢 WhatsApp Mau (no-oficial)

```yaml
messaging:
  provider: "mau"
```
Al arrancar muestra un QR para vincular con tu WhatsApp.

### ⚪ WhatsApp Business API

```yaml
messaging:
  provider: "meta"
  whatsapp:
    meta:
      token: "EAAN..."
      phone_number_id: "123456789"
      verify_token: "mi_verify_token"
```

### 📱 Twilio SMS

```yaml
messaging:
  provider: "twilio"
  twilio:
    account_sid: "ACxxxx"
    auth_token: "xxxx"
    from_number: "+1XXXXXXXXXX"
    webhook_port: 18790
```

### 📧 Email (IMAP + SMTP)

```yaml
messaging:
  provider: "email"
  email:
    imap_host: "imap.gmail.com"
    imap_port: 993
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    user: "tu_correo@gmail.com"
    password: "xxxx xxxx xxxx xxxx"   # App Password
    poll_interval_seconds: 30
```

---

## 💼 Arquitectura SaaS

### Single-Tenant (0–50 clientes)

```
VPS ($20 USD/mes — 4 GB RAM)
├── nexus-cliente-A (15-30 MB RAM)
├── nexus-cliente-B (15-30 MB RAM)
├── ... × 40-60 clientes
├── postgres (centralizado)
└── redis   (centralizado, prefijos por cliente)
```

### Multi-Tenant (50+ clientes)

```
VPS (Load Balancer)
├── nexus (1 proceso — miles de clientes)
├── postgres (multi-tenant)
└── redis
```

---

## ⚖️ Licencias

| Librería | Licencia | Uso comercial |
|---|---|---|
| `pgx`, `go-redis`, `cobra` | MIT / Apache 2.0 | ✅ |
| `generative-ai-go`, `go-openai` | Apache 2.0 | ✅ |
| `discordgo` | BSD 3-Clause | ✅ |
| `slack-go` | BSD 2-Clause | ✅ |
| `twilio-go` | MIT | ✅ |
| `go-imap` | MIT | ✅ |
| `telebot.v3` | MIT | ✅ |
| `whatsmeow` | MPL-2.0 | ✅ (código Nexus puede ser privado) |
| `robfig/cron` | MIT | ✅ |
| `cloud.google.com/go/texttospeech` | Apache 2.0 | ✅ |

---

## 🔧 Agregar un Nuevo Proveedor de Mensajería

1. Crea `internal/messaging/<nombre>/<nombre>.go`
2. Implementa la interfaz:
   ```go
   type Provider interface {
       Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error
       SendMessage(target string, text string) error
   }
   ```
3. Añade `SetHandler()` para inyectar el handler centralizado
4. Registra el caso en `provider.go` y `config.go`

## 🔧 Agregar un Nuevo Proveedor de Voz

1. Crea `internal/voice/<nombre>/voice.go`
2. Implementa la interfaz:
   ```go
   type VoiceProvider interface {
       TextToSpeech(text, lang string) ([]byte, error)
       MakeCall(to, message string) error
   }
   ```
3. Registra el caso en `voice/provider.go` y añade el struct en `config.go`