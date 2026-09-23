#!/usr/bin/env python3
"""
EnterpriseRag MCP Server installation script.
"""

from setuptools import setup


def read_readme():
    try:
        with open("README.md", "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        return "EnterpriseRag MCP Server - Model Context Protocol server for EnterpriseRag API"


def read_requirements():
    try:
        with open("requirements.txt", "r", encoding="utf-8") as f:
            return [
                line.strip() for line in f if line.strip() and not line.startswith("#")
            ]
    except FileNotFoundError:
        return [
            "mcp>=2,<3",
            "requests>=2.31.0",
            "starlette>=0.27.0",
            "uvicorn>=0.24.0",
        ]


setup(
    name="PYPI_NAME_PLACEHOLDER",
    version="1.1.1",
    author="EnterpriseRag Team",
    author_email="EMAIL_PLACEHOLDER",
    description="EnterpriseRag MCP Server - Model Context Protocol server for EnterpriseRag API",
    long_description=read_readme(),
    long_description_content_type="text/markdown",
    url="https://github.com/ORG_PLACEHOLDER/EnterpriseRag/tree/main/mcp-server",
    py_modules=["enterpriserag_mcp_server", "upload_paths", "main", "run_server", "run", "test_module"],
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Internet :: WWW/HTTP :: HTTP Servers",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
    ],
    python_requires=">=3.10",
    install_requires=read_requirements(),
    entry_points={
        "console_scripts": [
            "enterpriserag-mcp-server=main:sync_main",
            "enterpriserag-server=run_server:main",
        ],
    },
    include_package_data=True,
    data_files=[
        ("", ["README.md", "requirements.txt", "LICENSE"]),
    ],
    keywords="mcp model-context-protocol enterpriserag knowledge-management api-server",
)
