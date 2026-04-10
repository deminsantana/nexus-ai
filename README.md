# Nexus Core 🚀

## 🤖**Tu Asistente Personal con IA Integrada para WhatsApp**

Nexus es un agente de automatización desarrollado en **Go** que actúa como un puente inteligente entre la mensajería en tiempo real y modelos de lenguaje a gran escala (LLM). Este proyecto representa mi **primera incursión integrando IA Generativa** en arquitecturas de software robustas.

---

## 🏗️ Arquitectura Técnica

El sistema se basa en una arquitectura de micro-servicios desacoplados pero integrados mediante un núcleo central en Go:

* **Engine (Go):** Gestiona el ciclo de vida de los eventos de mensajería.
* **Memoria a Largo Plazo (PostgreSQL):** Almacena sesiones de WhatsApp y el historial de mensajes para análisis posterior.
* **Memoria de Corto Plazo (Redis):** (En implementación) Utilizada para caché de estados y gestión de contextos rápidos.
* **Cerebro (Gemini AI):** Procesamiento de Lenguaje Natural (NLP) para resúmenes y respuestas inteligentes.

---

## 📚 Teoría para el Desarrollador

### ¿Qué es NLP (Natural Language Processing)?

Es la disciplina que permite a Nexus no solo leer texto, sino extraer intención. Utilizamos modelos de **IA Generativa** para transformar logs de chat desestructurados en resúmenes ejecutivos.

### Event-Driven Architecture (EDA)

Nexus funciona bajo un modelo de eventos. Cada vez que llega un paquete de datos desde los servidores de WhatsApp, se dispara un evento que es capturado, guardado en DB y procesado asíncronamente por el paquete `internal/nlp`.

---

### Sistema RAG (Retrieval-Augmented Generation)

Para evitar alucinaciones, Nexus utiliza una Base de Conocimientos indexada en **PostgreSQL** mediante la extensión `pgvector`. Los documentos (ej. un catálogo en Markdown) se dividen en fragmentos, se procesan usando el modelo de incrustaciones de IA (`gemini-embedding-001`) y se inyectan como conocimiento duro. Cada vez que alguien pregunta en el chat, Nexus busca por Similitud de Coseno la respuesta más relevante antes de formular sus palabras.

### Limitador de Tasa (Rate Limiting)

Para proteger a la API de la IA de spam (Denegación de Servicio), la memoria en **Redis** registra los mensajes. Si un cliente de WhatsApp manda más de 10 mensajes repetitivos en un margen de 1 segundo, Nexus los interceptará, alertará al usuario y no desperdiciará capacidad de la IA mitigando gastos imprevistos.

---

## 💼 Estrategia SaaS y Modelo de Negocio

Si tu objetivo es comercializar Nexus y venderlo como un servicio automatizado (SaaS) a empresas, clínicas, pizzerías o corporaciones, aquí tienes los pilares de arquitectura de negocio:

### 1. APIs de Mensajería: Mau vs Oficial
Para garantizar el mejor servicio a tus clientes, existen dos enfoques técnicos:
*   **API No Oficial (whatsmeow - La Actual):** Ideal para crear prototipos, hacer demostraciones o vender a pequeñas empresas de bajo volumen. Al estar basada en ingeniería inversa sobre WebSockets, **no tiene costo por mensaje**. Sin embargo, conlleva riesgo de bloqueo de número (baneo por Meta) si se detecta spam o comportamiento abusivo automatizado.
*   **API Oficial (WhatsApp Cloud / Business API):** El estándar de oro para SaaS. Escala infinitamente. La transición en el código es sencilla mediante una Interfaz en Go (`whatsapp.Provider`). Se le cobra a Meta usando tu propia Tarjeta de Crédito y tú asumes la infraestructura para darle a tu cliente un "Todo Incluido". **Cero riesgo de baneos** siempre que se cumplan las políticas empresariales de Meta.

### 2. Estructura de Costos de Meta
Meta **no cobra por mensaje individual**, sino por **Conversación de 24 horas**. 
Si el cliente manda un mensaje al bot, se abre una ventana de 24 horas. Durante esa ventana, tu bot puede mandarle 1 o 1,000 mensajes, y **Meta solo te cobrará ~$0.01 USD** (aprox. un céntimo, varía ligeramente por país). Las primeras 1,000 conversaciones de servicio mensuales ¡son gratis!. Tu negocio radica en revender planes mensuales fijos que engloben tus costos de IA y Meta.

### 3. Trackeo de Cuotas (Rate Limiting y Billing)
Es mandatorio contabilizar el consumo de cada cliente usando tablas en **PostgreSQL**. Por ninguna razón se deben medir cuotas mensuales de los clientes utilizando archivos de configuración estáticos como `config.yaml`. Esa es una mala práctica de software. Todo conteo financiero va en BD.

---

## ⚖️ Asesoría Legal sobre Dependencias (Licencias)

Todas las librerías incluidas en `go.mod` han sido auditadas. **Conclusión: Estás 100% legal y libre de riesgo financiero o de demanda para comercializar software privado de código cerrado.**

*   **MIT / Apache 2.0 / BSD:** (Usadas en `pgx`, `go-redis`, `cobra`, `generative-ai-go`, `go-openai`). Te permiten usarlas, venderlas y lucrar sin retribuir regalías a los autores ni abrir el código fuente de Nexus.
*   **MPL-2.0 (Mozilla Public License):** (Usada en `whatsmeow` y `libsignal`). Es amigable para usos comerciales. Tu código de Golang de "Nexus" puede ser ultra secreto y millonario, bajo restricción de que si llegas a modificar deliberadamente *los archivos internos propios de la librería de tulir*, deberás contribuir a la comunidad compartiendo gratuitamente únicamente esa corrección. De lo contrario, puedes operar con todo tu SaaS privado.

---

## 🛠️ Instalación y Uso

### Requisitos

- **Go 1.21+**
- **Docker** (para Postgres y Redis en WSL, recomendamos la imagen `pgvector/pgvector:pg16` para la carga vectorial).
- **API Key** de Google AI Studio (Gemini)

### Comandos Principales

1. **Verificar y Operar Servicios:**
   ```bash
   nexus status      # Verifica la IA y la Base de Datos
   nexus ingest      # Lee un .md y carga la base de conocimientos RAG
   nexus serve       # Enciende al bot en WhatsApp
   nexus help-me     # Muestra ayuda interactiva
   nexus send        # Dispara un mensaje manual
   nexus summarize   # Resume la charla reciente
   ```