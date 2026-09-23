#!/usr/bin/env python3
"""
EnterpriseRag MCP Server convenience launcher.

A simplified launch script that provides only the basics. For more options,
use main.py instead.

Note: under the stdio transport, stdout is the JSON-RPC channel, so every
diagnostic or informational message must be written to stderr. Writing to
stdout would corrupt the MCP protocol stream and the client would report a
startup failure. Every print in this script therefore goes to stderr.
"""

import os
import sys
from pathlib import Path


def main():
    """Start the server with the default configuration."""
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))

    base_url = os.getenv("ENTERPRISERAG_BASE_URL", "http://localhost:8080/api/v1")
    api_key = os.getenv("ENTERPRISERAG_API_KEY", "")

    print("EnterpriseRag MCP Server", file=sys.stderr)
    print(f"Base URL: {base_url}", file=sys.stderr)
    print(f"API Key: {'set' if api_key else 'not set'}", file=sys.stderr)
    print("-" * 40, file=sys.stderr)

    try:
        from main import sync_main

        sync_main()
    except ImportError:
        print("Error: could not import the required modules", file=sys.stderr)
        print("Please run: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped", file=sys.stderr)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
