# SDD ledger — plan: docs/superpowers/plans/2026-09-22-enterpriserag-rebrand.md

Spec: docs/superpowers/specs/2026-09-22-enterpriserag-rebrand-design.md (reachable, read)
BASE: 4afda18ec1ae3a166ab75cbb9b0c270f155929d0
Branch: chore/enterpriserag-rebrand

## Pre-flight conflict scan

### Cross-task rows (tasks sharing a file or interface)

| Tasks | Produces → Consumes | Finding |
|---|---|---|
| T1 → T8 | vendor catalog (5), storage factory (4), sandbox stub | Clean. T8 browser QA asserts exactly these sets. |
| T1 → T7 | CJK code paths deleted → Devanagari multibyte fixtures | **CONFLICT-1**: the R1 tripwire lands 6 tasks after the deletion it guards. |
| T1 → T5 | `all.go` written with old module path | Clean — plan states this explicitly; T5 renames all at once. |
| T1 → T6 | `SANDBOX_ENABLED` introduced unprefixed | **CONFLICT-2**: violates the `ENTERPRISERAG_*` naming constraint, and T6's sed only matches `WEKNORA_*`, so it would never be renamed. |
| T2,T3,T4,T6,T7 → `.env.example` | mirrors → locale vars → translation → env rename → regional defaults | Clean (sequential). T4 translating comments for vars T6 then renames is churn, not conflict. |
| T2 → T4 | `website-docs/` deleted | Clean — T4 Step 7 scopes its Markdown sweep with `--glob '!docs/superpowers/**'` and lists only surviving paths. |
| T2 → T6 | `.github/workflows/dsh-plugin.yml` deleted | Clean — T6 rebrands 8 files, dsh-plugin.yml not among them. |
| T3 → T6 | `WEKNORA_LANGUAGE` deleted | Clean — T6's sed simply finds nothing. |
| T5 → T6 | module path renamed | Clean — `WEKNORA_` is an env prefix, untouched by the module-path sed. |
| T5,T6 → T8 | `*_PLACEHOLDER` tokens introduced | Clean — T8's sweep expects them and exempts them. |

### Per-task self-consistency rows

| Task | Tests vs code / files created vs touched | Finding |
|---|---|---|
| T1 | Step 12 updates `vendors_test.go` to the 5-vendor set that Step 4 registers. Step 8 stubs sandbox and deletes its integration test. | Self-consistent. |
| T2 | Pure deletion + config edits; Step 6 greps for dangling refs to everything Step 1 deleted. | Self-consistent. |
| T3 | Step 1 extracts the map that Step 3 consumes; Step 7 verifies Step 3-6 completeness. | Self-consistent. |
| T4 | Step 2 regenerates Swagger from Step 1's annotations and contract-tests it. | Self-consistent. |
| T5 | Step 1 substitutes, Step 2 verifies zero survivors, Step 3 rebuilds all 3 modules. | Self-consistent. |
| T6 | Step 4 renames the UA string and fixes `serply_test.go:25` in the same step. | Self-consistent. |
| T7 | Step 4 is both the change and its own verification (the tripwire). | Self-consistent. |
| T8 | Step 3 browser QA asserts the sets T1 produced. | Self-consistent. |
| ALL | Green gate runs `verify_frontend_pr.sh` → `npm test`; no task installs node deps. | **CONFLICT-3**: first run fails on a fresh clone with no `node_modules`. |

### Rulings

Ruling: CONFLICT-1 — keep the tripwire in T7 rather than moving it earlier. Writing Devanagari fixtures before T1 would mean authoring them against files whose Chinese fixtures still exist, doubling the edit. Mitigation: T1's dispatch instructs the implementer to delete only CJK-specific branches and preserve generic rune-width/UTF-8 logic. Cost if wrong: rework in T7 to restore chunker logic deleted in T1, plus a re-run of T1's gate.

Ruling: CONFLICT-2 — T1 may introduce the flag as `SANDBOX_ENABLED`, but T6 must explicitly rename it to `ENTERPRISERAG_SANDBOX_ENABLED` as a named step rather than relying on its `WEKNORA_*` sed. Carried into T6's dispatch. Cost if wrong: one env var off-convention, cosmetic, trivially fixed later.

Ruling: CONFLICT-3 — treat `npm ci` in `frontend/` as a precondition of the green gate, not a plan step. Adding it to every task would re-install deps eight times. Carried into each dispatch as an environment note. Cost if wrong: one spurious gate failure that is obvious from the error text.

---

## Progress

Task 1: dispatched (implementer, general-purpose). BASE 4afda18e.

Task 1: BLOCKED (environment). Zero source files changed; branch created at 4afda18e, no commits.
  - Verified: no `go`, `make`, `gcc`/`cc`/`clang`, `golangci-lint`, `swag`, `pytest` on PATH.
  - Verified: docker daemon not running (npipe dockerDesktopLinuxEngine absent); only WSL distro is `docker-desktop`.
  - Verified: 3 CGO-only deps (pg_query_go/v6, sqlite-vec-go-bindings, duckdb-go-bindings/lib/windows-amd64).
    `internal/utils` imports pg_query, and is imported repo-wide => `go build ./...` fails on a PRISTINE tree.
    Step 10's method ("work the compiler error list to zero") is therefore unusable on this machine.
  - Verified: frontend baseline 1171 pass / 5 fail, all pre-existing Windows artifacts (CRLF regex mismatches
    in SandboxConfigEditorDrawer.network.test.mjs + SkillSettings.test.mjs; /bin/sh absent for cliIntegration.test.ts).
  - Agent side effects on the machine: Go 1.27.1 extracted to ~/sdk/go (not on PATH); ~/go/pkg/mod ~1.7GB;
    frontend/node_modules installed; orphaned msiexec PID 25380 from a killed winget install.

NEW FINDING (in scope, not in plan): npm lockfiles pin Tencent mirror URLs.
  frontend/package-lock.json      328 x https://mirrors.tencent.com/npm/...
  website-docs/package-lock.json  288 x  (moot - Task 2 deletes website-docs)
  This is why `npm ci` fails with EALLOWREMOTE on npm 12. Every npm install currently pulls from Tencent.
  Ruling: add a step to Task 2 regenerating frontend/package-lock.json against registry.npmjs.org.
  Cost if wrong: dependency versions could drift on regeneration; mitigated by running the frontend suite after.

NEW FINDING (in scope, not in plan): go.mod has `github.com/tencentcloud/CubeSandbox/sdk/go` as a DIRECT dep.
  Task 1 Step 8 says "keep every exported symbol in internal/sandbox/" but the cube_*.go clients must be
  deleted, so the stub must reimplement those symbols rather than merely guard them. Larger than the step implies.

NEW FINDING: internal/types/faq.go:740 is the ONLY caller of internal/textconv.ToSimplified.
  Task 1 Step 7 deletes internal/textconv/ => exactly one dangling reference to fix.

BLOCKED ON USER: execution environment. Options are container / WSL install / MinGW, all of which are
machine-level side effects outside this worktree. Per SDD stop conditions, asking rather than ruling.

## Environment resolved (user decision)

Ruling (user-directed): all builds/tests run in a Linux container; git stays on the host.
  - Docker Desktop started (was stopped). Daemon 29.6.2, linux, 12cpu, ~8GB.
  - Built `enterpriserag-dev` (3.3GB) from .superpowers/sdd/Dockerfile.dev:
    golang:1.26-bookworm + make/gcc/g++/libc6-dev + golangci-lint v2 + swag + node22 + pytest + uv + ripgrep.
  - Invocation: docker run --rm -v <repo>:/src -w /src -v enterpriserag-gomod:/go/pkg/mod enterpriserag-dev bash
  - Cost if wrong: container drift from the real CI image; mitigated because CI (.github/workflows) is restored
    and will independently verify on push.

Ruling (user-directed): npm lockfile Tencent mirrors removed by in-place host rewrite, not regeneration.
  integrity sha512 covers tarball CONTENT (mirror-independent) => lossless, zero version drift.
  Added as Task 2 Step 4; Task 2 steps renumbered to 1-8.

Plan amendments applied (3 findings from the blocked Task 1):
  - Global Constraints: container mandated; host-side Go explicitly NOT a substitute; documents the
    pristine-tree CGO build failure and the 5 host-only frontend failures.
  - Task 1 Step 7: notes internal/types/faq.go:740 as the single caller of textconv.ToSimplified.
  - Task 1 Step 8: CubeSandbox is a DIRECT dep whose types appear in exported signatures => stub must
    REIMPLEMENT the exported surface, not guard it. Instructs `go doc ./internal/sandbox` audit first.
  - Task 2 Step 4 (new): rewrite 328 mirrors.tencent.com URLs in frontend/package-lock.json.

NEW FINDING: .dockerignore contains Chinese comments (编辑器/IDE 本地配置, Python 虚拟环境/缓存, ...).
  Add to Task 4's translation sweep - it was not in the agent inventory.

Task 1: re-dispatch pending container smoke test (go build ./... on pristine tree inside container).

Container verified GREEN: go1.26.8, make 4.3, gcc 12.2, golangci-lint 2.13.2, node 22.23, pytest 9.1.1.
  `go build ./...` on PRISTINE tree => GO_BUILD_EXIT=0 inside container (fails on host).
  One image fix needed: added `libsqlite3-dev` — asg017/sqlite-vec-go-bindings/cgo includes sqlite3.h.
  That was the ONLY compile error; all other CGO deps (pg_query_go, duckdb) built fine.
  Modules cached in named volume `enterpriserag-gomod` => subsequent builds fast.
  Gotchas recorded for implementers: use `bash -c` not `bash -lc` (login shell drops Go from PATH);
  never pipe build/test through head/tail (loses exit code); use PowerShell for docker (Git Bash mangles paths).

Task 1: re-dispatched (implementer #2, general-purpose, container-based). BASE 4afda18e.

Task 1: complete (commit 11035c43, 327 files, +478/-54819). Verified by controller:
  vendors/ now exactly {all.go, anthropic, gemini, generic, openai, openrouter, vendors_test.go}.
  Working tree clean except the 7 pre-existing items.

CONTROLLER CORRECTION to implementer #2's report: it claimed "the container has no outbound DNS".
  FALSE. Verified directly: getent hosts registry.npmjs.org resolves (8 A records);
  curl https://registry.npmjs.org/ => HTTP 200. Its probes were sandboxed, not the container.
  Consequence: its 4 "DNS-dependent pre-existing failures" need re-verification, and npm ci
  IS possible => Task 2 Step 4 verification is achievable. Re-test dispatched.

Ruling: gate redefined from "all green" to "matches pristine baseline at the task's BASE commit".
  The repo has NO green suite at HEAD (4 failing Go tests + ~234 lint issues pre-exist on a
  pristine worktree of 4afda18e). Demanding green would block every task forever.
  Cost if wrong: a real regression could hide inside the pre-existing failure set; mitigated by
  requiring each implementer to diff against a pristine worktree baseline rather than assert.

Ruling: accept implementer's Step 9 deviation — it replaced the Chinese questionWords alternation
  with the English equivalent rather than deleting it, because deletion would leave removeQuestionWords
  with an empty regex (a latent bug) while the plan said to keep the function signature. Correct call.
  Cost if wrong: trivial, one regex.

Ruling: accept selective staging over the plan's `git add -A`. Seven pre-existing untracked/modified
  items (.gitignore, AGENTS.md, CODEBUDDY.md, GEMINI.md, QODER.md, opencode.jsonc, docs/superpowers/)
  do not belong in a de-Sinicisation commit. Made standard for all remaining tasks.
  Cost if wrong: docs/superpowers/ (spec+plan) stays untracked; fold in deliberately at the end.

Ruling: accept the WeKnoraCloud blast-radius expansion (~20 extra Go files incl. types.CredentialsConfig
  and Tenant.Credentials) and the extra lkeap_reranker.go deletion. Both were entailed by the plan's
  own "go.mod CLEAN" gate. Cost if wrong: tenant-level credential storage removed; per-model
  Parameters.AppID/AppSecret retained, so the capability survives.

DEFERRED to Task 7: implementer kept `types.Jieba` / `currentIsHan` (Chinese word segmentation) in
  query_expansion.go tokenize(), judging it generic multibyte tokenisation. It is NOT — Jieba is a
  Chinese segmenter. User chose "delete ALL CJK-related code paths". Delete in Task 7 where the
  Devanagari fixtures provide the tripwire. Also kept: toHalfWidth/normalizeSpaces in faq.go (genuinely
  generic Unicode normalisation - those stay).

Task 2: dispatched (implementer, general-purpose). BASE 11035c43.

CONTROLLER RE-TEST (with verified working network): 3 of 4 alleged "DNS" failures STILL FAIL.
  TestPutTenantParserConfigAdminPreservesRedactedSecrets  FAIL @10.00s  (hang/timeout, not resolution)
  TestSSRFSafeURL/blocked_internal_service_port           FAIL @10.045s (hang/timeout)
  TestDeploymentCapabilityKeysMatchFrontend               FAIL @0.00s   (CRLF - see below)
  So implementer #2's CONCLUSION (pre-existing, not caused by its change) stands; its DIAGNOSIS
  (no DNS) was wrong. Baseline-comparison ruling is unaffected and remains correct.

ROOT CAUSE of the frontend/CRLF failures identified:
  core.autocrlf=true, and .gitattributes forces eol=lf ONLY for *.sh, *.go, cli/acceptance/**/*.json.
  => all 208 .vue and 385 .ts files are CRLF in the working tree but LF in git object storage.
  => the Linux container reads CRLF; regex/byte-equality tests that parse frontend source fail.
  This is NOT an unavoidable "Windows artifact". It is a local checkout config.

PLANNED between Task 2 and Task 3 (Task 3 rewrites exactly these files, so it must land first):
  1. cp .gitignore /tmp/gitignore.keep        # plugin-modified, must survive
  2. git config core.autocrlf false
  3. git rm --cached -r -q . && git reset --hard   # re-materialise working tree as LF
  4. cp /tmp/gitignore.keep .gitignore
  5. re-verify: file frontend/src/App.vue  => expect no "CRLF line terminators"
  6. re-run the 3 failures; expect TestDeploymentCapabilityKeysMatchFrontend to now PASS,
     which also re-baselines the frontend suite for Task 3.
  Untracked files (AGENTS.md, CODEBUDDY.md, GEMINI.md, QODER.md, opencode.jsonc, docs/superpowers/)
  are untouched by reset --hard. Must be run ONLY when no implementer is active.

Task 2: complete (commit c29b9398, 278 files, +471/-41389, gate identical to baseline).
  Lockfile mirrors.tencent.com 328 -> 0; diff is 329 resolved-line changes ONLY, zero version/integrity drift.
  Implementer also found +1 mirror the surveys missed: registry.npmmirror.com pin for highlight.js.
  npm ci clean vs registry.npmjs.org (388 pkgs) => rewrite proven lossless.
  Retargeted 11 LIVE website-docs URLs that would have 404'd from the running UI.
  Implementer corrected TWO controller errors (both accepted): container npm is 10.9.8 not 12, so
  EALLOWREMOTE was unreproducible; and the container does NOT fix the CRLF failures as the plan claimed.

LF NORMALISATION APPLIED (between Task 2 and Task 3, no implementer active):
  core.autocrlf true -> false; git rm --cached -r . && git reset --hard; .gitignore restored from backup.
  Verified: frontend/src/App.vue no longer reports CRLF. Untracked files untouched.
  RESULT — baseline materially improved:
    frontend npm test:  1172 pass / 4 fail  ->  1176 pass / 0 FAIL / 1 skipped   (FULLY GREEN)
    go failures:        4  ->  2  (TestDeploymentCapabilityKeysMatchFrontend now PASSES)
  Remaining 2 Go failures: TestPutTenantParserConfigAdminPreservesRedactedSecrets (@10s hang),
    TestSSRFSafeURL/blocked_internal_service_port.
  Ruling: this was worth doing before Task 3 rather than tolerating. Task 3 rewrites 208 .vue + 385 .ts
  files; without it every frontend signal during the riskiest task would have been noise.
  Cost if wrong: none observed - strictly fewer failures.

DEV IMAGE v3: added python-is-python3; PYTHONUSERBASE=/pyuser on a named volume so pip --user deps
  survive --rm runs (docreader/mcp-server pytest need urllib3/grpcio). Mount -v enterpriserag-pyuser:/pyuser.
  Also recorded: `npm run build` needs docker -m 8g AND NODE_OPTIONS=--max-old-space-size=6144 (else exit 134 OOM).
  Also recorded: verify_frontend_pr.sh set -e's out at npm test, so type-check/build must be run separately.

OPEN LOOSE END: .gitignore:59 still has `miniprogram/project.private.config.json`. Implementer correctly
  refused to stage .gitignore (controller constraint). One-line cleanup at the end.

Task 3: dispatched (implementer, general-purpose). BASE c29b9398.
  Baseline handed over: frontend MUST stay 0 fail; go exactly 2 named failures; lint <= 234.
  Explicitly told NOT to touch Jieba/currentIsHan (deferred to Task 7).

PYTHON GATE NOW WORKS (was never running in any prior task):
  Root causes: (a) Debian PEP 668 blocks pip --user without --break-system-packages;
    (b) 7 missing third-party modules (markitdown, PIL, docx, ebooklib, openpyxl, pypdfium2, lxml);
    (c) docreader is FLAT-LAYOUT so `pip install .` fails ("Multiple top-level packages discovered")
        and `import docreader` needs PYTHONPATH=/src;
    (d) lxml dropped the html_clean extra -> needs the separate lxml_html_clean package.
  docreader collection: 0 tests / 20 errors  ->  207 tests / 0 errors.
  mcp-server: pip install . succeeded.
  CORRECT INVOCATION for all later tasks:
    docker run --rm -m 8g -v <repo>:/src -w /src \
      -v enterpriserag-gomod:/go/pkg/mod -v enterpriserag-pyuser:/pyuser enterpriserag-dev bash -c \
      'cd /src/docreader && PYTHONPATH=/src python -m pytest -q'
  Deps persist in the named volume enterpriserag-pyuser.
  Relevance: Task 4 edits docreader docstrings/comments + mcp-server docs. Low risk, but "the suite
  has never run" is exactly where a real break hides. Closed before Task 4 rather than during it.

PYTHON BASELINE ESTABLISHED (at c29b9398, post-LF-normalisation):
  docreader  : 16 failed, 178 passed, 13 skipped, 11 subtests passed  (exit 1) -- PRE-EXISTING
               includes tests/test_ssrf.py + test_ssrf_proxy.py failures, mirroring the Go
               TestSSRFSafeURL failure => same underlying network-policy environment cause.
  mcp-server : 6 passed (exit 0)  -- needed pytest-asyncio; --strict-config turned the missing
               plugin into a hard "Unknown config option: asyncio_mode" error (exit 4, no tests run).
  Both now installed in the enterpriserag-pyuser volume.

CONSOLIDATED GATE for Tasks 4-8 (must match, not beat):
  frontend npm test        1177 tests / 1176 pass / 0 fail / 1 skipped     <- MUST STAY 0 FAIL
  frontend type-check      exit 0
  frontend build           exit 0  (needs -m 8g AND NODE_OPTIONS=--max-old-space-size=6144)
  go make test             exactly 2 failures: TestPutTenantParserConfigAdminPreservesRedactedSecrets,
                                               TestSSRFSafeURL/blocked_internal_service_port
  go make lint             234 issues
  docreader pytest         16 failed / 178 passed / 13 skipped
  mcp-server pytest        6 passed

Task 3: complete (commit f3bf1764, 355 files, +9346/-51163). THE BIG ONE - verified by controller:
  t()/$t() call sites  7060 -> 0 (verified independently: grep returns 0)
  vue-i18n + @intlify   GONE from package.json
  frontend/src/i18n/    DELETED (incl. 5 locale bundles ~2.2MB)
  internal/middleware/language.go  DELETED
  frontend npm test  1148 pass / 0 FAIL / 1 skip  (count 1177->1149; deleted i18n tests, net -28)
  type-check 0, build 0, lint 234 (unchanged), go build/vet 0.

  Controller checked the one thing that could have been a live defect: 4 files still contain
  `{{language}}`. Verified NOT a leak - internal/agent/prompts.go:378 is a DOC COMMENT, and
  prompts.go:398 still substitutes "language": language => always "English". Other 3 are tests. Clean.

CONTROLLER ERROR CORRECTED by implementer: my "exactly 2 Go failures" baseline was measured from a
  PARTIAL package run (./internal/handler/... ./internal/utils/... only). Implementer ran full `make test`
  and found 4, verifying each against a throwaway worktree at c29b9398. Real set:
    TestPutTenantParserConfigAdminPreservesRedactedSecrets      (10.00s timeout)
    TestSSRFSafeURL/blocked_internal_service_port               (10.01s timeout)
    TestUpdateMCPService_AppliesNonScalarUpdateWithoutName      (10.00s timeout)
    TestSkillPythonVerifier/a_requirement_the_venv_does_not_carry (venv/pip env)
  NOTE: these are FLAKY - three are 10s network-dial timeouts and did not all appear in my run.
  Ruling: treat the set as "these 4 names only"; any failure OUTSIDE the set is a regression.
  Cost if wrong: a genuine regression in one of those 4 tests would be masked. Accepted - they are
  network-policy tests that cannot pass in this container regardless.

Task 3 deliberate residue (recorded, not blocking):
  - internal/types/task.go Language fields retained, always written "en-US" (cascades through ~12 worker
    structs; not in plan). Dead weight, safe.
  - frontend/src/stores/modelProvidersState.ts still reads locale from backend catalog labels/descriptions
    map (backend API shape, not frontend i18n); always resolves en-US now.
  - 141 HTML-unsafe chars emitted as " / < inside <template> regions (runtime-identical) -
    necessary because inlined quotes/angle-brackets would break attribute/tag parsing.
  - 34 files gained module-scope Record<string,string> label maps for the 228 DYNAMIC keys that cannot
    be inlined; 2 new keyed modules (constants/auditActionLabels.ts, config/guideCopy.ts).
  - 23 keys were MISSING from en-US entirely and previously rendered as raw key strings at runtime -
    implementer wrote real English for each. That is a latent-bug fix, not just a refactor.

Task 4: dispatched (implementer, general-purpose). BASE f3bf1764. Given .dockerignore as an extra item.

TASK 4 IN FLIGHT - two REGRESSIONS/DEFECTS found via its helper agents, both ruled on and assigned to it:

Ruling 1: DEAD WeKnoraCloud FRONTEND = live regression from Task 1. Task 1 removed the entire
  WeKnoraCloud backend (verified: zero routes in internal/router/, zero handlers) but left the UI wired:
    frontend/src/views/settings/WeKnoraCloudSettings.vue        (15.6KB, still present)
      line 200 imports saveWeKnoraCloudCredentials + getWeKnoraCloudStatus from @/api/model -> dead endpoints
    frontend/src/views/settings/Settings.vue:89  renders it; :231 imports it  -> VISIBLE SETTINGS TAB
    frontend/src/components/ModelEditorDialog.vue:213 button -> :1142 goToWeKnoraCloudSettings
      -> uiStore.openSettings('weknoracloud')
  A user clicking that tab gets a runtime error. Directed Task 4 to delete the whole surface incl. the
  api/model functions and the 'weknoracloud' tab id. NOT a rename - WeKnoraCloud pointed at Tencent's
  hosted SaaS (weknora.weixin.qq.com) which the owner cannot serve.
  Cost if wrong: none - the backend is provably gone.

Ruling 2: FAQ CSV TEMPLATE SHIPS CHINESE HEADERS. FAQEntryManager.vue parser matches '标签(必填)',
  '问题(必填)', '机器人回答', '相似问题', '反例问题', '分类', '是否停用' and cell values '是'/'否';
  the downloadable example generators (~line 2186 CSV, ~2223-2230 Excel) EMIT those same headers.
  Self-consistent, so nothing breaks - but an English-only product hands users a Chinese template, and
  the UI tip ~line 515 ALREADY describes English headers, so the screen contradicts itself today.
  Directed Task 4 to change BOTH sides together to English headers + Yes/No, verify the round trip,
  and add NO dual-header compatibility (consistent with the project-wide no-shims decision).
  Cost if wrong: existing users' saved Chinese CSVs stop importing. Accepted - matches the hard-rename
  decision already taken for env vars, storage keys and the widget global.

PROCESS NOTE: Task 4's implementer fanned out to >=4 helper subagents. Told it to stop and finish the
  work itself. The helpers did careful work AND both defects above surfaced only because they wrote down
  what they had scoped out - but a helper lacking task context is exactly where "leave the Chinese CSV
  headers, they're program data" gets decided silently. SDD rule stands: review comes from the controller.

Ruling 3 (Task 4): VLM descriptionLanguage dropdown reduced to English only.
  UploadConfirmDialog.vue ~397-407 offered Chinese/English/Korean/Russian for multimodalConfig.
  descriptionLanguage (the language the vision model writes image descriptions in). Inconsistent:
  Task 3 hardcoded every {{language}} prompt placeholder to "English", so the product forces English
  output everywhere EXCEPT through this control. Also a pure Chinese-first-market artefact.
  Kept the field + clearable "Follow document language" (backend contract); removed the 3 options.
  Explicitly told NOT to add Hindi/Indic options - owner chose "defaults and docs only" for India fit.
  Cost if wrong: a user wanting non-English image descriptions loses that; deliberate per English-only.

SIGNIFICANT FINDING - comment deletion can break SOURCE-PARSING tests:
  frontend/src/views/platform/fileDrop.test.mjs:11 did
     source.indexOf('// 组件挂载时添加全局事件监听器')
  i.e. it slices index.vue's RAW TEXT using a Chinese comment as the anchor. Deleting that comment
  returned -1, the slice swallowed the <style> block, and ALL 7 TESTS CRASHED with
  "SyntaxError: Invalid regular expression". Helper caught it by round-tripping (7/7 pass -> 7/7 fail),
  re-anchored on 'onMounted(() => {' for a byte-identical slice, 7/7 pass again, then proactively
  re-ran 3 sibling source-parsing tests (12/12).
  CLASS OF BUG: comments are load-bearing in this repo's frontend tests. Same pattern as
  SandboxConfigEditorDrawer.network.test.mjs and SkillSettings.test.mjs (the CRLF casualties).
  => Retroactively validates the LF normalisation: had the frontend gate still sat at "4 known
  failures", this breakage would have landed INSIDE the noise and been indistinguishable from it.
  => Carry to Tasks 5-8: any task deleting or rewriting comments must re-run the source-parsing tests.

CLASS OF DEFECT IDENTIFIED (all 3 Task 4 rulings share it): internally-consistent code that becomes
  WRONG only because the product's intent changed. Tests pass on all three. No CJK scan flags the VLM
  dropdown (value="Chinese" is ASCII). The 3 survey agents catalogued Chinese TEXT and China-origin
  INTEGRATIONS thoroughly but nothing catalogued "features that only make sense for a Chinese market".
  => Task 8 browser QA must look for this class specifically, not just for CJK glyphs.

QUEUED FOR TASK 7 (India defaults) - MinerU OCR language defaults to CHINESE, both layers:
  frontend/src/views/settings/ParserEngineSettings.vue:389  mineru_language: 'ch'
  frontend/src/views/settings/ParserEngineSettings.vue:394  mineru_cloud_language: 'ch'
  frontend/src/views/settings/ParserEngineSettings.vue:522,527  ?? 'ch' fallbacks
  internal/infrastructure/docparser/mineru_converter.go:58        stringOr(overrides[...], "ch")
  internal/infrastructure/docparser/mineru_cloud_converter.go:51  stringOr(overrides[...], "ch")
  => every PDF parsed via MinerU runs CHINESE OCR out of the box. On English/Devanagari documents this
     degrades extraction, which degrades chunking and embedding, which degrades retrieval. Functional,
     not cosmetic. Change all 6 to 'en'.
  Invisible to every CJK scan in this project - 'ch' is ASCII. Same class as the VLM dropdown.

ALSO QUEUED FOR TASK 7: internal/infrastructure/chunker/tokens.go:18  LangChinese = "zh"
  Chunker language constant - assess alongside the deferred Jieba/currentIsHan deletion.

TASK 7 NOW CARRIES (consolidated):
  1. types.Jieba / currentIsHan deletion (deferred from Task 1) - Devanagari fixtures are the tripwire
  2. chunker/tokens.go LangChinese constant
  3. MinerU OCR defaults 'ch' -> 'en' (6 sites, frontend + Go)
  4. split_markers CJK punctuation -> Devanagari danda (config/config.yaml AND the frontend
     UploadConfirmDialog.vue separatorOptions + createDefaultUIState + initFromKbInfo defaults,
     AND KnowledgeBaseEditorModal.vue:760,905 - four places, not just the one the plan lists)
  5. TZ=Asia/Kolkata, ap-south-1 docs, India-context demo corpora, CJK test fixtures -> English/Devanagari

QUEUED FOR TASK 6 (brand/links) - Chinese-locale DOC URLs in surviving code (5 files, node_modules excluded):
  internal/models/api/openaichataudio/client.go:12      https://help.aliyun.com/zh/model-studio/qwen-asr-api-reference
  internal/models/api/openaichataudio/golden_test.go:18 (same URL)
    ^ NOTABLE: the aliyun VENDOR was deleted in Task 1, but this surviving OpenAI audio client still
      documents itself against ALIBABA's Qwen ASR reference. Repoint at the OpenAI audio API docs.
  internal/application/repository/retriever/milvus/repository.go:202  https://milvus.io/docs/zh/full-text-search.md
    ^ Chinese-language Milvus docs; English is the same URL without /zh.
  mcp-server/CHANGELOG.md:5  https://keepachangelog.com/zh-CN/1.0.0/
  mcp-server/CHANGELOG.md:6  https://semver.org/lang/zh-CN/
  All are ASCII URLs => invisible to every CJK scan. Same class as the MinerU 'ch' default.

STATUS: Task 4 implementer still running; it fanned out to ~8 helper subagents before I told it to stop.
  Helper output so far (their own reports): ~950 comments deleted, ~200 translated across the settings,
  agent, organisation, knowledge, platform and mcp-server surfaces. User-visible strings changed: ~21
  (mostly because Task 3 had already inlined them as English literals).
  Awaiting the implementer's OWN report with the 7 gate numbers + commit SHA.

Task 4: complete (3f8b2ec2 + fix round e8698228). 510 files, +30665/-35668. All 7 gates == baseline.
  1650 Swagger annotations translated; docs/swagger.{json,yaml} + docs.go regenerated CJK-CLEAN.
  Delivered the 3 controller rulings: FAQ CSV headers EN both sides (incl. backend export/import report),
  WeKnoraCloud surface removed (far larger than the 6 items I listed - also 28 sites in ModelEditorDialog,
  a parser engine, utils/weknoraCloudModels.ts, settingsAccess.ts, apiKeyCapabilities.ts),
  descriptionLanguage reduced to English in TWO files not one.
  Also fixed OUTSIDE remit: chat/index.vue:517 called rewindSkipMessage(reason, t) with t undefined
  => ReferenceError on every rewind-skip, left by Task 3. Plus 2 more comment-anchored tests.
  Contract test: proved COMMITTED swagger docs were STALE (269 defs vs 247 regenerated) via pristine
  worktree, kept test intent, added TestModelInUseErrorCodeContract pinning ErrModelInUse==2300. Sound.

*** METHODOLOGY FINDING - affects EVERY scan run on this project, including the 3 survey agents ***
  .gitignore line 1-2 is `.*` (ignore every hidden file). RIPGREP HONOURS .gitignore EVEN WITH --hidden.
  => every tracked dotfile except .env.example/.gitignore/.github/ was INVISIBLE to all rg scans.
  That is why .dockerignore had to be handed over, and why .env.lite.example (467 CJK chars),
  .air.toml and .golangci.yml were missed until a git-ls-files-based sweep.
  Separately: THIS HOST's `grep -P` rejects \x{4e00} ("character value too large") and returns 0 matches
  with exit 2 - a silent false negative. I hit this myself and wrongly reported "CJK fully eliminated".
  AUTHORITATIVE SWEEP for Task 8 and all later scans:
      git ls-files -z | xargs -0 rg -c '[\x{4e00}-\x{9fff}]'
  Carried into Task 5's dispatch and must go into Tasks 6-8.

CJK FINAL STATE after e8698228 (tracked text files, excl. licenses/NOTICES/CHANGELOG/binaries):
  21 files. 9 are docreader/mcp-server test + evalset fixtures (Task 7).
  12 non-test survivors, ALL declared program data:
    chunkingSamples.ts 150 (Task 7 corpus), types/memory.go 7 (secret-detect regex + "remember this"
    triggers), paradedb migration 5 (chinese_lindera BM25 smoke corpus), chunker/patterns.go 5
    (PageFooterPattern/ChineseChapterPattern), wiki_ingest.go 4 (rateLimitErrorIndicators matched on
    upstream error bodies), ocr_sanitizer.go 4 (knownEmptyReplies matched on VLM output),
    wiki_page.go 1, temporary_document.go 1, ptyEchoPredictor.ts 1, finalAnswer.ts 1.

QUEUED FOR TASK 6 (additions from the dotfile audit - 16 tracked dotfiles, 5 with brand strings):
  .air.toml:10        exclude_dir contains "WeKnora-Chrome-Extension" (dir does not exist in this repo)
  .dockerignore:31,41,42   WeKnora-Chrome-Extension/, WeKnora, WeKnora-lite
  .env.example        WEKNORA_VERSION, WEKNORA_BOOTSTRAP_*, prose
  .env.lite.example   "WeKnora Lite" paths, weknora-lite.log, data/weknora.db
  .gitignore:31,32,37,38   WeKnora, WeKnora-lite, data/weknora.db*  (controller must stage this one
                           manually at the end - implementers are barred from touching .gitignore)
  packages/.gitkeep   ORPHAN - packages/dsh-weknora was deleted in Task 2, only .gitkeep remains.
  PLUS from Task 4's routing: internal/types/storagebackend.go dead COS_*/TOS_*/OSS_*/OBS_* readers;
    internal/handler/sandbox_check.go:353 {label:"cn:baidu", url:"https://www.baidu.com"} egress probe;
    the 5 Chinese-locale doc URLs (aliyun x2, milvus /zh, keepachangelog+semver zh-CN).

Task 5: dispatched (implementer, general-purpose). BASE e8698228. Module path rename, ~1900 files.
