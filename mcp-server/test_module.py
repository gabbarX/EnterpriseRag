#!/usr/bin/env python3
"""
EnterpriseRag MCP Server module test script.

Tests the various ways the module can be started, and its behaviour.
unittest discover picks up the TestCases in this file; it can also be run
directly with: python test_module.py
"""

import os
import subprocess
import sys
import unittest
from pathlib import Path

MCP_SERVER_DIR = Path(__file__).resolve().parent

REQUIRED_FILES = [
    "__init__.py",
    "main.py",
    "run_server.py",
    "enterpriserag_mcp_server.py",
    "requirements.txt",
    "setup.py",
    "pyproject.toml",
    "README.md",
    "INSTALL.md",
    "LICENSE",
    "MANIFEST.in",
]


class ModuleIntegrationTest(unittest.TestCase):
    def test_imports(self):
        import mcp  # noqa: F401
        import requests  # noqa: F401
        import enterpriserag_mcp_server  # noqa: F401
        from enterpriserag_mcp_server import EnterpriseRagClient, run  # noqa: F401
        import main  # noqa: F401

    def test_environment_optional_vars(self):
        os.getenv("ENTERPRISERAG_BASE_URL")
        os.getenv("ENTERPRISERAG_API_KEY")

    def test_client_creation(self):
        from enterpriserag_mcp_server import EnterpriseRagClient

        base_url = os.getenv("ENTERPRISERAG_BASE_URL", "http://localhost:8080/api/v1")
        api_key = os.getenv("ENTERPRISERAG_API_KEY", "test_key")
        client = EnterpriseRagClient(base_url, api_key)
        self.assertEqual(client.base_url, base_url)
        self.assertEqual(client.api_key, api_key)

    def test_required_files_exist(self):
        missing = [name for name in REQUIRED_FILES if not (MCP_SERVER_DIR / name).exists()]
        self.assertEqual(missing, [], f"Missing files: {missing}")

    def test_main_help(self):
        result = subprocess.run(
            [sys.executable, "main.py", "--help"],
            capture_output=True,
            text=True,
            timeout=10,
            cwd=MCP_SERVER_DIR,
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_main_check_only(self):
        result = subprocess.run(
            [sys.executable, "main.py", "--check-only"],
            capture_output=True,
            text=True,
            timeout=10,
            cwd=MCP_SERVER_DIR,
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_wiki_tools(self):
        import enterpriserag_mcp_server

        client = enterpriserag_mcp_server.EnterpriseRagClient("http://localhost:8080/api/v1", "test")
        for method in ["wiki_search", "wiki_read_page", "wiki_index_view"]:
            self.assertTrue(hasattr(client, method), f"EnterpriseRagClient missing: {method}")
            self.assertTrue(callable(getattr(client, method)), f"{method} not callable")

    def test_pyproject_metadata(self):
        text = (MCP_SERVER_DIR / "pyproject.toml").read_text(encoding="utf-8")
        self.assertIn("PYPI_NAME_PLACEHOLDER", text)
        self.assertIn("enterpriserag-mcp-server", text)


if __name__ == "__main__":
    unittest.main(verbosity=2)
