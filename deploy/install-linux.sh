#!/usr/bin/env bash
#
# Instala gestion-ciclo-tres como servicio systemd en Linux.
# Uso: sudo ./deploy/install-linux.sh
#
set -euo pipefail

BIN_NAME="gestion-ciclo-tres"
BIN_PATH="/usr/local/bin/${BIN_NAME}"
SERVICE_NAME="${BIN_NAME}.service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UNIT_SRC="${SCRIPT_DIR}/systemd/${SERVICE_NAME}"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RUN_USER="gestion"
RUN_DIR="/var/lib/gestion-ciclo-tres"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Ejecutar como root (sudo)." >&2
  exit 1
fi

if [[ ! -f "${UNIT_SRC}" ]]; then
  echo "No se encontró el unit file: ${UNIT_SRC}" >&2
  exit 1
fi

echo "==> Compilando binario estático..."
cd "${REPO_ROOT}"
# go.mod requiere Go >= 1.27.1. Si el Go del sistema es más viejo (p.ej. el de
# /usr/bin bajo sudo), GOTOOLCHAIN=auto descarga el toolchain necesario.
export GOTOOLCHAIN=auto
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${BIN_NAME}" .

echo "==> Creando usuario y directorio de datos..."
if ! id "${RUN_USER}" &>/dev/null; then
  useradd --system --home-dir "${RUN_DIR}" --no-create-home --shell /usr/sbin/nologin "${RUN_USER}"
fi
install -d -o "${RUN_USER}" -g "${RUN_USER}" -m 750 "${RUN_DIR}"

echo "==> Instalando binario..."
install -m 0755 "${BIN_NAME}" "${BIN_PATH}"

echo "==> Instalando unit file..."
install -m 0644 "${UNIT_SRC}" "${UNIT_PATH}"

echo "==> Reiniciando systemd y habilitando el servicio..."
systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"

echo "==> Estado:"
systemctl --no-pager --full status "${SERVICE_NAME}" || true

echo
echo "Verificación rápida (healthcheck):"
sleep 2
PORT_VAR="$(awk -F= '/^Environment=PORT=/{print $2; exit}' "${UNIT_PATH}")"
HOSTPORT="${PORT_VAR:-:8088}"
HOSTPORT="${HOSTPORT#:}"
if curl -fsS --max-time 5 "http://127.0.0.1:${HOSTPORT}/healthz"; then
  echo "  -> OK HTTP"
else
  echo "  -> Healthcheck falló; revisar logs: journalctl -u ${SERVICE_NAME} -f" >&2
  exit 1
fi

echo "Logs: journalctl -u ${SERVICE_NAME} -f"
echo "Desinstalar: systemctl disable --now ${SERVICE_NAME} && rm ${UNIT_PATH} && rm ${BIN_PATH}"