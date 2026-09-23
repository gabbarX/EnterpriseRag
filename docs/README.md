# docs/

This directory holds only engineering resources, not product documentation. The
product overview, installation guide and feature reference live in the top-level
[`README.md`](../README.md).

Contents:

- `docs.go`, `swagger.json`, `swagger.yaml` and the contract test — these take part in
  the backend build and test run; the generated artefacts are refreshed by `make docs`.
- `LITE.md` — copied into the Lite release bundle as its offline README.
- `images/`, `assets/` — image assets still referenced by the README, Helm charts and so on.
- `poc/docker-sandbox/` — a standalone Go experiment module; its source and run notes are
  kept for the record and are not a current product guide.

These have build, release or historical value, so the directory cannot simply be deleted.
Do not add new product documentation here.
