# SARP Scrapper API

Servicio Go modular basado en `svc-mannaiah`: Fx, Fiber v3, FiberZap/Zap,
configuración con `caarlos0/env` + `godotenv`, OpenAPI generado y Scalar.

## Flujo

Cuando `API_TOKEN` está configurado, el servicio usa la API oficial y consulta
`GET https://api.sarp.es/api/map/icons` para casas y almacenes, y
`GET /api/businesses/available` para negocios. Si Cloudflare presenta un desafío y
`SARP_FLARESOLVERR_URL` está configurado, solicita a FlareSolverr una cookie de
clearance y repite la llamada con esa cookie, el mismo `User-Agent` y el token
Bearer. El token nunca se envía a FlareSolverr. Si la API sigue fallando, abre
automáticamente Chrome con el perfil dedicado y espera el login/2FA manual.

Estados de `GET /status`:

- `pending`: la API todavía no respondió correctamente o falta completar el login/2FA del fallback.
- `available`: la API respondió con JSON válido o Chrome llegó a una ruta autenticada.

Una vez autenticado, el scraper carga el mapa Leaflet y combina:

- respuestas JSON de `fetch`/XHR;
- frames JSON de WebSocket;
- popups visibles con el icono `radar_propertyG`.

Los snapshots se actualizan inmediatamente al arrancar los workers y luego cada
`30m ± 5m`. Si el login se completa después del arranque, llama una vez a
`POST /houses/refresh` para no esperar la siguiente ventana.

## Configuración

```sh
cp .env.example .env
```

Variables principales:

| Variable | Default | Descripción |
| --- | --- | --- |
| `SARP_SCRAPER_ENABLED` | `true` | Activa el worker API y su fallback de Chrome. |
| `API_TOKEN` | vacío | Token Bearer oficial para `/api/map/icons`. |
| `SARP_FLARESOLVERR_URL` | vacío | Endpoint completo de FlareSolverr, por ejemplo `https://flare.niflaot.dev/v1`. |
| `SARP_FLARESOLVERR_TIMEOUT` | `90s` | Tiempo máximo para resolver el desafío. |
| `SARP_EMAIL` | vacío | Correo utilizado por el login del navegador. |
| `SARP_PASSWORD` | vacío | Contraseña utilizada por el login del navegador. |
| `SARP_SCRAPE_INTERVAL` | `30m` | Intervalo central. |
| `SARP_SCRAPE_JITTER` | `5m` | Variación uniforme positiva/negativa. |
| `LISTINGS_STATE_FILE` | `.cache/listings-state.json` | Archivo JSON persistente con decisiones y notificaciones. |
| `LISTINGS_HOUSE_MIN_VALUE` | `500000` | Valor mínimo inclusivo para mostrar y notificar propiedades. |
| `LISTINGS_BUSINESS_MIN_VALUE` | `100000` | Valor mínimo inclusivo para mostrar y notificar negocios. |
| `LISTINGS_WAREHOUSE_MIN_VALUE` | `100000` | Valor mínimo inclusivo para mostrar y notificar almacenes. |
| `DISCORD_WEBHOOK_URL` | vacío | Webhook entrante para nuevos negocios y propiedades. |
| `DISCORD_WEBHOOK_TIMEOUT` | `10s` | Timeout de cada notificación. |
| `DISCORD_BOT_TOKEN` | vacío | Token de un bot de Discord para enviar mensajes directos (avisos de conexión). |
| `PRESENCE_WATCH_NAMES` | vacío | Personajes a vigilar, separados por coma. Vacío desactiva el aviso. |
| `PRESENCE_DISCORD_USER_ID` | vacío | ID de Discord que recibe el aviso. Obligatorio si hay personajes. |
| `PRESENCE_POLL_INTERVAL` | `60s` | Cada cuánto consulta quién está conectado (mínimo `10s`). |
| `PRESENCE_WATCH_ACCOUNT_IDS` | vacío | IDs de cuenta SARP a vigilar, separados por coma. Vacío desactiva el aviso. |
| `PRESENCE_ACCOUNT_POLL_INTERVAL` | `5m` | Cada cuánto consulta `/home/online-users` (mínimo `10s`). |

El resto de URLs, selectores, timeouts y opciones del navegador usan defaults
internos; no es necesario declararlos en `.env`.

`.env` y `.sarp-browser/` están ignorados por Git. El perfil contiene cookies de
sesión y debe tratarse como un secreto. No expongas las rutas de autenticación a
Internet sin un reverse proxy autenticado.

## API

Iniciar el login sin 2FA:

```sh
curl -X POST http://127.0.0.1:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{}'
```

Si SARP solicita 2FA responde `202`:

```json
{"status":"pending"}
```

Completar la pestaña pendiente:

```sh
curl -X POST http://127.0.0.1:8080/auth/2fa \
  -H 'Content-Type: application/json' \
  -d '{"code":"123456"}'
```

También se puede enviar el código desde el primer POST:

```json
{"twoFactorCode":"123456"}
```

| Ruta | Descripción |
| --- | --- |
| `GET /status` | `available` o `pending`. |
| `POST /api/query` | Ejecuta un `GET` autenticado contra una ruta relativa de `SARP_API_BASE_URL`. |
| `POST /auth/login` | Abre Chrome, envía credenciales y opcionalmente 2FA. |
| `POST /auth/2fa` | Completa un desafío pendiente. |
| `GET /houses` | Último snapshot de casas disponibles. |
| `POST /houses/refresh` | Ejecuta un scrape sincronizado. |
| `GET /businesses` | Último snapshot de negocios disponibles. |
| `POST /businesses/refresh` | Actualiza los negocios sincronizadamente. |
| `GET /warehouses` | Último snapshot de almacenes disponibles. |
| `POST /warehouses/refresh` | Actualiza los almacenes sincronizadamente. |
| `GET /listings` | Propiedades, negocios y almacenes visibles con estado de revisión. |
| `POST /listings/:kind/:key/review` | Marca como revisado y lo mantiene visible. |
| `POST /listings/:kind/:key/ignore` | Lo oculta permanentemente. |
| `GET /panel` | Panel HTML para revisar o ignorar resultados. |
| `GET /docs` | Scalar, sólo en desarrollo. |
| `GET /openapi.json` | OpenAPI, sólo en desarrollo. |

Consulta cualquier ruta `GET` de la API oficial sin exponer el token en el
body. La ruta debe ser relativa al `SARP_API_BASE_URL` configurado y los query
params se envían por separado:

```sh
curl -X POST http://127.0.0.1:8080/api/query \
  -H 'Content-Type: application/json' \
  -d '{"path":"/character/search","query":{"name":["John"]}}'
```

Para las rutas documentadas en `api.gta-rol.com`, selecciona el proveedor fijo
`gta-rol`; utiliza el mismo `API_TOKEN` pero no comparte cookies de clearance
con `api.sarp.es`:

```sh
curl -X POST http://127.0.0.1:8080/api/query \
  -H 'Content-Type: application/json' \
  -d '{"provider":"gta-rol","path":"/businesses/1995"}'
```

No acepta URLs absolutas, redirecciones de destino ni métodos de escritura. Un
El proveedor se selecciona de una lista fija; nunca se aceptan hosts enviados
por el cliente.

## Límites deliberados

El navegador no modifica `webdriver`, no falsifica huellas, no rota IPs y no
resuelve CAPTCHA ni desafíos de Cloudflare. Si aparece uno, el login falla de
forma explícita y el estado permanece `pending`. El jitter se usa para distribuir
carga, no para evadir controles.

El cliente API envía `Authorization: Bearer`, `Accept: application/json`,
`Accept-Language` y un `User-Agent` configurable. Cuando hay un desafío puede
pedir a FlareSolverr cookies y `User-Agent`; después hace la llamada autenticada
desde el propio proceso. Por eso el scraper y FlareSolverr deben compartir la
misma IP pública de salida. El token Bearer no forma parte de la solicitud a
FlareSolverr.

## Avisos de conexión de personajes

Con `PRESENCE_WATCH_NAMES` configurado, el servicio consulta cada
`PRESENCE_POLL_INTERVAL` `GET /public/global-metrics/online` (con el mismo
cliente, token y FlareSolverr que `/api/query`) y avisa cuando uno de esos
personajes pasa de desconectado a conectado:

- Mensaje directo a `PRESENCE_DISCORD_USER_ID`, si hay `DISCORD_BOT_TOKEN`. El bot
  debe compartir servidor con esa cuenta y ella debe aceptar mensajes directos.
- Mención `<@id>` por el webhook, si hay `DISCORD_WEBHOOK_URL`.

Basta con que uno de los dos canales entregue el aviso. Al arrancar sólo se
registra quién ya está conectado, sin avisar. Una consulta fallida no cuenta como
desconexión, y un aviso fallido se reintenta en el siguiente ciclo. El estado
vive en memoria.

### Cuentas

Con `PRESENCE_WATCH_ACCOUNT_IDS` configurado, cada `PRESENCE_ACCOUNT_POLL_INTERVAL`
consulta `GET /home/online-users` y avisa por los mismos canales cuando una de esas
cuentas aparece en `payload.users`. Esa lista cubre una ventana de actividad
(`windowHours`, 4 al escribir esto), no sólo sesiones abiertas: una cuenta que ya
está en la lista no vuelve a avisar hasta que salga de ella y regrese.

```sh
PRESENCE_WATCH_ACCOUNT_IDS=13297,15316
```

Ambos avisos comparten `PRESENCE_DISCORD_USER_ID`, `DISCORD_BOT_TOKEN` y el webhook.

```sh
PRESENCE_WATCH_NAMES=Harper_Brooks,Juana_Gonzales,Yolandi_Bartlett,Orla_Murphy
PRESENCE_DISCORD_USER_ID=278642935426449418
DISCORD_BOT_TOKEN=<token del bot>
```

## Panel, caché y Discord

Cada resultado se guarda por tipo (`house`, `business` o `warehouse`) y llave en el archivo
configurado por `LISTINGS_STATE_FILE`. Un resultado nuevo empieza como
`pending`; `review` lo deja visible con la etiqueta de revisado e `ignore` lo
elimina de `/panel`, `/listings`, `/houses`, `/businesses` o `/warehouses`, incluso si vuelve a
aparecer en scrapes posteriores.

Cuando `DISCORD_WEBHOOK_URL` está configurado, cada resultado nuevo genera una
notificación con `@everyone`, nombre, valor, llave e icono. La notificación se
marca como enviada solamente después de una respuesta exitosa, por lo que los
fallos se reintentan en el siguiente refresh. El webhook real nunca debe
guardarse en Git.

## Fallback automático con Chrome

Con `API_TOKEN` configurado, el orden de cada refresh es:

1. API oficial `/map/icons`.
2. FlareSolverr y nuevo intento de la API, si está configurado.
3. Chrome dedicado, abierto automáticamente y con sesión persistente.
4. Error combinado si todas las fuentes fallan.

Cuando encuentre Cloudflare, login o 2FA, deja Chrome visible y espera hasta
cinco minutos. Al terminar la autenticación vuelve a cargar el mapa dentro del
mismo refresh. El perfil persistente vive en `.sarp-browser/` y no utiliza ni
exporta cookies del perfil personal.

## Ejecución y validación

```sh
go run ./cmd

go fmt ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
```
