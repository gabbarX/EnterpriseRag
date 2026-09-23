# Docker Sandbox Backend Feasibility PoC

This PoC only preserves a historical feasibility experiment; it does not describe current adapter behaviour.

The program talks directly to the Docker Engine API and checks, item by item, whether Docker can
carry the EnterpriseRag `RemoteSandboxClient` contract plus the E2B snapshot workflow. It is not product
code and takes no part in building the main module (it carries its own `go.mod`); it exists only as
reproducible evidence for the research conclusions.

Every item prints `PASS/FAIL` along with the value actually observed. Steps marked `(GAP)` assert
what Docker *cannot* do, and they also end with PASS — a PASS there means the gap was reproduced,
not that the capability is present.

## How to Run It

You need a reachable Docker daemon (`DOCKER_HOST` is honoured) that can pull `python:3.11-slim`:

```bash
cd docs/poc/docker-sandbox
go run .            # may need sudo -E when the daemon listens on a local unix socket
```

The program builds its own template image containing a `user` account with uid 1000 (matching the
E2B template convention) and removes the containers it created when it finishes. Clean up the
`enterpriserag-poc/*` images it leaves behind with `docker image rm`.

## What Is Covered

- Lifecycle: Create / Connect (reconnecting with a fresh client) / List (filtering by label) / Delete / Pause / Stop+Start
- Execution: user, workdir, env, stdin, separated stdout and stderr, exit codes, timeouts
- File surface: reading, writing and stat through the archive API; mkdir/ls/rm implemented via exec
- Session semantics: `pip install` results and written files surviving across exec calls
- Snapshots: committing to an image, starting a new sandbox from a snapshot, v1→v2 increments, listing by label, layer count and size
- Gap reproduction: client-side cancellation not killing the process, root not getting past permission bits once CapDrop ALL is applied, snapshots not capturing memory state, the daemon having no idle TTL, and killing `docker run` leaving a container still running
