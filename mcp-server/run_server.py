#!/usr/bin/env python3
"""
EnterpriseRag MCP Server launch script.

Note: under the stdio transport, stdout is the JSON-RPC channel, so every
diagnostic or informational message must be written to stderr. Writing to
stdout would corrupt the MCP protocol stream and the client would report a
startup failure. Every print in this script therefore goes to stderr.
"""

import asyncio
import os
import sys


def check_environment():
    """Print the environment configuration and warn about missing values."""
    base_url = os.getenv("ENTERPRISERAG_BASE_URL")
    api_key = os.getenv("ENTERPRISERAG_API_KEY")

    if not base_url:
        print(
            "Warning: ENTERPRISERAG_BASE_URL is not set; using the default: http://localhost:8080/api/v1",
            file=sys.stderr,
        )

    if not api_key:
        print("Warning: ENTERPRISERAG_API_KEY is not set", file=sys.stderr)

    print(f"EnterpriseRag Base URL: {base_url or 'http://localhost:8080/api/v1'}", file=sys.stderr)
    print(f"API Key: {'set' if api_key else 'not set'}", file=sys.stderr)


def main():
    print("Starting EnterpriseRag MCP Server...", file=sys.stderr)
    check_environment()

    try:
        from enterpriserag_mcp_server import run

        asyncio.run(run())
    except ImportError as e:
        print(f"Import error: {e}", file=sys.stderr)
        print("Please install all dependencies: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped", file=sys.stderr)
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
