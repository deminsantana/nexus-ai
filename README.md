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

### 1. APIs de Mensajería: Mau vs Oficial (¡Múltiples Proveedores Implementados!)
Para garantizar escalamiento y flexibilidad, Nexus soporta ambas plataformas nativamente a través de la interfaz `whatsapp.Provider`. Puedes alternar dinámicamente entre motores directamente desde tu `config.yaml`:

```yaml
whatsapp:
  provider: "meta" # Opciones: "mau" (sockets) o "meta" (HTTP webhooks)
  meta:
    token: "EAA..."
    phone_number_id: "123456789"
    verify_token: "TUTOKEN"
```

*   **API No Oficial (Mau/whatsmeow):** Ideal para crear prototipos o vender a pequeñas empresas. Al basarse en WebSockets, **no tiene costo por mensaje**. Sin embargo, conlleva riesgo de baneo por Meta si se detecta spam.
*   **API Oficial (WhatsApp Cloud / Meta):** El estándar de oro para SaaS. Escala infinitamente. Nexus encenderá automáticamente un Servidor Web en tu puerto configurado para recibir los mensajes HTTP desde Meta. **Cero riesgo de baneos** siempre que cumplas sus políticas.

### 2. Estructura de Costos de Meta
Meta **no cobra por mensaje individual**, sino por **Conversación de 24 horas**. 
Si el cliente manda un mensaje al bot, se abre una ventana de 24 horas. Durante esa ventana, tu bot puede mandarle 1 o 1,000 mensajes, y **Meta solo te cobrará ~$0.01 USD** (aprox. un céntimo, varía ligeramente por país). Las primeras 1,000 conversaciones de servicio mensuales ¡son gratis!. Tu negocio radica en revender planes mensuales fijos que engloben tus costos de IA y Meta.

### 3. Trackeo de Cuotas y Restricciones Inteligentes (Implementado)
Nexus posee un protector financiero escrito en **PostgreSQL**. Cada vez que la Inteligencia Artificial formula y envía un mensaje, se actualiza el contador del usuario en la tabla `usage_quotas`. 

**¿Qué pasa cuando se acaba la cuota?**
Si el usuario excede la cantidad de mensajes asignados (ej. límite default de 1000 mensajes de cortesía), Nexus interceptará silenciosamente los mensajes entrantes y le **responderá un mensaje de alerta automatizado** ("Has alcanzado tu límite..."), garantizando que tus costos de IA (Tokens de Gemini) no operen en pérdida.

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

---

## ☁️ Arquitectura de Despliegue SaaS para Nexus

Vender Nexus como un SaaS (Software as a Service) a múltiples clientes (clínicas, pizzerías, empresas) requiere una estrategia de despliegue clara. Debido a que el software está escrito en **Go (Golang)**, tienes una ventaja técnica enorme: los binarios de Go son extremadamente rápidos y consumen poquísima memoria.

A continuación, presento los dos enfoques principales para escalar tu negocio, el consumo esperado y la arquitectura en contenedores.

### Enfoque 1: Single-Tenant (Un Contenedor por Cliente)
*Esta es la arquitectura compatible con la versión actual de tu código.*

En este modelo, despliegas un contenedor de Docker aislado para cada cliente. Cada contenedor de Nexus tiene su propio archivo `config.yaml` cargado con el Token de Meta y el Prompt de ese cliente.

#### Arquitectura en el VPS:
- **1 Contenedor Postgres:** Centralizado. Alojas múltiples bases de datos lógicas (ej. `db_cliente1`, `db_cliente2`) dentro del mismo servidor Postgres.
- **1 Contenedor Redis:** Centralizado. Gestiona la caché de todos los clientes usando prefijos diferentes de llaves.
- **N Contenedores Nexus:** Uno por cada cliente que pague la suscripción.

#### Escalabilidad (¿Cuántos clientes por VPS?)
Un binario de Go de esta naturaleza consume en reposo entre **15 MB y 30 MB de RAM**. Si consideramos un VPS estándar de **$10 a $20 USD mensuales (con 4 GB de RAM y 2 vCPUs)**:
- Podrías alojar fácilmente entre **40 y 60 contenedores Nexus (clientes)** de forma simultánea.
- El límite no será el CPU, sino la RAM asignada y los límites de conexiones de la base de datos Postgres.

#### ¿Cómo se escala?
Cuando llegues a los ~50 clientes y la RAM de tu VPS esté al 80%, aplicas "Escalamiento Horizontal":
1. Alquilas un **VPS #2** (Worker).
2. Instalás Docker.
3. Despliegas los clientes 51 al 100 en los contenedores de ese VPS, pero conectándolos (a través de la red) a la base de datos principal o a una réplica.

---

### Enfoque 2: True Multi-Tenant (El Santo Grial del SaaS)
*Requiere una refactorización considerable del código actual.*

En este modelo, en lugar de arrancar un contenedor por cliente, **el mismo contenedor de Nexus procese a miles de clientes al mismo tiempo**. 

#### ¿Cómo funciona con Meta API?
1. En tu App de Meta Developers, configuras **una única URL de Webhook** (ej. `api.tuservicio.com/webhook`).
2. Agregas los números de WhatsApp de los 1000 clientes a tu App de Meta.
3. Cuando llega un mensaje, el JSON de Meta incluye el `phone_number_id` (el número del cliente que recibió el mensaje).
4. Nexus captura el mensaje, **busca ese ID en la tabla de PostgreSQL (`integrations`)**, recupera en memoria caliente (Redis) el Token y el Prompt de IA de ese número, le envía la conversación a Gemini, y contesta.

#### Arquitectura en el VPS:
- **1 Contenedor Postgres:** Base de datos multi-inquilino.
- **1 Contenedor Redis:** Para caché, rate limits y manejo de estado.
- **1 Contenedor Nexus (API Central):** Un único binario de Go.

#### Escalabilidad (¿Cuántos clientes por VPS?)
Con un diseño Multi-Tenant, un servidor Go puede manejar miles de conexiones concurrentes por segundo. En el mismo VPS de **4 GB de RAM ($20 USD/mes)**:
- Podrías escalar hasta **200, 500 o incluso 1000 clientes** de mediano volumen.
- Aquí los cuellos de botella son el disco (I/O) de Postgres y los límites de solicitud de API (Quotas) impuestos por la API de Google Gemini (AI Studio).

Al requerir más poder, puedes montar un *Load Balancer* (Nginx/Traefik) frente a 3 contenedores Nexus operando sobre la misma base de datos, logrando escalabilidad global.

---

### Recomendación Estratégica

> **Fase 1 (0 a 30 clientes):** Utiliza el **Enfoque Single-Tenant**. Es más seguro aislar a los clientes mientras validas tu modelo de negocio; si un contenedor de un cliente "cae" o comete un error crítico, los otros 29 clientes no se enteran de nada. Mapea distintos puertos (ej: `:18001`, `:18002`, etc.) en tu VPS para los webhooks, o usa un proxy invertido (Nginx) para enrutarlos según las rutas (`/cliente1/webhook`).
> 
> **Fase 2 (+50 clientes):** El nivel logístico de usar `docker-compose.yaml` u Orquestadores con decenas de configuraciones se volverá una pesadilla. En este punto, inviertes capital en transformar el código a **True Multi-Tenant** y pasas toda la configuración empresarial a PostgreSQL.