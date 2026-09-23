#!/usr/bin/env python3
"""
EnterpriseRag MCP Server Package

A Model Context Protocol server that provides access to the EnterpriseRag knowledge management API.
"""

__version__ = "1.1.1"
__author__ = "EnterpriseRag Team"
__description__ = "EnterpriseRag MCP Server - Model Context Protocol server for EnterpriseRag API"

from enterpriserag_mcp_server import EnterpriseRagClient, run

__all__ = ["EnterpriseRagClient", "run"]
