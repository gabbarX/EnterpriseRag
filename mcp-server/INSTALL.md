# EnterpriseRag MCP Server Installation and Usage Guide

## Quick Start

### 1. Install dependencies
```bash
pip install -r requirements.txt
```

### 2. Set environment variables
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

There are several ways to run the server:

#### Option 1: Use the main entry point (recommended)
```bash
python main.py
```

#### Option 2: Use the original startup script
```bash
python run_server.py
```

#### Option 3: Run the server module directly
```bash
python enterpriserag_mcp_server.py
```

#### Option 4: Run as a Python module
```bash
python -m enterpriserag_mcp_server
```

## Installing as a Python Package

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
# Build the source distribution and the wheel
python setup.py sdist bdist_wheel

# Or use the build tool
pip install build
python -m build
```

## Command Line Options

The main entry point `main.py` supports the following options:

```bash
python main.py --help                 # Show help information
python main.py --check-only           # Only check the environment configuration
python main.py --verbose              # Enable verbose logging
python main.py --version              # Show version information
```

## Environment Check

Run the following command to check the environment configuration:
```bash
python main.py --check-only
```

This will show:
- The EnterpriseRag API base URL configuration
- The API key setup status
- The installation status of the dependencies

## Troubleshooting

### 1. Import errors
If you hit an `ImportError`, please make sure that:
- All dependencies are installed: `pip install -r requirements.txt`
- Your Python version is compatible (3.10+ recommended)
- There are no file name conflicts

### 2. Connection errors
If you cannot connect to the EnterpriseRag API:
- Check whether `ENTERPRISERAG_BASE_URL` is correct
- Confirm that the EnterpriseRag service is running
- Verify the network connectivity

### 3. Authentication errors
If you run into authentication problems:
- Check whether `ENTERPRISERAG_API_KEY` is set
- Confirm that the API key is valid
- Verify the permission settings

## Development Mode

### Project structure
```
EnterpriseRag/mcp-server/
├── __init__.py              # Package initialisation file
├── main.py                  # Main entry point
├── run_server.py           # Original startup script
├── enterpriserag_mcp_server.py   # MCP server implementation
├── requirements.txt        # Dependency list
├── setup.py               # Installation script
├── pyproject.toml         # Project metadata (PyPI: PYPI_NAME_PLACEHOLDER)
├── MANIFEST.in            # Included files manifest
├── LICENSE                # Licence
├── README.md              # Project overview
└── INSTALL.md             # Installation guide
```

### Adding new features
1. Add a new API method to the `EnterpriseRagClient` class
2. Register a new tool function with the `@mcp.tool()` decorator: annotate the parameters with types (the schema is generated automatically), put the description in the docstring, and call the newly added client method from the function body
3. Update the documentation and the tests

### Testing
```bash
# Run the basic tests
python check_imports.py

# Test the environment configuration
python main.py --check-only

# Test the server startup
python main.py --verbose
```

## Deployment

### Docker deployment
Create a `Dockerfile`:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN pip install -e .

ENV ENTERPRISERAG_BASE_URL=http://localhost:8080/api/v1
EXPOSE 8000

CMD ["enterpriserag-mcp-server"]
```

### System service
Create the systemd service file `/etc/systemd/system/enterpriserag-mcp.service`:
```ini
[Unit]
Description=EnterpriseRag MCP Server
After=network.target

[Service]
Type=simple
User=enterpriserag
WorkingDirectory=/opt/enterpriserag-mcp
Environment=ENTERPRISERAG_BASE_URL=http://localhost:8080/api/v1
Environment=ENTERPRISERAG_API_KEY=your_api_key
ExecStart=/usr/local/bin/enterpriserag-mcp-server
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable the service:
```bash
sudo systemctl enable enterpriserag-mcp
sudo systemctl start enterpriserag-mcp
```

## Support

If you run into problems, please:
1. Check the log output
2. Check the environment configuration
3. Refer to the troubleshooting section
4. Raise an issue on the project repository: https://github.com/ORG_PLACEHOLDER/EnterpriseRag/issues