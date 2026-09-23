# EnterpriseRag MCP Server

> **⚠️ Deprecated**
>
> EnterpriseRag now ships a built-in MCP server: create endpoints for a space under "Settings → Publish & Integrations → MCP Server". Multiple endpoints are supported, each with its own token, knowledge base scope and tool list, and clients connect directly to `/mcp/<endpoint_id>` over Streamable HTTP, so the Python process in this directory no longer needs to be deployed. This directory is kept only for compatibility with existing deployments and will be removed in a future release.

This is a Model Context Protocol (MCP) server that provides access to the EnterpriseRag knowledge management API.

## Quick Start

> We recommend following the [MCP configuration guide](./MCP_CONFIG.md) directly, in which case none of the steps below are required.

### 1. Install dependencies
```bash
pip install -r requirements.txt
```

### 2. Configure environment variables
```bash
# Linux/macOS
export ENTERPRISERAG_BASE_URL="http://localhost:8080/api/v1"
export ENTERPRISERAG_API_KEY="your_api_key_here"

# Windows PowerShell
$env:ENTERPRISERAG_BASE_URL="http://localhost:8080/api/v1"
$env:ENTERPRISERAG_API_KEY="your_api_key_here"

# Windows CMD
set ENTERPRISERAG_BASE_URL=http://localhost:8080/api/v1
set ENTERPRISERAG_API_KEY=your_api_key_here
```

### 3. Run the server

**Recommended approach - use the main entry point:**
```bash
python main.py
```

**Other ways to run it:**
```bash
# Use the original startup script
python run_server.py

# Use the convenience script
python run.py

# Run the server module directly
python enterpriserag_mcp_server.py

# Run as a Python module
python -m enterpriserag_mcp_server
```

### 4. Command line options
```bash
python main.py --help                 # Show help information
python main.py --check-only           # Only check the environment configuration
python main.py --verbose              # Enable verbose logging
python main.py --version              # Show version information
```

## Installing as a Python Package

### Install from PyPI

```bash
pip install PYPI_NAME_PLACEHOLDER
# Or run it directly with uvx (no prior installation needed)
uvx --from PYPI_NAME_PLACEHOLDER enterpriserag-mcp-server
```

> The official PyPI package name is **`PYPI_NAME_PLACEHOLDER`** (maintained by ORG_PLACEHOLDER/EnterpriseRag and published via Trusted Publishing).
> Please do not use the old community package `enterpriserag-mcp` any more.
> After installation, the command line entry points are still `enterpriserag-mcp-server` / `enterpriserag-server`.

### Development mode installation
```bash
pip install -e .
```

Once installed, you can use the command line tools:
```bash
enterpriserag-mcp-server
# or
enterpriserag-server
```

### Production mode installation
```bash
pip install .
```

### Building distribution packages
```bash
# Using setuptools
python setup.py sdist bdist_wheel

# Using modern build tooling
pip install build
python -m build
```

## Testing the Module

Run the test script to verify that the module works correctly:
```bash
python test_module.py
```

## Features

This MCP server provides the following tools:

### Space management
- `create_tenant` - Create a new space
- `list_tenants` - List all spaces

### Knowledge base management
- `create_knowledge_base` - Create a knowledge base
- `list_knowledge_bases` - List knowledge bases
- `get_knowledge_base` - Get knowledge base details
- `delete_knowledge_base` - Delete a knowledge base
- `hybrid_search` - Hybrid search

### Knowledge management
- `create_knowledge_from_file` - Create knowledge from a local file
- `create_knowledge_from_url` - Create knowledge from a URL
- `create_knowledge_from_text` - Create knowledge from text
- `update_knowledge_from_text` - Update manually written Markdown knowledge, either re-indexing it or saving it as a draft
- `list_knowledge` - List knowledge
- `get_knowledge` - Get knowledge details
- `delete_knowledge` - Delete knowledge

### Model management
- `create_model` - Create a model
- `list_models` - List models
- `get_model` - Get model details

### Session management
- `create_session` - Create a chat session
- `get_session` - Get session details
- `list_sessions` - List sessions
- `delete_session` - Delete a session

### Chat features
- `chat` - Send a chat message
- `agent_chat` - Invoke the agent to perform multi-step retrieval and tool calls

Both chat tools assemble events on SSE blank line boundaries, and a single event may contain multiple `data:` lines.
Any event that has not yet seen a blank line when the connection ends is discarded. The data buffer limit for a single
event is 8 MiB, and exceeding this limit returns an error and closes the response connection.

### Chunk management
- `list_chunks` - List knowledge chunks
- `delete_chunk` - Delete a knowledge chunk

## Troubleshooting

If you hit an import error, please make sure that:
1. All the required dependency packages are installed
2. Your Python version is compatible (3.10+ recommended)
3. There are no file name conflicts (avoid using `mcp.py` as a file name)

## Example of a Tool Call

<img width="950" height="2063" alt="118d078426f42f3d4983c13386085d7f" src="https://github.com/user-attachments/assets/09111ec8-0489-415c-969d-aa3835778e14" />

### Local upload directory boundary

All transports, including stdio, restrict local file uploads to the working
directory by default. Set `MCP_ALLOWED_UPLOAD_DIRS` to a comma-separated list of
trusted absolute directories when additional roots are needed. Starting in a
filesystem root requires an explicit directory configuration. Paths and symbolic
links resolving outside the selected roots are rejected.
