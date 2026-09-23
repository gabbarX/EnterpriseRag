# DocReader Service

DocReader is the gRPC service in the EnterpriseRag project that handles document parsing and processing. It supports reading many document formats, OCR recognition, multimodal processing and more.

## Docker Compose environment variables

In `docker-compose.yml`, the docreader service is configured with these environment variables:

```yaml
docreader:
  image: ORG_PLACEHOLDER/enterpriserag-docreader:${ENTERPRISERAG_VERSION:-latest}
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:${MINIO_PORT:-9000}
    - MINERU_ENDPOINT=${MINERU_ENDPOINT:-}
    - MAX_FILE_SIZE_MB=${MAX_FILE_SIZE_MB:-}
```

### Environment variable reference

#### 1. MINIO_ENDPOINT

- **Description**: Internal address of the MinIO service (container-to-container traffic)
- **Default**: `minio:9000`
- **Purpose**: DocReader uses this address to connect to MinIO object storage, for reading and storing files during document processing
- **Example**:
  ```yaml
  - MINIO_ENDPOINT=minio:9000  # Address inside the Docker network
  ```

#### 2. MINIO_PUBLIC_ENDPOINT

- **Description**: Public address of the MinIO service (external access)
- **Default**: `http://localhost:9000`
- **Purpose**: Used to generate externally reachable file URLs, for example the image links returned after a document is parsed
- **Important**:
  - If the files must be reachable from another machine or container, replace `localhost` with the actual IP address
  - You can set `MINIO_PORT` in the `.env` file to change the port
- **Example**:
  ```bash
  # .env file
  MINIO_PORT=9000
  ```
  Or edit docker-compose.yml directly:
  ```yaml
  - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000  # Use the real IP
  ```

#### 3. MINERU_ENDPOINT

- **Description**: Address of the MinerU service (optional)
- **Default**: Empty (MinerU is not used)
- **Purpose**: MinerU is an advanced document parsing service that handles more complex document structures. Once this variable is set, DocReader can call MinerU to parse documents
- **Example**:
  ```bash
  # .env file
  MINERU_ENDPOINT=http://mineru-service:8080
  ```

#### 4. MAX_FILE_SIZE_MB

- **Description**: Maximum upload file size, in MB
- **Default**: `50` MB
- **Purpose**: Limits the size of files the gRPC service will accept, so that an oversized file cannot crash the service or degrade performance
- **Example**:
  ```bash
  # .env file
  MAX_FILE_SIZE_MB=100  # Allow files of up to 100MB
  ```

## Other configurable environment variables

Besides the variables already set in docker-compose.yml, DocReader supports the following. Add them as required:

### gRPC configuration

- `DOCREADER_GRPC_MAX_WORKERS`: Maximum number of worker threads for the gRPC service (default: 4)
- `DOCREADER_GRPC_PORT`: Port the gRPC service listens on (default: 50051)

### Parser resource limits

- `DOCREADER_MARKITDOWN_MAX_WORKERS`: Maximum concurrency for MarkItDown parsing (default: 1; set to 0 to disable throttling)
- `DOCREADER_PDF_RENDER_MAX_WORKERS`: Maximum concurrency for rendering scanned PDFs to images (default: 1; set to 0 to disable throttling)
- `DOCREADER_PDF_RENDER_DPI`: DPI used when rendering scanned PDFs (default: 200)
- `DOCREADER_PDF_JPEG_QUALITY`: JPEG quality for rendered scanned PDFs (default: 85; the value is clamped to the range 1-95)

### OCR / VLM

DocReader no longer bundles its own OCR and VLM backends. Scanned PDFs are rendered to JPEG images and handed to the Go app, which calls the OCR/VLM services. See the main project documentation for those settings.

### Storage configuration

DocReader supports several storage backends:

#### MinIO/S3 storage (recommended)

- `STORAGE_TYPE`: Set to `minio`
- `MINIO_ACCESS_KEY_ID`: MinIO access key ID (default: minioadmin)
- `MINIO_SECRET_ACCESS_KEY`: MinIO secret access key (default: minioadmin)
- `MINIO_BUCKET_NAME`: MinIO bucket name (default: EnterpriseRag)
- `MINIO_PATH_PREFIX`: File path prefix
- `MINIO_USE_SSL`: Whether to use SSL (default: false)

### Proxy configuration

If external services must be reached through a proxy:

- `EXTERNAL_HTTP_PROXY`: HTTP proxy address
- `EXTERNAL_HTTPS_PROXY`: HTTPS proxy address

### Image processing

Scanned PDFs are rendered to JPEG images and passed to the Go app for OCR. If resource usage runs high while importing several large PDFs,
lower `DOCREADER_PDF_RENDER_MAX_WORKERS` or `DOCREADER_MARKITDOWN_MAX_WORKERS` first.

## Configuration examples

### Basic configuration (using MinIO)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:9000
    - MAX_FILE_SIZE_MB=50
```

### Advanced configuration (with MinerU enabled)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000
    - MINERU_ENDPOINT=http://mineru:8080
    - MAX_FILE_SIZE_MB=100
```

## Troubleshooting

### 1. The DocReader service will not start

Check the container logs for missing dependencies or permission errors. If needed, confirm that `MINIO_ENDPOINT` and the other storage environment variables are set correctly.

### 2. Images do not display

Check the `MINIO_PUBLIC_ENDPOINT` setting:
- Make sure the address is reachable from the browser
- When accessing from another machine, do not use `localhost`; use the actual IP address

### 3. File uploads fail

Check `MAX_FILE_SIZE_MB` and make sure the limit is large enough. The file size limits on the frontend and the backend must also agree.

## Service health check

The DocReader service is configured with a health check:

```yaml
healthcheck:
  test: ["CMD", "grpc_health_probe", "-addr=localhost:50051"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

You can check the service status with:

```bash
docker ps | grep docreader
docker logs EnterpriseRag-docreader
```

## Further details

- Service port: 50051 (gRPC)
- Container name: EnterpriseRag-docreader
- Network: EnterpriseRag-network
- Restart policy: unless-stopped
