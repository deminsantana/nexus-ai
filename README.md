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

## 🛠️ Instalación y Uso

### Requisitos

- **Go 1.21+**
- **Docker** (para Postgres y Redis en WSL)
- **API Key** de Google AI Studio (Gemini)

### Comandos Principales

1. **Verificar Servicios:**
   ```bash
   nexus status
   ```