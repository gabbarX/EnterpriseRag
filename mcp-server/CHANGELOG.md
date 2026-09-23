# Changelog

All notable changes to this project are recorded in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Added the `create_knowledge_from_text` tool: it creates a knowledge entry from manually written Markdown text by calling the existing `/knowledge-bases/{id}/knowledge/manual` endpoint, completing the "text" part of #323. It defaults to `status="publish"`, so the entry enters the parsing/indexing pipeline as soon as it is created and can be retrieved; pass `status="draft"` to only save it without indexing.
- Added the `update_knowledge_from_text` tool: it updates manually written Markdown knowledge through the existing `PUT /knowledge/manual/{id}` endpoint; it re-indexes by default and can also save the entry as a draft, completing #2378.
- Listed the existing `create_knowledge_from_file` in the README tool list.

## [1.1.1] - 2026-07-30

### Changed
- The official PyPI package name is now **`PYPI_NAME_PLACEHOLDER`** (published by the CI of the ORG_PLACEHOLDER/EnterpriseRag repository through Trusted Publishing).
  The earlier community package `enterpriserag-mcp` is not officially maintained, so please update your installation commands.
- Project URLs now point to the official [ORG_PLACEHOLDER/EnterpriseRag](https://github.com/ORG_PLACEHOLDER/EnterpriseRag) repository (the `mcp-server/` directory).

### Fixed
- The HTTP transport restores `stateless_http=True`, which matches the pre-migration `StreamableHTTPSessionManager(stateless=True)` behaviour.
- The SSE transport restores the `/sse/messages/` message endpoint, which matches the pre-migration routing.
- `EnterpriseRagClient` now uses a thread-local `requests.Session`, which avoids Session race conditions when MCP 2.x makes concurrent calls from a thread pool.
- File upload (`create_knowledge_from_file`) now respects the `ENTERPRISERAG_VERIFY_SSL` setting.

### Note
- When a tool fails, MCPServer 2.x returns `CallToolResult(isError=True)` (`ToolError`),
  instead of returning a text block prefixed with `"Error executing ..."` in a successful response, as the older low-level API did.
  Clients that only parse `content[0].text` will usually not notice any difference; integrations that rely on the `isError` flag will behave more in line with the MCP specification.

## [1.1.0] - 2026-07-30

### Fixed
- Fixed the server startup crash under MCP Python SDK 2.x (`AttributeError: 'Server' object has no attribute 'list_tools'`).
  SDK 2.0 removed the decorator API of the low-level `Server` (`@app.list_tools()` / `@app.call_tool()` / `app.get_capabilities()`),
  while launching the published package through `uvx` resolves to the latest 2.x, which caused the connection to be closed.

### Changed
- Migrated the MCP server implementation from the low-level `Server` API to the high-level `MCPServer` API (mcp 2.x, previously named FastMCP).
  - The 28 tools were rewritten in the `@mcp.tool()` function-signature style: input parameters are declared through type annotations (the schema is generated automatically by the framework),
    descriptions come from the docstring, and plain Python return values are serialised by the framework.
  - The transport layer now uses `run_stdio_async()` / `sse_app()` / `streamable_http_app()`, with authentication still wrapped by `MCPAuthMiddleware`.
  - The `EnterpriseRagClient` business logic (REST/SSE calls, resolve_*, wiki) is unchanged.
- The dependency upper bound was adjusted to `mcp>=2,<3` (the published package and the development environment now match, so an unbounded requirement can no longer resolve to a future breaking version).

### Note
- This version requires `mcp>=2` in the runtime environment. If you temporarily need the older SDK, you can add `--with "mcp<2"` to the startup command, but upgrading to this version is recommended.

## [1.0.1] - 2026-07-28

### Fixed
- Moved the diagnostic output of the entry scripts (`run_server.py`, `main.py`, `run.py`) to stderr, so that the MCP stdio protocol stream is not corrupted and clients no longer fail to start
- Fixed the `ModuleNotFoundError` caused by `upload_paths.py` missing from the wheel
- Changed `__init__.py` to use absolute imports, fixing the package import errors seen when unittest/pytest collects tests

### Changed
- The PyPI distribution name is now uniformly `enterpriserag-mcp` (the command-line entry points `enterpriserag-mcp-server` / `enterpriserag-server` are unchanged)
- Added a CI workflow (`.github/workflows/mcp-server.yml`); release tags use the `mcp-server-v*` format

## [1.0.0] - 2024-01-XX

### Added
- Initial release
- Core EnterpriseRag MCP Server functionality
- Complete EnterpriseRag API integration
- Space management tools
- Knowledge base management tools
- Knowledge management tools
- Model management tools
- Session management tools
- Chat tools
- Chunk management tools
- Support for several startup methods
- Command-line argument support
- Environment variable configuration
- Complete package installation support
- Development and production modes
- Detailed documentation and installation guide

### Tool list
- `create_tenant` - Create a new space
- `list_tenants` - List all spaces
- `create_knowledge_base` - Create a knowledge base
- `list_knowledge_bases` - List knowledge bases
- `get_knowledge_base` - Get knowledge base details
- `delete_knowledge_base` - Delete a knowledge base
- `hybrid_search` - Hybrid search
- `create_knowledge_from_url` - Create knowledge from a URL
- `list_knowledge` - List knowledge
- `get_knowledge` - Get knowledge details
- `delete_knowledge` - Delete knowledge
- `create_model` - Create a model
- `list_models` - List models
- `get_model` - Get model details
- `create_session` - Create a chat session
- `get_session` - Get session details
- `list_sessions` - List sessions
- `delete_session` - Delete a session
- `chat` - Send a chat message
- `list_chunks` - List knowledge chunks
- `delete_chunk` - Delete a knowledge chunk

### File structure
```
EnterpriseRag/mcp-server/
├── __init__.py              # Package initialisation file
├── main.py                  # Main entry point (recommended)
├── run.py                   # Convenience startup script
├── run_server.py           # Original startup script
├── enterpriserag_mcp_server.py   # MCP server implementation
├── test_module.py          # Module test script
├── requirements.txt        # Dependency list
├── setup.py               # Installation script (traditional)
├── pyproject.toml         # Modern project configuration
├── MANIFEST.in            # Packaged-file manifest
├── LICENSE                # MIT licence
├── README.md              # Project overview
├── INSTALL.md             # Detailed installation guide
└── CHANGELOG.md           # Changelog
```

### Ways to start
1. `python main.py` - Main entry point (recommended)
2. `python run_server.py` - Original startup script
3. `python run.py` - Convenience startup script
4. `python enterpriserag_mcp_server.py` - Run directly
5. `python -m enterpriserag_mcp_server` - Run as a module
6. `enterpriserag-mcp-server` - Command-line tool after installation
7. `enterpriserag-server` - Command-line tool after installation (alias)

### Technical features
- Based on Model Context Protocol (MCP) 1.0.0+
- Asynchronous I/O support
- Comprehensive error handling
- Detailed logging
- Environment variable configuration
- Command-line argument support
- Several installation methods
- Development and production modes
- Complete test coverage

### Dependencies
- Python 3.10+
- mcp >= 1.0.0
- requests >= 2.31.0

### Compatibility
- Supports Windows, macOS and Linux
- Supports Python 3.10-3.12
- Compatible with modern Python package management tools
