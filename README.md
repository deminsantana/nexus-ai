# Nexus Core 🚀

## 🤖 Tu Agente de IA Multi-Plataforma

Nexus es un agente de automatización desarrollado en **Go** que actúa como un puente inteligente entre múltiples plataformas de mensajería y modelos de lenguaje a gran escala (LLMs). Conecta **9 plataformas** desde un único binario, gestionado con un simple cambio de línea en `config.yaml`.

---

## 🗺️ Plataformas Soportadas

| Proveedor | Plataforma | Mecanismo | URL pública |
|---|---|---|---|
| `mau` | WhatsApp (no-oficial) | WebSocket — whatsmeow | ❌ No |
| `meta` | WhatsApp Business API | Webhook HTTP | ✅ Sí |
| `telegram` | Telegram | Long Polling | ❌ No |
| `discord` | Discord | Gateway WebSocket | ❌ No |
| `slack` | Slack | Socket Mode | ❌ No |
| `instagram` | Instagram DM | Meta Graph API Webhook | ✅ Sí |
| `messenger` | Facebook Messenger | Meta Graph API Webhook | ✅ Sí |
| `twilio` | SMS | Twilio REST API + Webhook | ✅ Sí (ngrok local) |
| `email` | Email (IMAP/SMTP) | Polling IMAP | ❌ No |
| `api` | API Webhook Genérico | HTTP POST (X-Nexus-API-Key) | ✅ Sí |

> **Recomendación para empezar:** `telegram` o `discord` — no requieren URL pública ni configuración de webhooks.

---

## 🏗️ Arquitectura Técnica

```
┌─────────────────────────────────────────────────────┐
│                   NEXUS CORE (Go)                    │
│                                                     │
│  ┌──────────────┐    ┌──────────────────────────┐   │
│  │  messaging/  │    │       internal/nlp/      │   │
│  │  Provider    │───►│  Brain (Gemini / OpenAI) │   │
│  │  Interface   │    │  + RAG (pgvector)        │   │
│  └──────┬───────┘    └──────────────────────────┘   │
│         │                                           │
│  ┌──────▼──────────────────────────────────────┐    │
│  │         handler.go (centralizado)           │    │
│  │  Rate Limit (Redis) → Quota (PostgreSQL)    │    │
│  │  → ProcessMessageWithContext → sendMsg()    │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
         │
    ┌────▼─────┐   ┌──────────┐   ┌──────────────┐
    │PostgreSQL│   │  Redis   │   │  Gemini /    │
    │(Historia │   │(Rate Lim.│   │  OpenAI API  │
    │ + RAG)   │   │ + Cache) │   │              │
    └──────────┘   └──────────┘   └──────────────┘
```

**Flujo de un mensaje:**
1. El proveedor activo recibe el mensaje (webhook o polling)
2. `handler.go` valida rate limit (Redis) y cuota (PostgreSQL)
3. El "Brain" busca contexto en la base de conocimientos RAG (pgvector)
4. El LLM genera la respuesta con contexto enriquecido
5. La respuesta se envía de vuelta al canal de origen

---

## 📦 Estructura del Proyecto

```
nexus/
├── cmd/nexus/main.go              # Punto de entrada
├── config.yaml                    # Tu configuración (en .gitignore)
├── config.example.yaml            # Plantilla de configuración ← copia esto
├── internal/
│   ├── api/                       # API Genérica (Webhook POST)
│   ├── cli/                       # Comandos CLI (cobra)
│   │   ├── serve.go               # nexus serve
│   │   ├── ingest.go              # nexus ingest
│   │   ├── send.go                # nexus send
│   │   ├── status.go              # nexus status
│   │   └── summarize.go           # nexus summarize
│   ├── config/config.go           # Structs de configuración YAML
│   ├── database/                  # Migraciones y queries PostgreSQL
│   ├── messaging/
│   │   ├── provider.go            # Registro y factory de proveedores
│   │   ├── handler.go             # Handler centralizado (rate limit + cuota + IA)
│   │   ├── whatsapp/
│   │   │   ├── mau.go             # WhatsApp no-oficial (whatsmeow)
│   │   │   └── meta.go            # WhatsApp Business API
│   │   ├── telegram/telegram.go   # Bot de Telegram
│   │   ├── discord/discord.go     # Bot de Discord
│   │   ├── slack/slack.go         # App de Slack (Socket Mode)
│   │   ├── instagram/instagram.go # Instagram DM
│   │   ├── messenger/messenger.go # Facebook Messenger
│   │   ├── twilio/twilio.go       # SMS via Twilio
│   │   └── email/email.go         # IMAP/SMTP
│   └── nlp/
│       ├── brain.go               # Orquestador principal de IA
│       ├── gemini.go              # Proveedor Google Gemini
│       ├── openai.go              # Proveedor OpenAI
│       └── rag.go                 # Sistema RAG con pgvector
└── knowledge/                     # Archivos .md para ingestar en RAG
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

# Copiar la plantilla de configuración
cp config.example.yaml config.yaml

# Editar con tus credenciales
# Cambia messaging.provider al canal que quieras usar
notepad config.yaml  # o tu editor preferido
```

### 2. Levantar infraestructura con Docker

```bash
docker-compose up -d
```

El `docker-compose.yml` levanta PostgreSQL (con pgvector) y Redis.

### 3. Compilar y ejecutar

```bash
# Compilar
go build -o nexus.exe ./cmd/nexus

# Encender Nexus
./nexus serve
```

### Comandos disponibles

```bash
nexus serve       # Inicia el agente en la plataforma configurada
nexus status      # Verifica conexión con IA y base de datos
nexus ingest      # Carga un archivo .md a la base de conocimientos RAG
nexus send        # Envía un mensaje manual desde la CLI
nexus summarize   # Resume la conversación reciente
nexus help-me     # Ayuda interactiva
```

---

## 🌐 API Genérica (Webhook de IA)

Nexus ofrece un endpoint universal para que cualquier aplicación externa (Web, Mobile, CRM) pueda consumir su inteligencia de forma segura.

### Endpoint
`POST /api/webhook/ai`

### Seguridad (API Key)
Debes incluir el siguiente header en tu petición:
`X-Nexus-API-Key: <tu_api_key_configurado>`

### Formato de Petición (JSON)
```json
{
  "user_id": "usuario_123",
  "message": "Hola Nexus, ¿cuál es el estado de mi pedido?"
}
```

### Formato de Respuesta (JSON)
```json
{
  "reply": "Respuesta procesada con RAG y memoria...",
  "session_id": "usuario_123"
}
```

---

## ⚙️ Configuración Global y por Plataforma

Copia `config.example.yaml` como `config.yaml` y configura la sección del proveedor que necesites.

### Configuración del Servidor y API Key
```yaml
server:
  port: 18789
  api_key: "tu_token_secreto_aquí" # Requerido para /api/webhook/ai
```

### 🔵 Telegram (recomendado para empezar)

No requiere URL pública. Usa Long Polling.

```yaml
messaging:
  provider: "telegram"
  telegram:
    bot_token: "1234567890:AAFxxxxxxxxxxxxxxxxxxx"
```

**Cómo obtener el token:**
1. Abre Telegram → busca **@BotFather**
2. Envía `/newbot` → elige nombre y @username
3. Copia el token que te entrega

---

### 🟣 Discord (sin URL pública)

Usa Gateway WebSocket. El bot responde en cualquier canal donde tenga permisos.

```yaml
messaging:
  provider: "discord"
  discord:
    bot_token: "TU_TOKEN"
    guild_id: ""   # Opcional: limita a un servidor
```

**Cómo obtener el token:**
1. [discord.com/developers](https://discord.com/developers/applications) → **New Application**
2. Sección **Bot** → **Reset Token** → copia el token
3. Activa **Message Content Intent** (Bot → Privileged Gateway Intents)
4. Invita el bot:
   ```
   https://discord.com/oauth2/authorize?client_id=TU_APP_ID&permissions=2048&scope=bot
   ```

---

### 🟡 Slack — Socket Mode (sin URL pública)

Socket Mode establece un WebSocket saliente: no necesitas abrir puertos.

```yaml
messaging:
  provider: "slack"
  slack:
    bot_token: "xoxb-..."
    app_token: "xapp-..."
    signing_secret: "..."
```

**Cómo configurar:**
1. [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → From scratch
2. **Socket Mode** → Enable → genera **App-Level Token** (`xapp-...`)
3. **OAuth & Permissions** → Bot Token Scopes: `chat:write`, `im:read`, `im:history`, `channels:history`, `users:read`
4. **Event Subscriptions** → Subscribe to bot events: `message.im`, `message.channels`
5. **Install to Workspace** → copia el **Bot User OAuth Token** (`xoxb-...`)

---

### 🟢 WhatsApp Mau (no-oficial)

```yaml
messaging:
  provider: "mau"
  # No requiere configuración extra
```

Al ejecutar `nexus serve` se mostrará un código QR para vincularlo con tu WhatsApp.

> ⚠️ Uso no oficial. Riesgo de baneo si se detecta uso masivo de spam.

---

### ⚪ WhatsApp Business API — Meta (oficial)

Requiere URL pública (o ngrok para desarrollo).

```yaml
messaging:
  provider: "meta"
  whatsapp:
    meta:
      token: "EAAN..."
      phone_number_id: "123456789"
      verify_token: "mi_verify_token"
```

**Webhook endpoint:** `POST /webhook`

---

### 🟠 Instagram DM

Requiere cuenta **Instagram Business o Creator** vinculada a una Página de Facebook.

```yaml
messaging:
  provider: "instagram"
  instagram:
    page_access_token: "TU_TOKEN"
    verify_token: "nexus_instagram_verify"
    ig_id: "TU_INSTAGRAM_BUSINESS_ID"   # ID numérico, no el @username
```

**Permisos necesarios:** `instagram_manage_messages`, `instagram_basic`, `pages_show_list`
**Webhook endpoint:** `GET|POST /webhook/instagram`

---

### 🔵 Facebook Messenger

```yaml
messaging:
  provider: "messenger"
  messenger:
    page_access_token: "TU_TOKEN"
    verify_token: "nexus_messenger_verify"
    page_id: "TU_PAGE_ID"
```

**Permisos necesarios:** `pages_messaging`, `pages_read_engagement`
**Webhook endpoint:** `GET|POST /webhook/messenger`

---

### 📱 Twilio SMS

Nexus levanta un servidor HTTP en `webhook_port` para recibir los SMS entrantes de Twilio.

```yaml
messaging:
  provider: "twilio"
  twilio:
    account_sid: "ACxxxxxxxxxxxxxxxx"
    auth_token: "TU_AUTH_TOKEN"
    from_number: "+1XXXXXXXXXX"   # Formato E.164
    webhook_port: 18790
```

**Para desarrollo local, expón el puerto con ngrok:**
```bash
ngrok http 18790
# Luego configura la URL en Twilio Console →
# Phone Numbers → Messaging → Webhook URL → https://xxxx.ngrok.io/webhook/sms
```

---

### 📧 Email (IMAP + SMTP)

El bot revisa el inbox cada N segundos buscando correos no leídos. Funciona con Gmail, Outlook, Zoho o cualquier proveedor IMAP estándar.

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

**Para Gmail:** activa **App Passwords** en `myaccount.google.com → Security → App passwords`.

---

## 🧠 Sistema RAG (Retrieval-Augmented Generation)

Para evitar alucinaciones, Nexus usa una base de conocimientos vectorial en PostgreSQL mediante `pgvector`. Los documentos se dividen en fragmentos, se procesan con el modelo de embeddings de IA y se almacenan para búsqueda por similitud de coseno.

```bash
# Ingesta tu base de conocimientos (acepta archivos Markdown)
nexus ingest --file knowledge/catalogo.md
nexus ingest --file knowledge/faq.md
```

Cada vez que llega un mensaje, Nexus recupera automáticamente los fragmentos más relevantes antes de formular la respuesta.

---

## 🛡️ Rate Limiting y Cuotas

### Rate Limit (Redis)
Protege la API de IA contra spam. Límite configurable de mensajes por segundo por usuario. Si se supera, Nexus responde automáticamente con una advertencia y descarta la solicitud.

### Cuotas Mensuales (PostgreSQL)
Cada usuario tiene un contador de mensajes procesados. Al superar el límite asignado, Nexus responde con un mensaje de alerta personalizado y deja de consumir tokens de IA, protegiendo tus costos.

---

## 🤖 Cómo hablar con Nexus

En todas las plataformas, el agente responde a mensajes que comiencen con la palabra clave **`nexus`**:

```
nexus ¿cuáles son los horarios de atención?
nexus necesito información sobre el producto X
nexus resumen de las últimas conversaciones
```

---

## 💼 Arquitectura SaaS y Modelo de Negocio

### Estructura de Costos

**Meta WhatsApp:** No cobra por mensaje individual, sino por conversación de 24 horas (~$0.01 USD). Las primeras 1,000 conversaciones de servicio mensuales son gratuitas.

**Google Gemini:** Tiene un free tier generoso. Para producción, los costos escalan con el volumen de tokens.

**Twilio SMS:** ~$0.0075 USD por SMS enviado/recibido en EE.UU. Varía por país.

### Enfoque 1: Single-Tenant (Un Contenedor por Cliente)

*Arquitectura actual — recomendada para 0 a 50 clientes.*

Un contenedor Docker aislado por cliente. Cada uno tiene su propio `config.yaml` con su token y prompt de IA.

```
VPS ($20 USD/mes — 4 GB RAM, 2 vCPUs)
├── nexus-cliente-A (15-30 MB RAM)
├── nexus-cliente-B (15-30 MB RAM)
├── ... × 40-60 clientes
├── postgres (centralizado, múltiples DBs)
└── redis (centralizado, prefijos por cliente)
```

Para escalar: agrega un VPS #2 y conecta los nuevos contenedores a la DB central.

### Enfoque 2: Multi-Tenant (Un Proceso para Todos)

*Requiere refactorización — recomendado para 50+ clientes.*

Un único binario Nexus gestiona miles de clientes. El `phone_number_id` (o equivalente en cada plataforma) identifica al cliente en la DB, que almacena su token y prompt de IA.

```
VPS (con Load Balancer)
├── nexus (1 proceso — miles de clientes)
├── postgres (multi-tenant)
└── redis
```

> **Estrategia recomendada:**
> - **Fase 1 (0–30 clientes):** Single-Tenant. Más seguro para validar el modelo de negocio.
> - **Fase 2 (+50 clientes):** Migrar a Multi-Tenant para eliminar la gestión de N contenedores.

---

## ⚖️ Licencias de Dependencias

Todas las librerías han sido auditadas. **Puedes comercializar Nexus sin restricciones.**

| Librería | Licencia | Uso comercial |
|---|---|---|
| `pgx`, `go-redis`, `cobra` | MIT / Apache 2.0 | ✅ Sin restricciones |
| `generative-ai-go`, `go-openai` | Apache 2.0 | ✅ Sin restricciones |
| `discordgo` | BSD 3-Clause | ✅ Sin restricciones |
| `slack-go` | BSD 2-Clause | ✅ Sin restricciones |
| `twilio-go` | MIT | ✅ Sin restricciones |
| `go-imap` | MIT | ✅ Sin restricciones |
| `telebot.v3` | MIT | ✅ Sin restricciones |
| `whatsmeow`, `libsignal` | MPL-2.0 | ✅ Código Nexus puede ser privado* |

> *MPL-2.0: Solo debes compartir modificaciones directas a los archivos de la librería. Tu código de Nexus puede ser completamente privado y comercial.

---

## 🔧 Agregar un Nuevo Proveedor

La arquitectura usa una interfaz `Provider` que facilita extender el sistema. Para agregar una nueva plataforma:

1. Crea el paquete en `internal/messaging/<nombre>/<nombre>.go`
2. Implementa la interfaz:
   ```go
   type Provider interface {
       Start(cfg *config.Config, dbDSN string, db *sql.DB, brain *nlp.Brain) error
       SendMessage(target string, text string) error
   }
   ```
3. Añade `SetHandler()` para inyectar el handler centralizado
4. Registra el nuevo caso en `provider.go`
5. Añade el struct de configuración en `config.go`
6. Agrega la sección en `config.example.yaml`