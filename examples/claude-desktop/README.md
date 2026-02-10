# ⚠️ Important Notice

**This version of loki-mcp uses HTTP transport and is NOT compatible with Claude Desktop.**

These configuration examples are from the original repository and are kept for reference only.

## For Claude Desktop Users

If you need Claude Desktop compatibility, please use the original repository:
- **Original Repository:** https://github.com/scottlepp/loki-mcp
- **Communication Method:** stdin/stdout (subprocess spawning)
- **MCP Library:** mark3labs/mcp-go

## This Version (AWS Bedrock AgentCore)

This modified version is designed for:
- **AWS Bedrock AgentCore** deployment
- **HTTP-based** MCP communication
- **Stateless** session management
- **Port 8000** with `/mcp` endpoint

See [DEPLOYMENT_GUIDE.md](../../DEPLOYMENT_GUIDE.md) for deployment instructions.
