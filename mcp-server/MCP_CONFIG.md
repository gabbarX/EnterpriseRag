# Running the EnterpriseRag MCP Server with uv

> We recommend using `uv` to run Python-based MCP services.
>
> You can also install it from PyPI with `pip install PYPI_NAME_PLACEHOLDER`, or run it with `uvx --from PYPI_NAME_PLACEHOLDER enterpriserag-mcp-server` (the official package name is `PYPI_NAME_PLACEHOLDER`, maintained by [ORG_PLACEHOLDER/EnterpriseRag](https://github.com/ORG_PLACEHOLDER/EnterpriseRag)).

## 1. Install uv

```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Or use Homebrew (macOS)
brew install uv

# Windows
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
```

## 2. MCP Client Configuration

### Claude Desktop configuration

Add the following in the Claude Desktop settings:

```json
{
  "mcpServers": {
    "enterpriserag": {
      "args": [
        "--directory",
        "/path/EnterpriseRag/mcp-server",
        "run",
        "run_server.py"
      ],
      "command": "uv",
      "env": {
        "ENTERPRISERAG_API_KEY": "your_api_key_here",
        "ENTERPRISERAG_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### Cursor configuration

In Cursor, edit the MCP configuration file (usually at `~/.cursor/mcp-config.json`):

```json
{
  "mcpServers": {
    "enterpriserag": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/EnterpriseRag/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "ENTERPRISERAG_API_KEY": "your_api_key_here",
        "ENTERPRISERAG_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### KiloCode configuration

For KiloCode or any other editor that supports MCP, the configuration is as follows:

```json
{
  "mcpServers": {
    "enterpriserag": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/EnterpriseRag/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "ENTERPRISERAG_API_KEY": "your_api_key_here",
        "ENTERPRISERAG_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### Other MCP clients

For a generic MCP client configuration:

```json
{
  "mcpServers": {
    "enterpriserag": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/EnterpriseRag/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "ENTERPRISERAG_API_KEY": "your_api_key_here",
        "ENTERPRISERAG_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```
