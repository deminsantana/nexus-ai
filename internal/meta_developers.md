¡Excelente decisión! Usar el número de prueba de Meta te ahorrará muchísimos dolores de cabeza al principio y te permitirá integrar y probar completamente Nexus sin afectar tus líneas reales.

Sí, el ecosistema de Meta For Developers es famoso por ser un poco enredado porque conecta los permisos de Facebook, los Business Managers y los activos de la aplicación. Pero no te preocupes, si sabes exactamente a dónde ir, en realidad toma unos pocos clics.

Aquí tienes la **guía paso a paso** para configurar tu entorno de pruebas con el número descartable:

### Paso 1: Ingresar a la Configuración de la API
1. Ve a la página de [Meta for Developers](https://developers.facebook.com/) y asegúrate de haber iniciado sesión con tu cuenta de Facebook.
2. Haz clic en **Mis aplicaciones** (My Apps) arriba a la derecha y selecciona la aplicación "Nexus" (o el nombre que le hayas puesto a la app donde estabas intentando registrar el número).
3. En el menú lateral izquierdo, busca el producto que dice **WhatsApp**. Si lo despliegas, verás una opción llamada **Configuración de la API** (API Setup). Haz clic ahí.

### Paso 2: Tu Panel de Pruebas (El lugar importante)
Una vez en la pantalla de "Configuración de la API", verás un panel dividido en secciones. Aquí es donde está la magia. Busca los siguientes elementos (no cierres esta pestaña porque vas a necesitar copiar datos):

1. **Token de acceso temporal:** Es un código muy largo. Este código servirá como contraseña (Bearer Token) para que la API de Nexus se comunique con Meta. *(Ojo: Este token expira cada 24 horas por medidas de seguridad mientras estás en pruebas. Más adelante, cuando pases a producción, generaremos uno permanente).*
2. **Enviar y recibir mensajes:** Debajo del token, verás dos campos bajo el título "Paso 1: Selecciona un número de teléfono":
   * **De (From):** Aquí debe estar seleccionado por defecto tu "Número de teléfono de prueba" (Test phone number). Se verá un número de teléfono de EE. UU. y justo debajo su **Identificador de número de teléfono** (Phone Number ID). Este ID lo necesita Nexus, cópialo.
   * **Para (To):** A la derecha verás este campo. Aquí es donde agregaremos TU número real personal para que reciba las pruebas de Nexus.

### Paso 3: Verificar tu propio número para recibir pruebas
Como estás "aislado" en zona de pruebas, Meta no te permite enviar mensajes a números indiscriminadamente para evitar el spam. Solo puedes escribir a hasta 5 números pre-autorizados.
1. En el menú desplegable de **Para (To)** haz clic en **Administrar lista de números de teléfono** o **Añadir nuevo número** (Add phone number).
2. Agrega **tu número de WhatsApp personal real** (asegúrate de poner correctamente el código de tu país sin el símbolo +).
3. Meta te enviará inmediatamente un mensaje a tu WhatsApp con un código de verificación. Es un mensaje de un número de EE.UU. u otro país con el check verde de Meta Oficial.
4. Ingresa el código en la pantalla de Meta for Developers.

*¡Listo! Tu número personal ahora tiene "permiso" de recibir mensajes desde el Bot de Nexus usando el número de prueba de Meta.*

### Paso 4: Enviar tu primer mensaje manual
Antes de ir al código de Nexus, asegúrate de que la conexión funcione nativamente:
1. En la misma pantalla de "Configuración de la API", una vez seleccionado tu número de prueba en el "De" y tu número verificado en el "Para"...
2. Sigue bajando en la página y verás un bloque de "Paso 2: Enviar un mensaje". Allí hay un ejemplo de código junto a un botón azul que dice **Enviar mensaje** (Send message).
3. **Haz clic en el botón azul.** Revisa tu celular: debería haberte llegado una bienvenida (la plantilla estándar "hello_world") desde de tu número de prueba de Meta.

---

### Paso 5: Preparar las credenciales para Nexus
Ahora que comprobamos que Meta funciona, debes tomar los 3 datos requeridos en esa página y tenerlos a mano para configurarlos en tu servidor de Nexus. Cópialos a un bloc de notas:

1. **Token de acceso temporal** (Temporary access token).
2. **Identificador del número de teléfono** (Phone number ID).
3. **Identificador de la cuenta de WhatsApp Business** (WABA ID - lo encuentras en la parte superior derecha de esa misma página o un poco debajo del identificador del número de teléfono).

Cuando estés listo para conectar tu base de código (o si quieres que yo te guíe en cómo configurar el archivo `.env` de tu proyecto o cómo levantar la API base para hacer un webhook y recibir los mensajes), ¡avísame! Ya sorteaste lo más fastidioso de Meta.