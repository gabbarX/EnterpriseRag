#!/usr/bin/env bash
# prepare.sh - deploy the EnterpriseRag runtime on a clean Linux instance, for baking a cloud image template.
# It does not clone the whole EnterpriseRag repository; it downloads only the 4 runtime files (~100KB).
# Compatible with: Ubuntu / Debian / CentOS / Rocky and other distributions with systemd + Docker.
# Usage:  sudo bash prepare.sh
# Tunable environment variables:
#   ENTERPRISERAG_REF              git ref to fetch (tag / branch / commit), default main
#   ENTERPRISERAG_DIR              deployment directory, default /opt/EnterpriseRag
#   ENTERPRISERAG_REPO             repository URL, default https://github.com/ORG_PLACEHOLDER/EnterpriseRag
#   ENTERPRISERAG_GH_PROXY         optional GitHub download prefix, default empty. Set it when the
#                            instance cannot reach github.com directly.
#                            (the download URL becomes ${ENTERPRISERAG_GH_PROXY}${ENTERPRISERAG_REPO}/archive/...)
#   DOCKER_INSTALL_MIRROR    optional Docker package repository, default empty (uses get.docker.com).
#                            Set it to a docker-ce apt repository URL when get.docker.com is not
#                            reachable, for example https://download.docker.com/linux/ubuntu.
#                            Installation then goes through apt against that repository and pulls
#                            docker-ce / containerd.io / docker-compose-plugin without ever
#                            contacting get.docker.com. Only apt-based distributions are supported.
#   DOCKER_REGISTRY_MIRROR   optional Docker Hub pull-through mirror, default empty.
#                            (it is written to /etc/docker/daemon.json and docker is restarted)
#   PRUNE_OLD_IMAGES         whether to clean up dangling / old-tag images on upgrade,
#                            default false. When set to true, images with no container
#                            referencing them (including ORG_PLACEHOLDER/enterpriserag-* from an
#                            older ENTERPRISERAG_VERSION) are removed after the new images are
#                            pulled, to keep the baked cloud image smaller.
set -euo pipefail

ENTERPRISERAG_REF="${ENTERPRISERAG_REF:-main}"
ENTERPRISERAG_DIR="${ENTERPRISERAG_DIR:-/opt/EnterpriseRag}"
ENTERPRISERAG_REPO="${ENTERPRISERAG_REPO:-https://github.com/ORG_PLACEHOLDER/EnterpriseRag}"
ENTERPRISERAG_GH_PROXY="${ENTERPRISERAG_GH_PROXY:-}"
DOCKER_INSTALL_MIRROR="${DOCKER_INSTALL_MIRROR:-}"
DOCKER_REGISTRY_MIRROR="${DOCKER_REGISTRY_MIRROR:-}"
PRUNE_OLD_IMAGES="${PRUNE_OLD_IMAGES:-false}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[prepare] please run with sudo or as root" >&2
  exit 1
fi

# Install the full docker-ce set (including compose-plugin) through apt from the given repository.
# Used when the instance cannot reach get.docker.com directly.
install_docker_via_apt_mirror() {
  local mirror="$1"
  if ! command -v apt-get >/dev/null 2>&1; then
    echo "[prepare] DOCKER_INSTALL_MIRROR currently supports apt-based distributions only (Ubuntu/Debian)" >&2
    return 1
  fi
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "${mirror%/}/gpg" | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  local arch codename
  arch="$(dpkg --print-architecture)"
  codename="$(lsb_release -cs)"
  echo "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.gpg] ${mirror%/} ${codename} stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io \
                     docker-buildx-plugin docker-compose-plugin curl tar
}

echo "[prepare] 1/6 installing Docker and dependencies"
if ! command -v docker >/dev/null 2>&1; then
  if [[ -n "${DOCKER_INSTALL_MIRROR}" ]]; then
    echo "[prepare]   installing docker-ce via apt from ${DOCKER_INSTALL_MIRROR} (skipping get.docker.com)"
    install_docker_via_apt_mirror "${DOCKER_INSTALL_MIRROR}"
  else
    curl -fsSL https://get.docker.com | bash
  fi
fi
systemctl enable --now docker

if ! docker compose version >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y docker-compose-plugin curl tar
  elif command -v yum >/dev/null 2>&1; then
    yum install -y docker-compose-plugin curl tar
  fi
fi

# Optional: configure a Docker Hub pull-through mirror, for hosts where a direct connection to
# registry-1.docker.io times out. daemon.json is only touched when the user passes the variable.
if [[ -n "${DOCKER_REGISTRY_MIRROR}" ]]; then
  echo "[prepare] 1.5/6 configuring Docker Hub mirror: ${DOCKER_REGISTRY_MIRROR}"
  mkdir -p /etc/docker
  # Merge into an existing daemon.json through python so other user settings are not overwritten
  if [[ -s /etc/docker/daemon.json ]] && command -v python3 >/dev/null 2>&1; then
    python3 - "$DOCKER_REGISTRY_MIRROR" <<'PY'
import json, sys, pathlib
p = pathlib.Path("/etc/docker/daemon.json")
mirror = sys.argv[1]
try:
    cfg = json.loads(p.read_text())
except Exception:
    cfg = {}
mirrors = cfg.get("registry-mirrors") or []
if mirror not in mirrors:
    mirrors.insert(0, mirror)
cfg["registry-mirrors"] = mirrors
p.write_text(json.dumps(cfg, indent=2) + "\n")
PY
  else
    cat >/etc/docker/daemon.json <<EOF
{
  "registry-mirrors": ["${DOCKER_REGISTRY_MIRROR}"]
}
EOF
  fi
  systemctl restart docker
  # Give the docker daemon a moment to come back, so the compose pull below does not hit EOF
  for _ in 1 2 3 4 5; do
    docker info >/dev/null 2>&1 && break
    sleep 1
  done
fi

echo "[prepare] 2/6 fetching EnterpriseRag runtime files (ref=${ENTERPRISERAG_REF})"
# Download only the 4 files actually needed instead of cloning the whole repository (MBs -> KBs)
mkdir -p "${ENTERPRISERAG_DIR}/config"

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

tarball_url="${ENTERPRISERAG_GH_PROXY}${ENTERPRISERAG_REPO}/archive/${ENTERPRISERAG_REF}.tar.gz"
echo "[prepare]   tarball: ${tarball_url}"
curl -fsSL "${tarball_url}" -o "${tmp}/repo.tar.gz"
# Extract only the required paths, which is markedly faster and saves space
tar -xzf "${tmp}/repo.tar.gz" -C "${tmp}" \
  --wildcards \
  '*/docker-compose.yml' \
  '*/.env.example' \
  '*/config/config.yaml'
src=$(find "${tmp}" -maxdepth 1 -mindepth 1 -type d -name 'EnterpriseRag-*' | head -1)
if [[ -z "${src}" ]]; then
  echo "[prepare] extraction failed, no EnterpriseRag-* directory found" >&2
  exit 1
fi

cp    "${src}/docker-compose.yml" "${ENTERPRISERAG_DIR}/"
cp    "${src}/.env.example"       "${ENTERPRISERAG_DIR}/"
cp    "${src}/config/config.yaml" "${ENTERPRISERAG_DIR}/config/"

# Record metadata for firstboot / upgrade to refer to
cat >"${ENTERPRISERAG_DIR}/.cloud-image-meta" <<EOF
ENTERPRISERAG_REF=${ENTERPRISERAG_REF}
ENTERPRISERAG_REPO=${ENTERPRISERAG_REPO}
PREPARED_AT=$(date -Iseconds)
EOF

echo "[prepare] 3/6 preparing .env (defaults; firstboot replaces these with random secrets)"
cd "${ENTERPRISERAG_DIR}"
[[ -f .env ]] || cp .env.example .env
sed -i 's/^GIN_MODE=.*/GIN_MODE=release/' .env || true

# Align ENTERPRISERAG_VERSION with ENTERPRISERAG_REF so docker compose pulls the image tag matching the ref.
# Overwritten unconditionally, so a stale version left by a previous prepare run does not survive.
# Tag naming convention for ORG_PLACEHOLDER/enterpriserag-* on Docker Hub:
#   - floating tag: main (always points at the latest build)
#   - pinned release tag: v prefix + semver (e.g. v0.7.2, v0.5.2)
# So the v prefix is neither stripped nor mapped to latest here.
ENTERPRISERAG_VERSION_VAL="${ENTERPRISERAG_REF}"
if grep -qE '^ENTERPRISERAG_VERSION=' .env; then
  sed -i "s|^ENTERPRISERAG_VERSION=.*|ENTERPRISERAG_VERSION=${ENTERPRISERAG_VERSION_VAL}|" .env
else
  echo "ENTERPRISERAG_VERSION=${ENTERPRISERAG_VERSION_VAL}" >>.env
fi
echo "[prepare]   -> ENTERPRISERAG_VERSION=${ENTERPRISERAG_VERSION_VAL}"

echo "[prepare] 4/6 pulling and starting the 5 default resident containers (frontend/app/docreader/postgres/redis)"
docker compose pull
docker compose up -d

# Pull the sandbox image ahead of time (the Agent Skills runtime is started on demand by app
# via docker run, so it is not a resident container).
# Without this, the first Skill run stalls on the download.
echo "[prepare] 4.5/6 pre-pulling the sandbox image (used by Agent Skills, not resident)"
docker compose --profile full pull sandbox || true

# The other vector stores / observability components (qdrant, milvus, weaviate, neo4j,
# langfuse-*, minio, dex) are not pre-pulled, saving 5-15GB. To enable one:
#   cd /opt/EnterpriseRag && docker compose --profile <name> up -d

# Upgrade path: clean up old-tag ORG_PLACEHOLDER/enterpriserag-* images.
# Off by default to keep a rollback path; turn it on explicitly before baking an image.
#
# Note: do NOT use `docker image prune -af`!
# The sandbox image is only pulled by compose, never started (Agent Skills run it on demand
# from app), so no container references it, and `prune -a` would delete the current sandbox
# image too, defeating the purpose of the pre-pull in step 4.5.
# The comparison below is tag-exact: it removes only images in the ORG_PLACEHOLDER/enterpriserag-*
# repositories whose tag differs from the current ENTERPRISERAG_VERSION, leaving infrastructure
# images (paradedb / redis) untouched.
if [[ "${PRUNE_OLD_IMAGES,,}" == "true" || "${PRUNE_OLD_IMAGES}" == "1" ]]; then
  echo "[prepare] 4.6/6 removing old images under ORG_PLACEHOLDER/enterpriserag-* (PRUNE_OLD_IMAGES=true, keep=${ENTERPRISERAG_VERSION_VAL})"
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '^ORG_PLACEHOLDER/enterpriserag-' \
    | grep -vE ":${ENTERPRISERAG_VERSION_VAL}\$" \
    | xargs -r docker rmi -f 2>/dev/null || true
fi

echo "[prepare] 5/6 installing systemd units"
# Locate the docker binary; distributions place it in /usr/bin or /usr/local/bin
DOCKER_BIN="$(command -v docker)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "[prepare] docker binary not found" >&2
  exit 1
fi
echo "[prepare]   docker binary: ${DOCKER_BIN}"

install -m 0644 "${SCRIPT_DIR}/systemd/enterpriserag.service"           /etc/systemd/system/enterpriserag.service
install -m 0644 "${SCRIPT_DIR}/systemd/enterpriserag-firstboot.service" /etc/systemd/system/enterpriserag-firstboot.service
install -m 0755 "${SCRIPT_DIR}/firstboot.sh"                      /usr/local/sbin/enterpriserag-firstboot.sh

# Substitute the docker path template in the systemd units with the actual path
sed -i "s|@DOCKER_BIN@|${DOCKER_BIN}|g" /etc/systemd/system/enterpriserag.service

systemctl daemon-reload
systemctl enable enterpriserag.service
systemctl enable enterpriserag-firstboot.service

echo "[prepare] 6/6 done"
echo
echo "  The EnterpriseRag runtime has been deployed to ${ENTERPRISERAG_DIR}"
echo "    docker-compose.yml / config/config.yaml / .env"
echo "  Version: ${ENTERPRISERAG_REF}  (see ${ENTERPRISERAG_DIR}/.cloud-image-meta)"
echo
echo "  Open a browser at  http://<this host's public IP>  to verify"
echo
echo "  Once verified, run the cleanup and bake the image:"
echo "      sudo bash ${SCRIPT_DIR}/cleanup.sh"
