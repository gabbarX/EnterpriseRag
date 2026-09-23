# Milvus Multilingual BM25 Migration

Milvus collections created by older EnterpriseRag versions have only a single default text analyser, so content in some languages may end up without usable BM25 keywords. `milvus-migrate` keeps the old collection intact and copies the existing dense vectors into a new multilingual collection; Milvus then regenerates the BM25 sparse vectors based on the `language` field of each row.

The metric for the dense vectors (IP / COSINE / L2) is read from the embedding index of the source collection by default. Do not change it to something different from the source collection, otherwise the same vectors will be ranked by the wrong distance function.

## Usage

Make sure Milvus is running, then execute this from the project root:

```bash
go run ./cmd/milvus-migrate --source enterpriserag_embeddings --target enterpriserag_embeddings_multilingual
```

If `MILVUS_ADDRESS` and `MILVUS_COLLECTION` are already exported in your current shell, the corresponding flags can be omitted. Note that simply writing the variables into `.env.local` does not inject them into the `go run` process; when in doubt, pass the flags explicitly:

```bash
go run ./cmd/milvus-migrate \
  --address 127.0.0.1:19530 \
  --source enterpriserag_embeddings \
  --target enterpriserag_embeddings_multilingual
```

To double-check the metric, pass the same `--metric-type` as the source collection (or set the `MILVUS_METRIC_TYPE` environment variable). The value you pass must match the source index, otherwise the migration fails.

Once the migration has finished, change EnterpriseRag's `MILVUS_COLLECTION` to the target prefix and restart the service, **leaving `MILVUS_METRIC_TYPE` exactly as it was**:

```dotenv
MILVUS_COLLECTION=enterpriserag_embeddings_multilingual
```

Collections are listed by an exact `{prefix}_{dimension}` match, so `enterpriserag_embeddings` will not accidentally pick up `enterpriserag_embeddings_multilingual_*`. Only after the prefix is switched will new writes as well as vector and keyword retrieval use the new collection.

The migration tool does not delete the old collection. Once you have confirmed that BM25 recall works for every language you care about, delete the old collection through your Milvus administration tooling; take a backup before you do.

The migration reads 64 rows per batch by default and only reads the fields needed to rebuild the target collection, so it never reads the BM25 sparse vectors generated in the old collection. If individual text chunks are unusually long, lower the batch size explicitly, for example by appending `--batch-size 32`.

## Notes for Windows PowerShell

EnterpriseRag's `internal/utils` parses SQL using `pg_query_go`, and that dependency requires CGO. If you run `go run` directly in a session where `CGO_ENABLED=0`, you will see `undefined: pg_query.Parse` or `undefined: pg_query.Deparse`.

The project ships with MSYS2 GCC, so you can enable CGO temporarily in the current PowerShell session before running the migration. The settings below affect only the current window and do not modify your system-wide Go configuration:

```powershell
# Confirm that the current directory is the project root containing go.mod
if (!(Test-Path -LiteralPath '.\go.mod')) { throw 'Please switch to the EnterpriseRag project root first' }

# Use the bundled GCC so that Go can find a C compiler
$compilerBin = Join-Path (Get-Location) '.local-tools\msys64\ucrt64\bin'
$env:CGO_ENABLED = '1'
$env:CC = Join-Path $compilerBin 'gcc.exe'
$env:CXX = Join-Path $compilerBin 'g++.exe'
$env:PATH = "$compilerBin;$env:PATH"

# Run the migration; the metric follows the source collection and the old collection is kept
go run ./cmd/milvus-migrate --address 127.0.0.1:19530 --source enterpriserag_embeddings --target enterpriserag_embeddings_multilingual
```

If your project directory does not contain `.local-tools\msys64\ucrt64\bin`, install a working GCC first and point `$env:CC` and `$env:CXX` at the absolute paths of the corresponding `gcc.exe` and `g++.exe`.
