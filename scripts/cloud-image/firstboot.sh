#!/usr/bin/env bash
# firstboot.sh - run automatically by enterpriserag-firstboot.service the first time a new instance boots.
# Tasks: generate random secrets into .env -> start the containers -> print the credentials -> mark done + self-disable.
#
# Idempotency strategy:
#   1) Write ${MARKER} first to record that the secrets have been generated;
#      even if docker compose then fails and aborts the script, on the next boot the
#      unit's ConditionPathExists=!${MARKER} stops it regenerating secrets over .env.
#      (The old secrets are already in the postgres data volume, so regenerating them would lock the database out permanently.)
#   2) On failure the unit is marked failed and the operator can recover with docker compose up -d;
#      the credentials remain readable in ${ENV_FILE}.
set -euo pipefail

ENTERPRISERAG_DIR="${ENTERPRISERAG_DIR:-/opt/EnterpriseRag}"
ENV_FILE="${ENTERPRISERAG_DIR}/.env"
ENV_TEMPLATE="${ENTERPRISERAG_DIR}/.env.example"
CRED_FILE="/root/enterpriserag-credentials.txt"
LOG_FILE="/var/log/enterpriserag-firstboot.log"
MARKER="${ENTERPRISERAG_DIR}/.firstboot.done"

# Open LOG_FILE early and also copy stderr into the systemd journal for easier debugging
# (with a plain exec >> LOG_FILE, an early failure is invisible because stderr has already been swallowed)
mkdir -p "$(dirname "${LOG_FILE}")"
exec > >(tee -a "${LOG_FILE}") 2>&1
echo "==== firstboot started at $(date -Iseconds) ===="

if [[ -f "${MARKER}" ]]; then
  echo "marker ${MARKER} exists, skip (already initialized)"
  exit 0
fi

# cleanup.sh no longer keeps a .env, so copy the template from .env.example and substitute into it.
# This guarantees that before firstboot there is no .env holding plaintext default passwords that
# enterpriserag.service could use to initialise the postgres data volume with the wrong credentials.
if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${ENV_TEMPLATE}" ]]; then
    echo "creating ${ENV_FILE} from ${ENV_TEMPLATE}"
    cp "${ENV_TEMPLATE}" "${ENV_FILE}"
  else
    echo "ERROR: neither ${ENV_FILE} nor ${ENV_TEMPLATE} found"
    exit 1
  fi
fi

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "ERROR: docker binary not found in PATH"
  exit 1
fi

# Generate a strong 32-byte random string (for the AES-256 key; it must be exactly 32 bytes)
# Use `() ... ()` to run in a subshell with pipefail off: head closes stdin after reading N bytes,
# tr then receives SIGPIPE (exit code 141), and under `set -o pipefail` the whole pipeline counts
# as failed, which would trip the top-level `set -e` and kill firstboot.sh within milliseconds.
gen32() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 )
# General passwords: 24 characters, excluding / + = (to avoid trouble in URLs and sed substitutions)
genpw() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24 )

DB_PWD=$(genpw)
REDIS_PWD=$(genpw)
JWT=$(genpw)$(genpw)
SYS_AES=$(gen32)

# Use | as the sed delimiter to avoid clashes; only replace lines starting with KEY=
replace() {
  local key="$1" val="$2"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    sed -i "s|^${key}=.*|${key}=${val}|" "${ENV_FILE}"
  else
    echo "${key}=${val}" >>"${ENV_FILE}"
  fi
}

replace DB_PASSWORD     "${DB_PWD}"
replace REDIS_PASSWORD  "${REDIS_PWD}"
replace JWT_SECRET      "${JWT}"
replace SYSTEM_AES_KEY  "${SYS_AES}"
replace GIN_MODE        "release"

# Restore ENTERPRISERAG_REF, recorded in .cloud-image-meta during the prepare.sh stage, into .env as
# ENTERPRISERAG_VERSION; otherwise docker compose falls back to the :latest default and the image version
# no longer matches the one pulled at prepare time.
META_FILE="${ENTERPRISERAG_DIR}/.cloud-image-meta"
if [[ -f "${META_FILE}" ]]; then
  META_REF=$(grep -E '^ENTERPRISERAG_REF=' "${META_FILE}" | tail -1 | cut -d= -f2- || true)
  if [[ -n "${META_REF}" ]]; then
    replace ENTERPRISERAG_VERSION "${META_REF}"
    echo "restored ENTERPRISERAG_VERSION=${META_REF} from ${META_FILE}"
  fi
fi

# Important: write the marker as soon as .env has been updated.
# After this point, even if docker compose up fails, a reboot will not rewrite .env again,
# which prevents a mismatch with the password already persisted by postgres.
umask 077
touch "${MARKER}"
chmod 0600 "${MARKER}"

echo "env updated, marker written, starting docker compose..."
cd "${ENTERPRISERAG_DIR}"
"${DOCKER_BIN}" compose up -d

# Try the public IP, falling back to the private one
PUB_IP=$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null \
  || curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null \
  || hostname -I | awk '{print $1}')

cat >"${CRED_FILE}" <<INFO
========================================
    EnterpriseRag instance initialisation complete
    Generated at: $(date -Iseconds)
========================================

URL      : http://${PUB_IP}

To close registration once you have signed up, edit ${ENV_FILE}:
    DISABLE_REGISTRATION=true
then run:  cd ${ENTERPRISERAG_DIR} && docker compose up -d

The random credentials below were written to ${ENV_FILE}; keep them safe (root-readable only):
  DB_PASSWORD     = ${DB_PWD}
  REDIS_PASSWORD  = ${REDIS_PWD}
  JWT_SECRET      = ${JWT}
  SYSTEM_AES_KEY  = ${SYS_AES}

Notes:
    - This file only reflects the secrets at first boot; ${ENV_FILE} is authoritative afterwards.
    - Never expose infrastructure ports such as 5432 / 6379 / 9000 directly to the internet.
    - Serve only on 80 / 443, with a reverse proxy and HTTPS where required.

INFO

echo "credentials written to ${CRED_FILE}"

# Only stop the unit, do not delete the unit file (otherwise systemd may mark the running oneshot as failed).
# On the next boot, enterpriserag-firstboot.service is skipped automatically via ConditionPathExists=!${MARKER}.
systemctl disable enterpriserag-firstboot.service || true

echo "==== firstboot finished at $(date -Iseconds) ===="
