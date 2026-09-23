# EnterpriseRag Rebrand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strip all China-origin integrations, branding and language support from the WeKnora fork, producing an English-only, white-labelled product called EnterpriseRag.

**Architecture:** Eight sequential phases on one branch, each a single commit that leaves the full test suite green. Deletion-heavy phases land first (smaller tree for everything after), the mechanical module rename lands late (it conflicts with everything), and verification-by-running-the-app lands last.

**Tech Stack:** Go 1.26 + Gin (3 modules: root, `cli/`, `client/`), Vue 3 + Vite + TypeScript + Pinia + TDesign, Python (`docreader/` gRPC, `mcp-server/`), Docker Compose, Helm.

**Spec:** `docs/superpowers/specs/2026-09-22-enterpriserag-rebrand-design.md`

## Global Constraints

- **Branch:** `chore/enterpriserag-rebrand`. One commit per phase, in order.
- **All work runs inside a Linux container.** The host is Windows and lacks `make`, a C
  compiler, `golangci-lint` and `swag`. Three dependencies are CGO-only
  (`pg_query_go/v6`, `sqlite-vec-go-bindings`, `duckdb-go-bindings`), and `internal/utils`
  imports `pg_query`, so **`go build ./...` fails on a pristine tree on the host**. Host-side
  Go is not a substitute. Build the image once:

  ```bash
  docker build -t enterpriserag-dev -f .superpowers/sdd/Dockerfile.dev .
  docker run --rm -it -v "$PWD":/src -w /src \
    -v enterpriserag-gomod:/go/pkg/mod enterpriserag-dev bash
  ```

  Git operations happen on the host; builds and tests happen in the container.
- **Green gate — every commit must pass, no exceptions, run inside the container:**
  ```bash
  make fmt && make lint && make test
  bash scripts/verify_frontend_pr.sh
  cd docreader && pytest && cd ..
  cd mcp-server && pytest && cd ..
  ```
  The container also resolves the 5 pre-existing frontend failures seen on the host, which
  are CRLF regex mismatches and a missing `/bin/sh`, not code defects.
- **Module path:** `github.com/ORG_PLACEHOLDER/EnterpriseRag` (literal token, substituted later by the owner).
- **Placeholders, used verbatim:** `ORG_PLACEHOLDER`, `DOMAIN_PLACEHOLDER`, `EMAIL_PLACEHOLDER`, `NPM_SCOPE_PLACEHOLDER`, `PYPI_NAME_PLACEHOLDER`.
- **Naming:** product `EnterpriseRag`; binary `enterpriserag`; containers `EnterpriseRag-*`; images `ORG_PLACEHOLDER/enterpriserag-{app,ui,docreader}`; env prefix `ENTERPRISERAG_*`; DB `enterpriserag`.
- **No compatibility shims.** Old env vars, storage keys and globals are removed outright.
- **Never delete:** `THIRD_PARTY_NOTICES.md`, `licenses/` (except `OpenCC-Apache-2.0.txt` in Task 1), and `metadata.tencentyun.com` at `internal/utils/security.go:223`.
- **Never rewrite git history.**
- **Deleting a feature deletes its tests in the same commit.** A test asserting Chinese output is rewritten to assert English, not deleted.
- **Language:** all new strings, comments and docs in Indian English.

---

### Task 1: Remove China-origin integrations

**Files:**
- Delete: 22 dirs under `internal/models/vendors/`; `internal/models/api/{dashscopeembeddings,dashscoperank,arkembeddings}/`; `internal/models/rerank/volcengine_reranker.go`; `internal/application/service/file/{cos,oss,tos,obs,ks3}*.go`; `internal/application/repository/retriever/{tencentvectordb,doris}/`; `internal/im/{wechat,wecom,qqbot,feishu,dingtalk,yunzhijia}/`; `internal/infrastructure/web_search/{baidu,zhipu,bocha,metaso}.go`; `internal/datasource/connector/{ima,feishu,dingtalk,yuque}/`; `internal/textconv/`; `licenses/OpenCC-Apache-2.0.txt`; `internal/sandbox/cube_egress_integration_test.go`
- Modify: `internal/models/vendors/all.go`, `internal/models/vendors/vendors_test.go`, `internal/types/vectorstore.go:718,916`, `internal/types/knowledge.go:27-33`, `internal/im/types.go:99`, `internal/types/datasource.go`, `internal/application/service/chat_pipeline/query_expansion.go:160-191`, `internal/application/service/file/factory.go`, `docker/Dockerfile.sandbox:115`, `go.mod`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: a vendor catalog of exactly `openai`, `anthropic`, `gemini`, `openrouter`, `generic`. A storage factory accepting only `local`, `minio`, `s3`, `dummy`. A sandbox package whose exported interface is unchanged but always returns `ErrSandboxUnavailable`.

- [ ] **Step 1: Create the branch**

```bash
git checkout -b chore/enterpriserag-rebrand
git status --short   # expect only the untracked agent files + docs/superpowers/
```

- [ ] **Step 2: Record the pre-change test baseline**

Run the full green gate *before* touching anything and save the output. If something is already failing on a fresh clone, you must know now — otherwise you will blame your own change for it later.

```bash
make test 2>&1 | tail -30 > /tmp/baseline-go.txt
bash scripts/verify_frontend_pr.sh 2>&1 | tail -30 > /tmp/baseline-fe.txt
cat /tmp/baseline-go.txt /tmp/baseline-fe.txt
```

Expected: both green. If not, list the pre-existing failures and report them before continuing.

- [ ] **Step 3: Delete the 22 vendor packages**

```bash
cd internal/models/vendors
rm -rf hunyuan lkeap aliyun qianfan volcengine zhipu moonshot deepseek \
       siliconflow minimax modelscope qiniu longcat mimo weknoracloud \
       azure_openai jina litellm gpustack novita nvidia requesty
ls   # expect: all.go anthropic gemini generic openai openrouter vendors_test.go
cd -
```

- [ ] **Step 4: Rewrite the vendor registry**

Replace the import block in `internal/models/vendors/all.go` with exactly these five lines (keep the existing package comment and `package vendors` declaration):

```go
import (
	// Each vendor registers itself with the catalog in init.
	_ "github.com/Tencent/WeKnora/internal/models/vendors/anthropic"
	_ "github.com/Tencent/WeKnora/internal/models/vendors/gemini"
	_ "github.com/Tencent/WeKnora/internal/models/vendors/generic"
	_ "github.com/Tencent/WeKnora/internal/models/vendors/openai"
	_ "github.com/Tencent/WeKnora/internal/models/vendors/openrouter"
)
```

Note: the module path is still `github.com/Tencent/WeKnora` at this point. Task 5 renames it everywhere at once — do not rename it here.

- [ ] **Step 5: Delete the bespoke China-vendor clients and rerankers**

```bash
rm -rf internal/models/api/dashscopeembeddings \
       internal/models/api/dashscoperank \
       internal/models/api/arkembeddings
rm -f  internal/models/rerank/volcengine_reranker.go
```

- [ ] **Step 6: Delete the China object-storage backends**

```bash
cd internal/application/service/file
rm -f cos.go cos_parse_test.go oss.go oss_test.go tos.go obs.go obs_test.go \
      ks3.go ks3_path_test.go ks3_redirect_test.go
ls   # survivors must include: local.go minio.go s3.go dummy.go factory.go
cd -
```

- [ ] **Step 7: Delete the vector stores, IM channels, web search and connectors**

```bash
rm -rf internal/application/repository/retriever/tencentvectordb \
       internal/application/repository/retriever/doris
rm -rf internal/im/wechat internal/im/wecom internal/im/qqbot \
       internal/im/feishu internal/im/dingtalk internal/im/yunzhijia
rm -f  internal/infrastructure/web_search/baidu.go \
       internal/infrastructure/web_search/zhipu.go \
       internal/infrastructure/web_search/bocha.go \
       internal/infrastructure/web_search/metaso.go
rm -rf internal/datasource/connector/ima internal/datasource/connector/feishu \
       internal/datasource/connector/dingtalk internal/datasource/connector/yuque
rm -rf internal/textconv
rm -f  licenses/OpenCC-Apache-2.0.txt
```

`internal/textconv` has exactly **one** caller: `internal/types/faq.go:740` calls
`textconv.ToSimplified`. That call converts traditional Chinese to simplified for FAQ
matching — delete the call and its surrounding normalisation branch, do not try to
substitute a replacement.

- [ ] **Step 8: Stub the sandbox**

Delete the Tencent base-image stage at `docker/Dockerfile.sandbox:115` (`FROM --platform=linux/amd64 ghcr.io/tencentcloud/cubesandbox-base:2026.16 AS cube`) and every `COPY --from=cube` that depends on it. Delete `internal/sandbox/cube_egress_integration_test.go`.

**This is larger than "keep the interface" suggests.** `github.com/tencentcloud/CubeSandbox/sdk/go`
is a **direct** dependency in `go.mod`, and the `cube_*.go` clients in `internal/sandbox/` are
built on its types. Deleting those clients removes the implementations behind exported symbols,
so the stub must **reimplement** each exported symbol with a native signature — replacing any
CubeSandbox type that appears in an exported signature with a local struct — rather than merely
guarding the existing ones. Audit the package's exported surface first (`go doc ./internal/sandbox`)
and keep that list identical.

Add to the package:

```go
// ErrSandboxUnavailable is returned by every sandbox operation while no
// execution runtime is configured. The CubeSandbox runtime was removed; a
// replacement is tracked as a separate follow-up spec.
var ErrSandboxUnavailable = errors.New("sandbox: no execution runtime configured")
```

Make each exported constructor/method return `ErrSandboxUnavailable` when `SANDBOX_ENABLED` is not `true`, and default `SANDBOX_ENABLED` to `false`.

- [ ] **Step 9: Strip the Chinese query-expansion logic**

In `internal/application/service/chat_pipeline/query_expansion.go:160-191`, delete the Chinese stop-word set and the `questionWords` regex alternation `^(什么是|如何|怎么|为什么|…)`. Leave the English equivalents and the surrounding function signature untouched.

- [ ] **Step 10: Let the compiler find every dangling reference**

This is the real work of the task. Deleting packages breaks the DI container, the storage factory, the retriever registry and the IM dispatcher. Work the error list to zero.

```bash
go build ./... 2>&1 | head -60
```

Fix each error by removing the dead branch — registry entries, `switch` cases, factory arms, config struct fields, enum constants. Expected hot spots: `internal/container/container.go`, `internal/application/service/file/factory.go`, `internal/types/vectorstore.go:718,916`, `internal/types/knowledge.go:27-33`, `internal/im/types.go:99`, `internal/types/datasource.go`, `internal/handler/initialization.go`. Re-run until the build is clean.

- [ ] **Step 11: Tidy the Go dependency graph**

```bash
go mod tidy && cd cli && go mod tidy && cd ../client && go mod tidy && cd ..
grep -iE "tencent|aliyun|volcengine|larksuite|dingtalk|ks3sdklib|milvus-io" go.mod || echo "CLEAN"
```

Expected: `CLEAN`, except `github.com/milvus-io/milvus/client/v2`, which stays (Milvus is a retained vector store).

- [ ] **Step 12: Update the vendor catalog test**

`internal/models/vendors/vendors_test.go` asserts the registered catalog. Update its expected set to exactly `{anthropic, gemini, generic, openai, openrouter}`.

```bash
go test ./internal/models/... -run TestVendor -v
```

Expected: PASS.

- [ ] **Step 13: Run the full green gate**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
```

Expected: all PASS. Frontend is untouched by this task, so `verify_frontend_pr.sh` must match the Step 2 baseline exactly.

- [ ] **Step 14: Commit**

```bash
git add -A
git commit -m "refactor: remove China-origin model, storage, IM, search and sandbox integrations"
```

---

### Task 2: Remove China-platform directories and package mirrors

**Files:**
- Delete: `miniprogram/`, `tests/miniprogram/`, `packages/dsh-weknora/`, `.github/workflows/dsh-plugin.yml`, `website-docs/`, `README_CN.md`, `README_JA.md`, `README_KO.md`, `dataset/README_zh.md`
- Modify: `README.md:39`, `.env.example:35-45`, `frontend/Dockerfile:14-15,39-42`, `docker-compose.yml:5,42,357,408`, `docker/Dockerfile.app:22-35`, `docker/Dockerfile.docreader:6-9,84-87`, `scripts/cloud-image/prepare.sh:13-21,32-35,87-103`

**Interfaces:**
- Consumes: Task 1's tree.
- Produces: a repo with no WeChat/Tencent platform couplings and no Chinese package mirrors. No new symbols.

- [ ] **Step 1: Delete the China-platform directories**

```bash
rm -rf miniprogram tests/miniprogram packages/dsh-weknora .github/workflows/dsh-plugin.yml
rm -rf website-docs
rm -f README_CN.md README_JA.md README_KO.md dataset/README_zh.md
```

`miniprogram/project.config.json:32` carried a hardcoded WeChat AppID (`wxbaf7dc10724effab`); it goes with the directory.

- [ ] **Step 2: Remove the README language cross-links**

Delete the `English | 简体中文 | 日本語 | 한국어` row at `README.md:39` and any links to the deleted README variants.

- [ ] **Step 3: Revert the package mirrors to upstream defaults**

In `.env.example`:

```diff
-APK_MIRROR_ARG=mirrors.tencent.com
+APK_MIRROR_ARG=
-# 国内可设 mirrors.tencent.com 加速
+# Optional Alpine APK mirror host. Leave empty to use upstream defaults.
```

Do the same for `APT_MIRROR` and `NPM_REGISTRY` — keep the build args (they are legitimately useful) but empty the defaults and rewrite their comments in English with no China-specific suggestions. Remove the `registry.npmmirror.com` suggestion from `frontend/Dockerfile:14-15,39-42` and `docker-compose.yml`.

- [ ] **Step 4: Strip the Tencent mirror out of the npm lockfile**

`frontend/package-lock.json` pins **328** dependency tarballs to `https://mirrors.tencent.com/npm/...`, so every `npm install` in this repo pulls packages from Tencent infrastructure. It is also why `npm ci` fails on npm 12 with `EALLOWREMOTE`.

Rewrite the host in place. The `integrity` sha512 hashes describe tarball *content*, which is byte-identical on any mirror, so this is lossless — same versions, same hashes, no dependency drift:

```bash
sed -i 's|https://mirrors\.tencent\.com/npm/|https://registry.npmjs.org/|g' frontend/package-lock.json
grep -c 'mirrors\.tencent\.com' frontend/package-lock.json || echo "MIRROR CLEAN"
```

Then prove the lockfile still resolves and the hashes still validate:

```bash
cd frontend && rm -rf node_modules && npm ci && cd ..
```

Expected: a clean install with no `EINTEGRITY` and no `EALLOWREMOTE`. An `E404` here would mean a package existed only on the Tencent mirror — report it rather than working around it.

`website-docs/package-lock.json` has 288 more, but Step 1 deletes that directory, so it needs no treatment.

- [ ] **Step 5: Restore Go checksum verification**

In `docker/Dockerfile.app:22-35`:

```diff
-ARG GOSUMDB_ARG=off
+ARG GOSUMDB_ARG=sum.golang.org
```

`GOSUMDB=off` disables module checksum verification — a supply-chain control that should not have been off by default.

- [ ] **Step 6: De-Sinicise the cloud-image script**

In `scripts/cloud-image/prepare.sh:13-21,32-35,87-103`, remove the `mirrors.tencent.com`, `mirrors.aliyun.com` and `mirror.ccs.tencentyun.com` defaults and the TencentOS branch. Keep `DOCKER_INSTALL_MIRROR` and `DOCKER_REGISTRY_MIRROR` as empty, documented opt-in variables. Translate the Chinese comments in this file to English now (it is small and self-contained; the bulk translation is Task 4).

- [ ] **Step 7: Verify nothing referenced the deleted paths**

```bash
rg -n "website-docs|miniprogram|dsh-weknora|README_CN|README_JA|README_KO" \
   --glob '!docs/superpowers/**' . | grep -v '^\.git/' || echo "NO DANGLING REFS"
```

Fix any hit — expected in `Makefile`, `docker-compose.yml`, `.github/workflows/`, `README.md`.

- [ ] **Step 8: Run the green gate and commit**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "chore: remove WeChat mini-program, docs site, translated READMEs and China package mirrors"
```

---

### Task 3: Rip out internationalisation

The highest-regression-risk task in the plan. `en-US.ts` is already a complete English translation (7,424 lines, 2 CJK lines), so this is mechanical substitution, not translation.

**Files:**
- Delete: `frontend/src/i18n/` (entire directory), `frontend/src/stores/localizedResourceCache.ts`, `internal/middleware/language.go`
- Modify: 208 `.vue` + 385 `.ts` files under `frontend/src/`; `frontend/package.json`; `internal/types/context_helpers.go:353-428`; `internal/types/placeholder.go:98`; `internal/config/config.go:331-430`; `config/builtin_agents.yaml`; `config/agent_type_presets.yaml`; `config/prompt_templates/*.yaml`

**Interfaces:**
- Consumes: Task 2's tree.
- Produces: no `t()` / `$t()` call sites anywhere; no `vue-i18n` dependency; `LanguageFromContext` and `ResolveLanguage` removed from `internal/types`; `{{language}}` no longer a valid prompt placeholder.

- [ ] **Step 1: Extract the English key→string map**

Before deleting anything, dump `en-US.ts` into a flat `key → English string` map. This is the substitution source for every call site.

```bash
node -e "
const m = require('./frontend/src/i18n/locales/en-US.ts');
const out = {};
(function walk(o, p) {
  for (const k in o) {
    const v = o[k], key = p ? p + '.' + k : k;
    typeof v === 'object' && v !== null ? walk(v, key) : out[key] = v;
  }
})(m.default || m, '');
require('fs').writeFileSync('/tmp/en-flat.json', JSON.stringify(out, null, 2));
console.log('keys:', Object.keys(out).length);
"
```

If the module is TypeScript-only, run it through `tsx` instead. Verify the key count is in the thousands before proceeding.

- [ ] **Step 2: Inventory every call site**

```bash
rg -n "\\\$?\bt\(['\\\"\`]" frontend/src --glob '*.vue' --glob '*.ts' -c | sort -t: -k2 -rn | head -30
rg -c "\\\$?\bt\(['\\\"\`]" frontend/src --glob '*.vue' --glob '*.ts' | awk -F: '{s+=$2} END {print "total call sites:", s}'
```

Record the total. After substitution it must be zero.

- [ ] **Step 3: Substitute, file by file**

Work in batches of ~20 files. For each call site, replace with the English literal from `/tmp/en-flat.json`:

```diff
- {{ t('chat.sendMessage') }}
+ Send message

- const label = t('settings.general.title')
+ const label = 'Settings'
```

Interpolated calls become template literals:

```diff
- t('kb.docCount', { n: count })
+ `${count} documents`
```

The 53 calls with Chinese fallbacks — `t('key', '中文')`, concentrated in `views/settings/McpServiceDialog.vue` (27), `ParserEngineSettings.vue` (17), `WebSearchSettings.vue` (4), `VectorStoreSettings.vue` (3) — take the English value from the map and **discard the Chinese fallback entirely**.

Run `npx vue-tsc --build` after each batch. Do not let errors accumulate.

- [ ] **Step 4: Delete the i18n machinery**

```bash
rm -rf frontend/src/i18n frontend/src/stores/localizedResourceCache.ts
```

Then remove from `frontend/src/main.ts` and `frontend/src/embed-main.ts`: the `createI18n` import, the `app.use(i18n)` call, and any locale bootstrapping.

- [ ] **Step 5: Remove the language switchers and locale plumbing**

- `views/settings/GeneralSettings.vue:12-26,225-238` — delete the language dropdown and its handler
- `views/auth/Login.vue:438,524-525` — delete the flag switcher
- `components/AgentEmbedChannelPanel.vue:538` — delete the per-channel locale picker; also replace the hardcoded brand colour `#07C05F` at line 492 with the theme token
- Delete the `Accept-Language` header from `utils/request.ts:68`, `api/chat/streame.ts:171`, `components/SkillInstallTimeline.vue:228`, `composables/useConfigSkillInstallProgress.ts:93`
- Delete the locale watchers at `stores/menu.ts:62` and `stores/modelProviders.ts:29`

- [ ] **Step 6: Drop the dependency and its scripts**

```bash
cd frontend
npm uninstall vue-i18n @intlify/core-base
```

Remove the `check-i18n`, `scan-i18n-gaps` and `regenerate-i18n-locales` scripts from `package.json`.

- [ ] **Step 7: Verify the frontend is locale-free**

```bash
rg -n "vue-i18n|\\\$t\(|useI18n|locale" frontend/src | grep -v '\.test\.' || echo "CLEAN"
npx vue-tsc --build
```

Expected: `CLEAN` and a clean type-check.

- [ ] **Step 8: Remove the backend language machinery**

```bash
rm -f internal/middleware/language.go
```

- Delete `EnvLanguage`, `DefaultLanguage`, `LanguageFromContext`, `ResolveLanguage`, `ResolveLanguageName`, `LanguageLocaleName` from `internal/types/context_helpers.go:9-17,353-428`, and delete `internal/types/context_helpers_test.go` cases covering them
- Delete `PromptTemplateI18n` and its resolution from `internal/config/config.go:331-430`
- Delete the `{{language}}` registration at `internal/types/placeholder.go:98`
- Unregister the middleware in `internal/router/`

- [ ] **Step 9: Hardcode English in the prompts**

```bash
rg -l '\{\{language\}\}' config/ internal/
```

In every hit, substitute the literal:

```diff
- Default response language: {{language}}; follow an explicit request for another language.
+ Default response language: English; follow an explicit request for another language.
- - ALWAYS respond in {{language}}
+ - ALWAYS respond in English
```

- [ ] **Step 10: Strip the locale blocks from the YAML configs**

In `config/builtin_agents.yaml`, `config/agent_type_presets.yaml` and `config/prompt_templates/*.yaml`, delete the `zh-CN`, `zh-TW`, `ja-JP` and `ko-KR` blocks, keeping only `default` / `en-US`. Update `internal/types/builtin_agent_localization_test.go` to assert the single remaining variant.

- [ ] **Step 11: Remove the locale environment variables**

Delete `WEKNORA_LANGUAGE`, `DEFAULT_LOCALE` and `VITE_DEFAULT_LOCALE` from `.env.example:63-72`, `docker-compose*.yml`, `frontend/public/config.js` and `frontend/docker-entrypoint.sh`.

- [ ] **Step 12: Run the green gate and commit**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "refactor: remove internationalisation, ship English only"
```

---

### Task 4: Translate contracts, delete Chinese comments

**Files:**
- Modify: `internal/handler/*.go` (1,671 Swagger annotation lines), `.env.example` (362 Chinese lines), `scripts/*.sh` (637 lines), `Makefile`, `docker-compose*.yml`, `internal/agent/act.go:165-180`, `internal/types/sandbox_network_policy.go:178-211`, `internal/application/service/knowledge_faq_import.go`, `internal/handler/initialization.go`, `internal/handler/knowledge.go`, `internal/handler/session/image_upload.go:100-106`, `internal/application/service/memory/topic_resolve.go:164-189`, `internal/application/service/memory/consolidate.go:453+`, `config/prompt_templates/rewrite.yaml:37-93`
- Regenerate: `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go`

**Interfaces:**
- Consumes: Task 3's tree.
- Produces: no CJK characters outside `licenses/`, `THIRD_PARTY_NOTICES.md` and test fixtures (which Task 7 handles). No new symbols.

- [ ] **Step 1: Translate the Swagger annotations**

```bash
rg -c '// @(Summary|Description|Param|Failure|Success|Tags)' internal/handler/ | sort -t: -k2 -rn
```

Translate each annotation to English, preserving the exact `// @Key value` structure — `swag` is whitespace-sensitive. Example:

```diff
-// @Summary 从文件创建知识
-// @Description 上传文件并创建知识条目
+// @Summary Create knowledge from a file
+// @Description Upload a file and create a knowledge entry
```

- [ ] **Step 2: Regenerate and contract-test the API docs**

```bash
make docs
go test ./docs/... -run TestSwaggerContract -v
rg -c '[\x{4e00}-\x{9fff}]' docs/swagger.json || echo "SWAGGER CLEAN"
```

Expected: `SWAGGER CLEAN` and a passing contract test.

- [ ] **Step 3: Translate the user-facing Chinese strings**

These bypass i18n entirely and are the strings a user actually sees:

- `internal/agent/act.go:165-180` — agent tool display names (`"查看外部工具"` → `"View external tools"`, `"深度思考"` → `"Deep thinking"`, `"检索知识库"` → `"Search knowledge base"`, and the rest)
- `internal/types/sandbox_network_policy.go:178-211` — 44 lines of validation errors
- `internal/application/service/knowledge_faq_import.go` — e.g. `werrors.NewBadRequestError("FAQ 条目不能为空")` → `"FAQ entry must not be empty"`
- `internal/handler/initialization.go`, `internal/handler/knowledge.go` — assorted error strings

- [ ] **Step 4: Translate the Chinese LLM prompts**

`internal/handler/session/image_upload.go:100-106` currently forces Chinese model output regardless of locale:

```diff
-"请分析这张图片的内容…用简洁的中文回答。"
+"Analyse the content of this image. Answer concisely in English."
-"…用简洁的中文回答，只输出分析结果。"
+"Answer concisely in English. Output only the analysis result."
```

Translate `internal/application/service/memory/topic_resolve.go:164-189` (`topicAdjudicationPrompt`, ~25 lines, plus the builder strings at 241-245: `"已有主题："` → `"Existing topics:"`, `"新出现的说法："` → `"Newly observed phrasings:"`) and `internal/application/service/memory/consolidate.go:453+` (`consolidationSystemPrompt`).

Rewrite the Chinese few-shot and negative examples at `config/prompt_templates/rewrite.yaml:37-38,44-64,81-93` as English equivalents that exercise the same behaviours.

- [ ] **Step 5: Translate `.env.example`**

All 362 Chinese comment lines to English. This file is the de-facto configuration reference — every variable needs a comment a reader can act on. Remove the blocks for variables deleted in Tasks 1-3 (`COS_*`, `OSS_*`, `TOS_*`, `OBS_*`, `TENCENT_VECTORDB_*`, `FEISHU_DOCX_PARSE_MODE`, `WEKNORA_LANGUAGE`, `DEFAULT_LOCALE`, `VITE_DEFAULT_LOCALE`).

- [ ] **Step 6: Translate the build and ops surface**

`scripts/*.sh` echo/log messages (`start_all.sh` 234 lines, `dev.sh` 128, `build_images.sh` 103), the `Makefile` help text (73 lines), and the `docker-compose*.yml` section headers (e.g. `# ========== 腾讯云 VectorDB ==========`, now a deleted service).

- [ ] **Step 7: Translate the remaining Chinese Markdown docs**

`website-docs/` is gone, but Chinese Markdown survives elsewhere. Translate each to English:

```bash
rg -l '[\x{4e00}-\x{9fff}]' --glob '*.md' --glob '!docs/superpowers/**' \
   --glob '!THIRD_PARTY_NOTICES.md' --glob '!CHANGELOG.md' .
```

Expected hits: `docs/README.md`, `docs/LITE.md`, `docs/poc/docker-sandbox/README.md`, `mcp-server/{README,INSTALL,EXAMPLES,PROJECT_SUMMARY,CHANGELOG,MCP_CONFIG}.md`, `docreader/README.md`, `migrations/mysql/README.md`, `cmd/milvus-migrate/README.md`, `examples/mcp-demo/README.md`, `examples/skills/README.md`, `internal/event/{SUMMARY,usage_example}.md`.

Special case — `client/`: `README.md` is Chinese and `README_EN.md` is its English twin (inverted from the root convention). Promote the English file:

```bash
git rm client/README.md && git mv client/README_EN.md client/README.md
```

Leave the root `CHANGELOG.md` (160 KB of historical entries) as-is; it is a record, not documentation.

- [ ] **Step 8: Delete the Chinese code comments**

Roughly 3,600 lines across Go, TypeScript and Python — including `docreader/` docstrings in `utils/request.py`, `auth.py` and `splitter/` (~275 lines) and the Chinese comments in `mcp-server/setup.py`. **Delete rather than translate** — a translated stale comment is worse than none. Preserve any comment that documents non-obvious behaviour; translate those few instead.

- [ ] **Step 9: Verify and commit**

```bash
rg -c '[\x{4e00}-\x{9fff}]' --glob '!licenses/**' --glob '!THIRD_PARTY_NOTICES.md' \
   --glob '!**/*_test.go' --glob '!testdata/**' . | sort -t: -k2 -rn | head -20
```

Remaining hits should be test fixtures only (Task 7). Then:

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "docs: translate API annotations, config reference and user-facing strings to English"
```

---

### Task 5: Rename the Go module path

Purely mechanical, ~1,900 files. Land it alone — it conflicts with every other branch.

**Files:** `go.mod`, `cli/go.mod`, `client/go.mod`, every `.go` file, `Makefile`, `scripts/get_version.sh`

**Interfaces:**
- Consumes: Task 4's tree.
- Produces: module path `github.com/ORG_PLACEHOLDER/EnterpriseRag`. No symbol changes.

- [ ] **Step 1: Rewrite the module declarations and every import**

```bash
grep -rl 'github.com/Tencent/WeKnora' --include='*.go' --include='go.mod' --include='Makefile' --include='*.sh' . \
  | xargs sed -i 's|github.com/Tencent/WeKnora|github.com/ORG_PLACEHOLDER/EnterpriseRag|g'
```

The `cli/` module is `github.com/Tencent/WeKnora/cli`; the same substitution handles it.

- [ ] **Step 2: Verify no old path survives**

```bash
rg -n 'Tencent/WeKnora' --glob '!docs/superpowers/**' --glob '!LICENSE' --glob '!THIRD_PARTY_NOTICES.md' . || echo "CLEAN"
```

Expected: `CLEAN`. `LICENSE` and `THIRD_PARTY_NOTICES.md` retain upstream references by design.

- [ ] **Step 3: Rebuild all three modules**

```bash
go build ./... && (cd cli && go build ./...) && (cd client && go build ./...)
go mod tidy && (cd cli && go mod tidy) && (cd client && go mod tidy)
```

- [ ] **Step 4: Run the green gate and commit**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "refactor: rename Go module to github.com/ORG_PLACEHOLDER/EnterpriseRag"
```

---

### Task 6: Rename brand identifiers

**Files:** `frontend/index.html`, `frontend/embed.html`, `frontend/public/weknora-widget.js` → `enterpriserag-widget.js`, ~20 `localStorage` keys across `frontend/src/`, ~60 `WEKNORA_*` env vars, `docker-compose*.yml`, `scripts/build_images.sh:132-282,361-362`, `helm/{Chart,values}.yaml`, `Makefile`, `cmd/desktop/wails.json`, `Formula/weknora-lite.rb`, `deploy/weknora-lite.service`, `cli/cmd/root.go:137`, `mcp-server/{setup.py,pyproject.toml}`, `internal/config/config.go` (~line 500), `.github/` (8 files), `LICENSE`

**Interfaces:**
- Consumes: Task 5's tree.
- Produces: `window.EnterpriseRag` replaces `window.WeKnora`; env prefix `ENTERPRISERAG_`; container prefix `EnterpriseRag-`.

- [ ] **Step 1: Rename the environment variables**

```bash
grep -rl 'WEKNORA_' --exclude-dir=.git --exclude-dir=docs/superpowers . \
  | xargs sed -i 's/WEKNORA_/ENTERPRISERAG_/g'
rg -n 'WEKNORA_' --glob '!docs/superpowers/**' . || echo "CLEAN"
```

`docreader/config.py` reads env vars through `_get_first_env`, which accepts alias lists — update the literal names there, do not add fallbacks (the spec forbids shims).

- [ ] **Step 2: Rename the browser storage keys and the widget global**

```bash
cd frontend
grep -rl 'weknora_\|WeKnora_\|weknoraDesktopTemplate' src public \
  | xargs sed -i 's/weknora_/enterpriserag_/g; s/WeKnora_/EnterpriseRag_/g; s/weknoraDesktopTemplate/enterpriseRagDesktopTemplate/g'
git mv public/weknora-widget.js public/enterpriserag-widget.js
sed -i 's/window\.WeKnora/window.EnterpriseRag/g; s/\bWeKnora\b/EnterpriseRag/g' public/enterpriserag-widget.js
cd -
```

Update every reference to the old widget filename in `embed.html` and the docs.

- [ ] **Step 3: Rename the containers, images and database**

In `docker-compose.yml`, `docker-compose.dev.yml`, `scripts/build_images.sh`, `helm/values.yaml`, `helm/Chart.yaml` and the `Makefile`:

```diff
-image: wechatopenai/weknora-app:${WEKNORA_VERSION:-latest}
+image: ORG_PLACEHOLDER/enterpriserag-app:${ENTERPRISERAG_VERSION:-latest}
-container_name: WeKnora-postgres
+container_name: EnterpriseRag-postgres
-BINARY_NAME=WeKnora
+BINARY_NAME=enterpriserag
-DOCKER_IMAGE=wechatopenai/weknora-app
+DOCKER_IMAGE=ORG_PLACEHOLDER/enterpriserag-app
```

Also `DB_NAME`, `ELASTICSEARCH_INDEX`, `OPENSEARCH_INDEX` and the four `LANGFUSE_INIT_*` values in `.env.example`, and `helm/values.yaml`'s `host: weknora.example.com` → `DOMAIN_PLACEHOLDER` and `dbName: weknora` → `enterpriserag`.

- [ ] **Step 4: Rename the User-Agent strings and fix the assertion**

Five distinct strings: `internal/application/service/embed_webhook.go:94`, `internal/application/service/tenant_skill_source.go:28`, `internal/datasource/connector/rss/client.go:27`, `internal/infrastructure/docparser/image_resolver.go:1013`, and `internal/infrastructure/web_search/{duckduckgo,searxng,serply}.go`.

`internal/infrastructure/web_search/serply_test.go:25` asserts the old value — update it in this same commit.

```bash
go test ./internal/infrastructure/web_search/... -v
```

Expected: PASS.

- [ ] **Step 5: Rename the HTML, packaging and CI surface**

- `frontend/index.html` — `<title>`, `meta description`, `meta keywords`; `frontend/embed.html` — `<title>`
- `cmd/desktop/wails.json` — `name`, `outputfilename`, `author`, `productName`, `copyright` (currently "Copyright 2026 Tencent"), `comments`
- `Formula/weknora-lite.rb` → `Formula/enterpriserag-lite.rb`, class `EnterpriseRagLite`, homepage and tarball URLs under `ORG_PLACEHOLDER`
- `deploy/weknora-lite.service` → `deploy/enterpriserag-lite.service`
- `cli/cmd/root.go:137` — `Use: "weknora"` → `Use: "enterpriserag"`
- `mcp-server/setup.py` + `pyproject.toml` — name `tencent-weknora-mcp` → `PYPI_NAME_PLACEHOLDER`, email `support@weknora.com` → `EMAIL_PLACEHOLDER`, URL under `ORG_PLACEHOLDER`
- `.github/` — `workflows/anydoc.yml`, `workflows/cli-e2e.yml`, `workflows/release-lite.yml`, `dependabot.yml`, `ISSUE_TEMPLATE/{bug_report,config,feature_request,question}.yml`

- [ ] **Step 6: Fix the stale viper search paths**

In `internal/config/config.go` (~line 500), replace the placeholder paths left by upstream:

```diff
-viper.AddConfigPath("$HOME/.appname")
-viper.AddConfigPath("/etc/appname/")
+viper.AddConfigPath("$HOME/.enterpriserag")
+viper.AddConfigPath("/etc/enterpriserag/")
```

- [ ] **Step 7: Rewrite the LICENSE**

Replace the Tencent header with the new owner's copyright, retaining the upstream MIT block and every third-party notice below it. Do not touch `THIRD_PARTY_NOTICES.md` or `licenses/`.

- [ ] **Step 8: Run the green gate and commit**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "refactor: rename brand identifiers to EnterpriseRag"
```

---

### Task 7: India defaults and demo data

**Files:** `config/config.yaml`, `docker-compose*.yml`, `.env.example`, `frontend/src/views/knowledge/settings/chunkingSamples.ts`, `testdata/wiki_test/`, ~290 test files with CJK fixtures; delete `frontend/src/views/dev/MarkdownTestPage.vue`

**Interfaces:**
- Consumes: Task 6's tree.
- Produces: the Devanagari fixtures that serve as the Risk R1 tripwire for Task 1's CJK deletions.

- [ ] **Step 1: Set the chunking markers**

```diff
-split_markers: ["\n\n", "\n", "。"]
+split_markers: ["\n\n", "\n", "।"]
```

`।` is the Devanagari danda, the sentence terminator in Hindi and Marathi.

- [ ] **Step 2: Set the regional defaults**

Add `TZ=Asia/Kolkata` to the service definitions in `docker-compose.yml` and `docker-compose.dev.yml`, and document `ap-south-1` (Mumbai) as the reference AWS region in the `.env.example` S3 block.

- [ ] **Step 3: Replace the demo corpora**

Rewrite `frontend/src/views/knowledge/settings/chunkingSamples.ts` (151 lines of Chinese demo text) and `testdata/wiki_test/*.md` with India-context English: an HR leave-and-expenses policy, a GST FAQ, and a product manual. Use `₹`/INR and Indian English spelling.

- [ ] **Step 4: Convert the CJK test fixtures — this is the R1 tripwire**

Across ~290 test files (2,564 CJK lines), heaviest in `internal/infrastructure/chunker/splitter_test.go` (117), `internal/application/service/memory/*_test.go` (~380), `frontend/src/utils/chatMarkdownRenderer.test.ts` (75), `internal/handler/session/artifact_reference_test.go` (57), `internal/im/think_test.go` (54):

- Where the fixture is incidentally Chinese, replace with English.
- Where the test genuinely exercises **multibyte handling** — the chunker and splitter tests especially — replace with **Devanagari**, not ASCII:

```diff
-  input := "什么是RAG架构"
+  input := "RAG आर्किटेक्चर क्या है"
```

- Memory tests asserting Chinese preferences (`"回答请用中文"`, `"始终使用中文回复我"`) become their English equivalents.

```bash
go test ./internal/infrastructure/chunker/... -v
```

**If these fail, Task 1 deleted generic multibyte logic rather than CJK-specific logic. Stop and restore it.** That is the entire point of this step.

- [ ] **Step 5: Delete the dev-only page**

```bash
rm -f frontend/src/views/dev/MarkdownTestPage.vue
```

Remove its route entry.

- [ ] **Step 6: Run the green gate and commit**

```bash
make fmt && make lint && make test && bash scripts/verify_frontend_pr.sh
git add -A
git commit -m "feat: India-context defaults, demo corpora and Devanagari multibyte fixtures"
```

---

### Task 8: Screenshots, README and final sweep

**Files:** `docs/images/*.png` (~40), `docs/assets/`, `README.md`

**Interfaces:**
- Consumes: Task 7's tree.
- Produces: the shippable repo.

- [ ] **Step 1: Bring the stack up**

```bash
make build-images && make start-all && make check-env
```

Expected: all containers healthy under the `EnterpriseRag-*` names.

- [ ] **Step 2: Re-capture the screenshots**

Drive the English UI and re-capture every screenshot referenced by `README.md`: `qa.png`, `agent-qa.png`, `settings.png`, `config.png`, `knowledgebases.png`, `knowledges.png`, `kb-document-list.png`, `kb-chunk-edit.png`, `skill-catalog.png`, `wiki-*.png`, `rbac-*.png`, `mcp-configuration/*.png`, `browser-*.png`, `chat-steer-*.png`, `langfuse.png`.

```bash
rm -f docs/assets/milvus-bm25-chinese-fixed.png
```

Keep `architecture.png` and `pipeline*.png` if they contain no Chinese text; otherwise redraw them.

- [ ] **Step 3: Run browser QA**

Follow the `ecc:browser-qa` skill against the running stack, read-only, with test credentials. Phase 1 smoke test plus Phase 2 interaction test. Specifically assert:

- No CJK glyph renders on any page
- No language switcher exists in Settings or on the login page
- The model-provider list offers exactly OpenAI, Anthropic, Gemini, OpenRouter and Generic
- The storage settings offer exactly Local, MinIO, S3 and Dummy
- No network request targets a `*.cn`, `*.qq.com`, `aliyuncs.com`, `volces.com` or `bigmodel.cn` host
- The sandbox reports unavailable rather than erroring

- [ ] **Step 4: Rewrite the README**

English throughout, new badges, no `weknora.weixin.qq.com` (16 refs) or `chatbot.weixin.qq.com` (8 refs) links, and the regenerated screenshots.

- [ ] **Step 5: Final sweep**

```bash
rg -in 'weknora|tencent|wechat|weixin|qq\.com|hunyuan|dashscope|volces|bigmodel' \
   --glob '!docs/superpowers/**' --glob '!.git/**' . | grep -v -E '^(LICENSE|THIRD_PARTY_NOTICES\.md|licenses/)'
rg -c '[\x{4e00}-\x{9fff}]' --glob '!docs/superpowers/**' --glob '!licenses/**' --glob '!THIRD_PARTY_NOTICES.md' .
rg -n 'PLACEHOLDER' --glob '!docs/superpowers/**' -c
```

Expected: the only survivors are `internal/utils/security.go:223` (`metadata.tencentyun.com`, a deliberate SSRF control), the retained licence files, and the intentional `*_PLACEHOLDER` tokens. Zero CJK.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "docs: regenerate English screenshots, rewrite README, final de-Sinicisation sweep"
```

---

## Handoff

After Task 8, hand the owner the placeholder checklist from spec §6. The tree builds with the placeholders in place (Go does not resolve module paths at build time), but publishing requires substituting all five tokens.

**Deferred to a separate spec:** rebuilding the code-execution sandbox on gVisor or rootless Docker, with its own threat model.
