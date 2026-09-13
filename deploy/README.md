# Despliegue como servicio

> `gestion-ciclo-tres` es un HTTP server en Go (puro, sin dependencias del sistema)
> que ya implementa *graceful shutdown* ante `SIGTERM`/`SIGINT`, ideal para correr
> como servicio nativo en **Linux (systemd)** y **Windows (NSSM)**.

Configuración (todo vía variables de entorno, ver `internal/config`):

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `:8088` | Puerto del servidor |
| `DB_PATH` | `~/.config/nicolas-sandoval/gestion-ciclo-tres/gestion-ciclo-tres.db` | Ruta del SQLite |
| `TZ` | `America/Argentina/Buenos_Aires` | Zona horaria (afecta filtro "hoy") |
| `APP_NAME` | `Gestión Ciclo Tres` | Nombre mostrado en la UI |

Healthcheck: `GET /healthz`.

---

## Linux (systemd)

### 1. Compilar (en tu máquina o en el server)

```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gestion-ciclo-tres .
```

Resulta un binario estático (portable, corre igual en cualquier distro x86-64).

> **Requisitos de Go** — `go.mod` exige Go >= 1.27.1. El script `install-linux.sh`
> exporta `GOTOOLCHAIN=auto` antes de compilar: si el Go del server es más viejo
> (por ejemplo el de `/usr/bin` que usa `sudo`, que sanitiza el PATH), Go se
> descarga solo el toolchain necesario. Esto requiere **internet** en el server
> (igual lo necesita para bajar los módulos).

### 2. Instalar (script automatizado)

```sh
sudo ./deploy/install-linux.sh
```

El script (idempotente) hace todo esto:

- Compila el binario estático.
- Crea el usuario de servicio `gestion` y `/var/lib/gestion-ciclo-tres`.
- Instala el binario en `/usr/local/bin/gestion-ciclo-tres`.
- Instala `deploy/systemd/gestion-ciclo-tres.service`, corre `systemctl daemon-reload`
  y `systemctl enable --now`.
- Verifica `http://127.0.0.1:8088/healthz`.

### 3. Comandos útiles

```sh
systemctl status gestion-ciclo-tres     # estado
journalctl -u gestion-ciclo-tres -f     # logs
systemctl restart gestion-ciclo-tres    # reiniciar (tras actualizar binario)
```

### 4. Configurar puerto / ruta / zona horaria

Editar `/etc/systemd/system/gestion-ciclo-tres.service` (bloque `Environment=...`)
y `systemctl daemon-reload && systemctl restart gestion-ciclo-tres`.

### 5. Desinstalar

```sh
systemctl disable --now gestion-ciclo-tres
rm /etc/systemd/system/gestion-ciclo-tres.service
rm /usr/local/bin/gestion-ciclo-tres
systemctl daemon-reload
```

---

## Windows (NSSM)

NSSM (Non-Sucking Service Manager, https://nssm.cc) envuelve el `.exe` sin
modificar código. El servicio queda registrado y con arranque automático.

### Requisitos

- NSSM en el `PATH` (o setear la variable `NSSM_EXE`).
- Binario para Windows, ya sea por build manual:

  ```sh
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o gestion-ciclo-tres.exe .
  ```

  o dejar que el propio script lo compile (requiere Go).

### Instalar

```bat
:: con Go en el PATH, compila y deja todo listo:
C:\gestion-ciclo-tres\deploy\install-windows.bat

:: o apuntando a un exe ya compilado:
install-windows.bat D:\app\gestion-ciclo-tres.exe
```

El script configura:

- Servicio `gestion-ciclo-tres` (Autostart) apuntando a `C:\gestion-ciclo-tres\gestion-ciclo-tres.exe`.
- `AppEnvironmentExtra` → `PORT=:8088`, `DB_PATH=C:\gestion-ciclo-tres\data\gestion-ciclo-tres.db`,
  `TZ=...`, `APP_NAME=...` (ver `internal/config`).
- STDIN/STDERR redirigidos a `C:\gestion-ciclo-tres\logs\` con rotación.

### Comandos útiles

```bat
nssm status gestion-ciclo-tres    :: estado
nssm restart gestion-ciclo-tres   :: reiniciar
nssm log gestion-ciclo-tres       :: ver logs
```

Para logs del servicio también se puede usar el Visor de eventos del Administrador
de servicios, o revisar `C:\gestion-ciclo-tres\logs\`.

> Nota: `nssm log` en versiones recientes puede no estar disponible. Alternativa:
> `nssm get gestion-ciclo-tres AppStdout` para ubicar el archivo de log.

### Desinstalar

```bat
deploy\desinstalar-windows.bat
```

---

## Actualizar a una versión nueva

En ambos casos el procedimiento es:

1. Rebuild del binario (mismo comando de build según SO).
2. Reemplazar el binario.
3. Reiniciar el servicio (`systemctl restart` / `nssm restart`).
4. Verificar `GET /healthz` en el puerto configurado.

La base de datos es SQLite persistente y no se toca en las actualizaciones;
las migraciones corren automáticamente al arrancar (embebidas en el binario).
