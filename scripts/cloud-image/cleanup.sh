#!/usr/bin/env bash
# cleanup.sh - wipe private data before building a cloud image.
# WARNING: this script removes SSH public keys, clears the database and logs, and then powers the machine off.
# Once it has run, create the image / snapshot / AMI from the cloud console directly; do not SSH back in.
set -euo pipefail

ENTERPRISERAG_DIR="${ENTERPRISERAG_DIR:-/opt/EnterpriseRag}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[cleanup] Please run with sudo or as root" >&2
  exit 1
fi

read -r -p "[cleanup] This cannot be undone. Continue? Type YES to proceed: " ans
if [[ "${ans}" != "YES" ]]; then
  echo "[cleanup] Cancelled"
  exit 0
fi

echo "[cleanup] 1/8 Stopping the EnterpriseRag containers"
COMPOSE_PROJECT=""
if [[ -d "${ENTERPRISERAG_DIR}" ]]; then
  cd "${ENTERPRISERAG_DIR}"
  # Prefer compose ls to get the real project name (it defaults to the lowercased directory name, e.g. enterpriserag)
  COMPOSE_PROJECT="$(docker compose ls --format json 2>/dev/null \
    | grep -oE '"Name":"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
  docker compose down -v --remove-orphans || true
fi

echo "[cleanup] 2/8 Clearing EnterpriseRag application data, the first-boot marker and the logs"
if [[ -d "${ENTERPRISERAG_DIR}" ]]; then
  rm -rf "${ENTERPRISERAG_DIR}/data"/* "${ENTERPRISERAG_DIR}/logs"/* 2>/dev/null || true
  # Deliberately not recreating .env here: leaving it absent from the image means any
  # docker compose that starts before firstboot fails because .env is missing, which
  # prevents plaintext default passwords (postgres123!@# and the like) from initialising
  # the postgres data volume. firstboot.sh copies .env.example and substitutes the secrets itself.
  rm -f "${ENTERPRISERAG_DIR}/.env" "${ENTERPRISERAG_DIR}/.firstboot.done"
fi
rm -f /root/enterpriserag-credentials.txt /var/log/enterpriserag-firstboot.log

echo "[cleanup] 3/8 Removing leftover docker volumes and build cache"
# Match strictly on the compose project name prefix so other postgres/redis volumes on the same host are untouched.
if [[ -n "${COMPOSE_PROJECT}" ]]; then
  docker volume ls -q --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" \
    | xargs -r docker volume rm -f || true
fi
# Note: this only clears volumes, stopped containers and build cache - never images.
# `docker system prune -af --volumes` was used previously and removed the
# ORG_PLACEHOLDER/enterpriserag-* images that prepare.sh had pre-pulled, so instances built
# from the image had to re-download gigabytes from Docker Hub at firstboot, defeating the point of pre-installing them.
docker container prune -f      || true
docker builder    prune -af    || true
# Only prune dangling, unmounted volumes (compose down -v has already cleared the application volumes)
docker volume     prune -f     || true

echo "[cleanup] 4/8 Clearing the system logs"
journalctl --rotate || true
journalctl --vacuum-time=1s || true
find /var/log -type f \( -name '*.log' -o -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r truncate -s 0 || true
find /var/log -type f \( -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r rm -f || true

echo "[cleanup] 5/8 Clearing SSH history and authorised keys (after this you cannot SSH back in)"
rm -f /root/.ssh/authorized_keys /root/.ssh/known_hosts /root/.bash_history
for d in /home/*; do
  [[ -d "$d" ]] || continue
  rm -f "$d/.ssh/authorized_keys" "$d/.ssh/known_hosts" "$d/.bash_history"
done
find / -xdev -type f \( -name 'id_rsa*' -o -name '*.pem' -o -name '*.key' \) \
  -not -path '/etc/ssl/*' -not -path '/usr/*' -not -path '/var/lib/docker/*' 2>/dev/null \
  | tee /tmp/cleanup-secrets-found.txt || true
echo "[cleanup]   \u2191 The entries above are suspected leftover key files; review them by hand if needed"

echo "[cleanup] 6/8 Resetting cloud-init / machine-id (so new instances get a fresh ID)"
cloud-init clean --logs --seed 2>/dev/null || true
truncate -s 0 /etc/machine-id || true
rm -f /var/lib/dbus/machine-id || true

echo "[cleanup] 7/8 Clearing apt / tmp"
if command -v apt-get >/dev/null 2>&1; then
  apt-get clean
  rm -rf /var/lib/apt/lists/*
fi
rm -rf /tmp/* /var/tmp/* /root/.cache /home/*/.cache 2>/dev/null || true

echo "[cleanup] 8/8 Syncing the disks and powering off"
history -c || true
sync
echo
echo "  Powering off shortly. Once it is down, create the image / snapshot / AMI from the cloud console."
echo
sleep 3
poweroff
