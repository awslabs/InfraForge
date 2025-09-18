# MCP Configuration Migration Guide

## Changes Made

### 1. Removed Old Configuration
- Deleted `cmd/infraforge/.amazonq/` directory (old MCP configuration location)

### 2. Updated Documentation

#### README.md & README.zh-CN.md
- Added MCP server registration step: `q mcp add --force --name infraforge --command infraforge_mcp_server --timeout 7200000`
- Updated tool naming format from `infraforge___toolName` to `@infraforge/toolName`
- Updated Q Chat command with new tool names

#### User Guides (docs/user-guide.md & docs/user-guide.zh-CN.md)
- Added MCP server registration step in setup instructions
- Updated tool naming format in tables and commands
- Updated Q Chat command with new tool names

### 3. New MCP Configuration Format

#### Old Format (deprecated):
```bash
# Old location: cmd/infraforge/.amazonq/mcp.json
# Old tool names: infraforge___getDeploymentStatus
# Old command: q chat --trust-tools=infraforge___getDeploymentStatus,...
```

#### New Format:
```bash
# New registration: q mcp add --force --name infraforge --command infraforge_mcp_server --timeout 7200000
# New tool names: @infraforge/getDeploymentStatus
# New command: q chat --trust-tools=@infraforge/getDeploymentStatus,@infraforge/getStackOutputs,@infraforge/getOperationManual,@infraforge/listTemplates
```

## Updated Commands

### Setup MCP Server:
```bash
cd tools/mcp/
go build
sudo cp infraforge_mcp_server /usr/local/bin/
sudo chmod +x /usr/local/bin/infraforge_mcp_server
q mcp add --force --name infraforge --command infraforge_mcp_server --timeout 7200000
```

### Start Q Chat:
```bash
q chat --trust-tools=@infraforge/getDeploymentStatus,@infraforge/getStackOutputs,@infraforge/getOperationManual,@infraforge/listTemplates
```

## Tool Name Mapping

| Old Name | New Name |
|----------|----------|
| `infraforge___deployInfra` | `@infraforge/deployInfra` |
| `infraforge___getDeploymentStatus` | `@infraforge/getDeploymentStatus` |
| `infraforge___getOperationManual` | `@infraforge/getOperationManual` |
| `infraforge___getStackOutputs` | `@infraforge/getStackOutputs` |
| `infraforge___listTemplates` | `@infraforge/listTemplates` |

## Migration Complete

All documentation has been updated to reflect the new MCP configuration format and tool naming conventions.
