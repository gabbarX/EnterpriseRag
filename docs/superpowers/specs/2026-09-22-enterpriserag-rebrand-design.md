# EnterpriseRag — de-Sinicisation & white-label rebrand

**Date:** 2026-09-22
**Status:** Awaiting review
**Scope:** Whole repository (~2,300 of 4,275 tracked files)

---

## 1. Context

This repository is `github.com/Tencent/WeKnora`, a Tencent-authored open-source RAG
platform. The owner has acquired rights and approval from Tencent to remove Tencent
branding and white-label the product. The goal is a product that presents as
Indian-built, ships English-only, and carries no China-origin vendor integrations,
configuration variables, or platform couplings.

This is **not** a string-replacement exercise. China-origin coverage in this codebase
is first-class: 15 model vendors with their own catalogs and clients, 5 object-storage
backends, 6 IM channels, 4 web-search providers, 4 data-source connectors, a vector-store
driver, and the code-execution sandbox base image. Roughly 46,000 lines across 947
files contain CJK characters. Removing all of it is a feature-removal project.

### Licensing boundary

The commercial agreement covers Tencent's own copyright. It cannot relicense
**third-party** code that Tencent redistributes. The following are retained regardless:

- `THIRD_PARTY_NOTICES.md` and `licenses/` — NumPy (BSD), PaddlePaddle (Apache-2.0),
  gRPC, Microsoft, Google, Wails (MIT), go-m1cpu (MPL-2.0), go-sql-driver/mysql (MPL-2.0)
- `licenses/OpenCC-Apache-2.0.txt` becomes removable **only** once `internal/textconv/`
  is deleted in Phase 1, since that is the code the licence covers.

Root `LICENSE` is rewritten to the new owner copyright with the retained third-party
block preserved. Git history is **not** rewritten.

---

## 2. Decisions

| Area | Decision |
|---|---|
| i18n | **Full rip-out.** Inline English strings into components; delete vue-i18n, all 5 locale bundles, the embed i18n bundle, and the audit tooling. Backend language machinery removed too. |
| Module path | `github.com/ORG_PLACEHOLDER/EnterpriseRag` |
| Naming | "EnterpriseRag everywhere": binary `enterpriserag`, containers `EnterpriseRag-*`, images `ORG_PLACEHOLDER/enterpriserag-*`, env prefix `ENTERPRISERAG_*`, DB `enterpriserag` |
| Breaking changes | **Hard rename, no compatibility shims.** Env vars, localStorage keys, widget global, container names all change outright. |
| CJK processing code | Delete **all** CJK-specific code paths, including chunker branches. |
| Docs & comments | Translate contracts (Swagger, `.env.example`, error/log strings). Delete Chinese internal comments rather than translate. |
| Docs site | **Delete `website-docs/` entirely** (178 files, incl. the Next.js homepage and all Chinese vendor brand assets). |
| Screenshots | **Regenerate now** by running the rebranded English stack. |
| Model catalog | Keep **5 of 27** vendor packages: `openai`, `anthropic`, `gemini`, `openrouter`, `generic`. Delete 22 — the 15 China vendors (incl. `weknoracloud`) **and** `azure_openai`, `jina`, `litellm`, `gpustack`, `novita`, `nvidia`, `requesty`. Note: `ollama` is **not** a vendor package (it lives in `internal/models/chat/ollama.go` and `internal/infrastructure/web_search/ollama.go`) and survives automatically. |
| Storage | `s3`, `minio`, `local`, `dummy`. Default `minio`. Docs reference AWS `ap-south-1`. |
| Vector store | Drop `tencentvectordb` and `doris`. Keep paradedb, qdrant, milvus, weaviate, elasticsearch, opensearch. |
| IM | `slack`, `telegram`, `mattermost` only. |
| Search / connectors | `duckduckgo`, `searxng`, `serply`. All connectors (`ima`, `feishu`, `dingtalk`, `yuque`) deleted. |
| Sandbox | Tencent CubeSandbox removed now; interface preserved behind `SANDBOX_ENABLED=false`. Generic runtime rebuild is a **separate follow-up spec**. |
| India fit | Defaults and docs only — no new vendor integrations. |
| Identity strings | Placeholders now (see §6), substituted later by the owner. |
| Delivery | One branch, 8 phased commits, **suite green at every commit**. |

### Deliberately NOT removed

- `internal/utils/security.go:223` — `metadata.tencentyun.com` stays in the SSRF
  blocklist alongside the AWS and GCP metadata hosts. It is a defensive control.
- Generic multibyte/rune-width handling is distinguished from CJK-specific logic
  wherever the two can be separated (see Risk R1).

---

## 3. Phased plan

Branch: `chore/enterpriserag-rebrand`

### Phase 1 — Remove China-origin integrations

Largest and riskiest phase; lands first so everything after it operates on a smaller tree.

- `internal/models/vendors/` — delete 22 of 27 packages and their `all.go` registrations.
  China-origin (15): `hunyuan`, `lkeap`, `aliyun`, `qianfan`, `volcengine`, `zhipu`,
  `moonshot`, `deepseek`, `siliconflow`, `minimax`, `modelscope`, `qiniu`, `longcat`,
  `mimo`, `weknoracloud`. Also dropped by owner decision (7): `azure_openai`, `jina`,
  `litellm`, `gpustack`, `novita`, `nvidia`, `requesty`.
  **Survivors (5):** `openai`, `anthropic`, `gemini`, `openrouter`, `generic`.
  Update `internal/models/vendors/vendors_test.go` to match the new catalog.
- Delete bespoke clients: `internal/models/api/dashscopeembeddings/`, `dashscoperank/`,
  `arkembeddings/`, `internal/models/rerank/volcengine_reranker.go`
- `internal/application/service/file/` — delete `cos.go`, `oss.go`, `tos.go`, `obs.go`,
  `ks3.go` and their tests; keep `s3.go`, `minio.go`, `local.go`, `dummy.go`
- Delete `internal/application/repository/retriever/tencentvectordb/` and `retriever/doris/`;
  remove the driver constants from `internal/types/vectorstore.go:718,916`
- Delete `internal/im/{wechat,wecom,qqbot,feishu,dingtalk,yunzhijia}/` and the channel
  constants in `internal/types/knowledge.go:27-33` and `internal/im/types.go:99`
- Delete `internal/infrastructure/web_search/{baidu,zhipu,bocha,metaso}.go`
- Delete `internal/datasource/connector/{ima,feishu,dingtalk,yuque}/` and the
  `ConnectorType*` constants in `internal/types/datasource.go`
- Delete `internal/textconv/` (OpenCC dictionaries) and `licenses/OpenCC-Apache-2.0.txt`
- `chat_pipeline/query_expansion.go:160-191` — remove the Chinese stop-word set and the
  Chinese `questionWords` regex
- **Sandbox:** delete the `ghcr.io/tencentcloud/cubesandbox-base:2026.16` stage from
  `docker/Dockerfile.sandbox:115`, the CubeSandbox SDK, and
  `internal/sandbox/cube_egress_integration_test.go`. Keep the exported interface of
  `internal/sandbox/`; return "sandbox unavailable" behind `SANDBOX_ENABLED=false`.
- `go.mod` — drop the 16 China-vendor modules; `go mod tidy`
- Remove matching frontend vendor icons and the dead `STORAGE_TYPE` / `RETRIEVE_DRIVER`
  enum values

### Phase 2 — Remove China-platform directories

- `miniprogram/` and `tests/miniprogram/` (incl. the hardcoded WeChat AppID
  `wxbaf7dc10724effab` at `miniprogram/project.config.json:32`)
- `packages/dsh-weknora/` (npm scope `@wxg-prc-cpg` = Tencent WeChat Group), and with it
  `.github/workflows/dsh-plugin.yml`, which exists only to build that package
- `website-docs/` in full — 178 files, the VitePress site and the Next.js homepage,
  including `public/brands/` (Tencent Cloud favicon, WeChat Dialog icon, hunyuan/qwen/
  deepseek/wecom/feishu/ima/yuque logos) and `app/brands.tsx`
- `README_CN.md`, `README_JA.md`, `README_KO.md`, `dataset/README_zh.md`;
  remove the language cross-link row at `README.md:39`
- Mirrors: `.env.example:35` `APK_MIRROR_ARG=mirrors.tencent.com` → upstream default;
  clear the `npmmirror` suggestions in `frontend/Dockerfile:14-15,39-42` and
  `docker-compose.yml`; strip the Tencent/Aliyun mirror defaults from
  `scripts/cloud-image/prepare.sh:13-21,32-35,87-103`; change `GOSUMDB_ARG=off` to
  restore checksum verification in `docker/Dockerfile.app:22-35`

### Phase 3 — Rip out i18n

Frontend:
- Inline English strings from `en-US.ts` (7,424 lines — already a complete translation,
  so this is mechanical, not a translation task) into the 208 `.vue` and 385 `.ts` files
- Delete `frontend/src/i18n/` entirely: all 5 locale bundles, `embed.ts`,
  `localeKeyAudit.ts`, `localeGapScan.ts`, `regeneratePrunedLocales.ts`,
  `localeSerialize.ts`, `resolveDefaultLocale.ts`, `auditAction*`, and their tests
- Remove `vue-i18n` and `@intlify/core-base` from `package.json`; drop the `check-i18n`,
  `scan-i18n-gaps` and `regenerate-i18n-locales` scripts
- Delete the language switchers: `views/settings/GeneralSettings.vue:12-26,225-238`,
  `views/auth/Login.vue:438,524-525`, `components/AgentEmbedChannelPanel.vue:538`
- Remove `Accept-Language` stamping from `utils/request.ts:68`, `api/chat/streame.ts:171`,
  `components/SkillInstallTimeline.vue:228`, `composables/useConfigSkillInstallProgress.ts:93`
- Delete `stores/localizedResourceCache.ts`; drop the locale watchers in `stores/menu.ts:62`
  and `stores/modelProviders.ts:29`
- Resolve the 53 `t(key, <chinese>)` fallbacks (concentrated in `views/settings/`) to
  plain English

Backend:
- Delete `internal/middleware/language.go`, the `WEKNORA_LANGUAGE` env var, and
  `LanguageLocaleName` / `ResolveLanguage` in `internal/types/context_helpers.go:353-428`
- Delete `PromptTemplateI18n` (`internal/config/config.go:331-430`)
- Replace every `{{language}}` with the literal `English`; remove the placeholder
  registration at `internal/types/placeholder.go:98`
- Drop the `zh-CN` / `zh-TW` / `ja-JP` / `ko-KR` blocks from `config/builtin_agents.yaml`,
  `config/agent_type_presets.yaml` and `config/prompt_templates/*.yaml`
- Remove `DEFAULT_LOCALE` and `VITE_DEFAULT_LOCALE`

### Phase 4 — Translate contracts, delete Chinese comments

- **1,671 Swagger annotation lines** across `internal/handler/*.go` → English;
  regenerate `docs/swagger.{json,yaml}` and `docs/docs.go` via `make docs`
- `.env.example` — 362 Chinese comment lines → English
- Hardcoded Chinese user-facing strings, none routed through i18n:
  `internal/agent/act.go:165-180` (agent tool display names),
  `internal/types/sandbox_network_policy.go:178-211` (validation errors),
  `internal/application/service/knowledge_faq_import.go`,
  `internal/handler/initialization.go`, `internal/handler/knowledge.go`
- Chinese LLM prompts → English: `internal/handler/session/image_upload.go:100-106`
  (currently hard-forces Chinese model output regardless of locale),
  `internal/application/service/memory/topic_resolve.go:164-189`,
  `internal/application/service/memory/consolidate.go:453+`
- `config/prompt_templates/rewrite.yaml:37-93` — replace the Chinese few-shot and
  negative examples
- `scripts/*.sh` echo/log messages (637 lines), `Makefile` help text,
  `docker-compose*.yml` section headers
- **Delete** the ~3,600 Chinese internal code-comment lines rather than translate them

### Phase 5 — Module path rename

Single mechanical commit, ~1,900 files:
- `go.mod`, `cli/go.mod`, `client/go.mod`: `github.com/Tencent/WeKnora` →
  `github.com/ORG_PLACEHOLDER/EnterpriseRag`
- Rewrite every import; update the `Makefile` ldflags
  (`-X '<module>/internal/handler.Version=...'`) and `scripts/get_version.sh`

### Phase 6 — Brand identifiers

- `frontend/index.html` and `embed.html`: `<title>`, meta description/keywords, favicon
- ~20 `localStorage` keys: `weknora_token` → `enterpriserag_token`, `WeKnora_theme`,
  `WeKnora_settings`, `weknora_tenant`, `weknora_current_kb`, etc.
- Widget global: `window.WeKnora` → `window.EnterpriseRag`
  (`frontend/public/weknora-widget.js`, file renamed)
- ~60 `WEKNORA_*` env vars → `ENTERPRISERAG_*` across Go, shell scripts, compose,
  helm and the CLI
- Docker: images `ORG_PLACEHOLDER/enterpriserag-{app,ui,docreader}`, containers
  `EnterpriseRag-*` (`docker-compose*.yml`, `scripts/build_images.sh:132-282,361-362`,
  `helm/values.yaml`, `Makefile`)
- `DB_NAME`, `ELASTICSEARCH_INDEX`, `OPENSEARCH_INDEX`, `LANGFUSE_INIT_*`
- 5 User-Agent strings, incl. `internal/infrastructure/web_search/serply.go`
  (**`serply_test.go:25` asserts this — update in the same commit**)
- `cmd/desktop/wails.json` (name, author, `productName`, copyright),
  `Formula/weknora-lite.rb`, `deploy/weknora-lite.service`, CLI `Use: "weknora"`
  at `cli/cmd/root.go:137`, `mcp-server/setup.py` + `pyproject.toml`
  (`tencent-weknora-mcp` → `PYPI_NAME_PLACEHOLDER`, `support@weknora.com` →
  `EMAIL_PLACEHOLDER`)
- Fix the stale viper search paths `$HOME/.appname` and `/etc/appname/`
  (`internal/config/config.go`, ~line 500)
- **`.github/`** — rebrand the 8 surviving files that reference the old identity:
  `workflows/anydoc.yml`, `workflows/cli-e2e.yml`, `workflows/release-lite.yml`,
  `dependabot.yml`, `ISSUE_TEMPLATE/{bug_report,config,feature_request,question}.yml`
  (module path, image names, artifact names, repo links)
- Rewrite root `LICENSE` per §1

### Phase 7 — India defaults and demo data

- `TZ=Asia/Kolkata` in compose and docs
- `config/config.yaml`: `split_markers` drops the Chinese full stop, adds the
  Devanagari danda — `["\n\n", "\n", "।"]`
- Storage docs reference AWS `ap-south-1` (Mumbai)
- Indian English spelling; `₹`/INR in sample data
- Replace demo corpora with India-context English (HR leave-and-expenses policy,
  GST FAQ, product manual) in
  `frontend/src/views/knowledge/settings/chunkingSamples.ts` and `testdata/wiki_test/`
- Convert the 2,564 CJK lines across 290 test files to English. Where a test genuinely
  exercises multibyte handling, substitute **Devanagari** so coverage survives and is
  relevant to the market (see Risk R1)
- Delete `frontend/src/views/dev/MarkdownTestPage.vue` (dev-only, 48 CJK lines)

### Phase 8 — Screenshots and final sweep

- `make start-all`, drive the English UI, re-capture the ~40 screenshots under
  `docs/images/` (`qa.png`, `settings.png`, `knowledgebases.png`, `rbac-*.png`,
  `mcp-configuration/*`, `browser-*.png`, `chat-steer-*.png`); delete
  `docs/assets/milvus-bm25-chinese-fixed.png`
- Rewrite `README.md` in English with new badges; remove the `weknora.weixin.qq.com`
  (16 refs) and `chatbot.weixin.qq.com` (8 refs) links
- Final sweep: `rg -i 'weknora|tencent|wechat|weixin|qq\.com'` plus a CJK range scan
  must return only the retained third-party licence notices and the SSRF blocklist entry

---

## 4. Verification

Gate for **every** commit:

```bash
make fmt && make lint && make test
bash scripts/verify_frontend_pr.sh     # npm test -> type-check -> build
cd docreader && pytest
cd mcp-server && pytest
```

End-to-end, after Phase 8:

```bash
make build-images && make start-all && make check-env
```

Then manually: create a knowledge base, upload an English PDF and a Hindi document,
run a chat query, confirm no Chinese renders anywhere and no China-origin host is
contacted (check egress logs).

**CI:** `.github/` had been deleted in the working tree before this work began; it has been
restored from HEAD via `git checkout -- .github`. It holds 12 workflows — `app.yml`,
`frontend.yml`, `go-lint.yml`, `go-lint-cache.yml`, `docreader.yml`, `cli.yml`,
`cli-e2e.yml`, `docker-image.yml`, `anydoc.yml`, `mcp-server.yml`, `release-lite.yml`,
`dsh-plugin.yml` — plus `dependabot.yml`, issue templates and a PR template. These are the
automated gate for every phase; also install the local hooks via
`scripts/install-git-hooks.sh`.

Nine `.github/` files reference the old brand and are rebranded in Phase 6. The exception is
`dsh-plugin.yml`, which builds `packages/dsh-weknora/` — since Phase 2 deletes that package,
the workflow is **deleted in Phase 2** rather than rebranded.

---

## 5. Risks

| # | Risk | Mitigation |
|---|---|---|
| R1 | Deleting *all* CJK code paths may degrade tokenisation for other non-ASCII scripts. Devanagari, Tamil and Bengali are multibyte too, and the chunker CJK branches often share generic rune-width logic. | The Phase 7 Devanagari fixtures are the guard: they must pass after the Phase 1 deletions. If they fail, the deleted logic was generic rather than CJK-specific and gets restored. |
| R2 | Phase 3 inlines ~380 KB of strings across 208 components by hand — the largest source of silent regressions. | `vue-tsc --build` plus the 274 existing frontend tests; the locale audit tooling is deleted only *after* inlining is verified. |
| R3 | Hard rename with no shims logs out every existing user, invalidates all `.env` files, and breaks any customer site embedding the widget. | Accepted — treated as a fresh deployment. Must be stated in the release notes. |
| R4 | Sandbox ships disabled, so any skill that executes code is unavailable. | Explicit `SANDBOX_ENABLED=false` default and a documented follow-up spec. |
| R5 | The Phase 5 rename conflicts with any other in-flight branch. | Land it as an isolated commit; freeze other work across it. |
| R6 | Placeholders leave the tree unbuildable until substituted. | §6 checklist; a `rg PLACEHOLDER` sweep is part of the Phase 8 gate. |

---

## 6. Placeholder substitution checklist

The owner must supply and substitute these before the build will pass:

| Token | Appears in |
|---|---|
| `ORG_PLACEHOLDER` | `go.mod`, `cli/go.mod`, `client/go.mod`, every Go import, `Makefile` ldflags, Docker image names, `helm/values.yaml`, `Formula/`, README badges |
| `DOMAIN_PLACEHOLDER` | `helm/values.yaml` (`host:`), README, `mcp-server` metadata, `frontend_base_url` |
| `EMAIL_PLACEHOLDER` | `mcp-server/setup.py`, `mcp-server/pyproject.toml`, `SECURITY.md` |
| `NPM_SCOPE_PLACEHOLDER` | any republished frontend package |
| `PYPI_NAME_PLACEHOLDER` | `mcp-server/setup.py`, `mcp-server/pyproject.toml` |

### ⚠️ Substituting `ORG_PLACEHOLDER` requires regenerating the protobuf stubs

**Do not substitute this token with a plain find-and-replace.** `docreader/proto/docreader.pb.go` and
`docreader/proto/docreader_pb2.py` embed a serialised `FileDescriptorProto` in which the `go_package`
path is **length-delimited**. A textual substitution changes the path's byte length without updating the
varint length prefixes, which corrupts the descriptor. The failure is not a compile error — it is a
panic at process start:

```
panic: runtime error: slice bounds out of range [-4:]
  google.golang.org/protobuf/internal/filedesc.(*File).unmarshalSeed
  .../docreader/proto.file_docreader_proto_init()
```

During Task 5 this took down 15 packages and was repaired by hand-patching two varints. **The correct
procedure when substituting the real org name is to regenerate instead:**

**Use the same generator versions the committed files were built with**, or the diff will be large and
unreviewable. The header of `docreader/proto/docreader.pb.go` records them:
`protoc-gen-go v1.36.7`, `protoc v5.29.3`. Debian bookworm's `protobuf-compiler` package is
`libprotoc 3.21.12` — several major versions older — so install `protoc` v5.29.3 from the
`protocolbuffers/protobuf` releases rather than from apt.

```bash
# inside the dev container, with protoc v5.29.3 and protoc-gen-go v1.36.7 on PATH
protoc -I docreader/proto \
  --go_out=paths=source_relative:docreader/proto \
  --go-grpc_out=paths=source_relative:docreader/proto \
  docreader/proto/docreader.proto

python -m grpc_tools.protoc -I docreader/proto \
  --python_out=docreader/proto --pyi_out=docreader/proto \
  --grpc_python_out=docreader/proto \
  docreader/proto/docreader.proto
```

Then run `go build ./...` and `cd docreader && PYTHONPATH=/src python -m pytest -q` to confirm the
descriptor loads. `docreader/proto/docreader.proto:5` is the single source of truth for `go_package`.

Note also that `go_package` points at `.../internal/docreader/proto` while the generated files live at
`docreader/proto/` — a pre-existing upstream inconsistency, left as-is.

---

## 7. Out of scope

- Rebuilding the sandbox on gVisor or rootless Docker — **separate follow-up spec**,
  with its own threat model
- A replacement documentation or marketing site
- New Indic model-vendor integrations (Sarvam AI, Krutrim)
- WhatsApp Business as an IM channel
- Any rewrite of git history
