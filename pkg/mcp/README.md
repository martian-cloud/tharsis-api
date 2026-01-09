# MCP Server Package

Reusable infrastructure for building Model Context Protocol (MCP) servers in Tharsis.

## What is MCP?

The Model Context Protocol (MCP) is an open protocol that standardizes how applications provide context to Large Language Models (LLMs). It enables communication between clients (like AI assistants) and servers that expose tools, resources, and prompts.

## Architecture

The package provides a layered approach to building MCP servers, inspired by GitHub's MCP server architecture:

1. **ServerConfig** - Configuration for server metadata, instructions, and tool selection
2. **ToolsetGroup** - Collection of related toolsets that can be enabled/disabled
3. **Toolset** - Named group of related tools, resources, and prompts
4. **Tools** - Individual MCP tools for performing actions
5. **Resources** - Read-only data that LLMs can access as context
6. **Resource Templates** - Parameterized resources using URI templates
7. **Prompts** - Reusable prompt templates

## Instructions

Instructions are server-level guidance provided to the LLM client about how to use the server's capabilities. They appear in the client's context and help the LLM understand:

- What the server does and when to use it
- How to effectively use the available tools and resources
- Best practices and workflows
- Any limitations or requirements

Instructions are set in `ServerConfig.Instructions` and should be concise but informative, guiding the LLM to make effective use of the exposed functionality.

## Key Concepts

### Tools vs Resources

**Tools** are for performing actions:
- Execute operations that may have side effects
- Examples: `create_workspace`, `trigger_run`, `upload_configuration`
- The LLM calls tools to accomplish tasks

**Resources** are for reading data:
- Expose read-only data that LLMs can access as context
- Examples: configuration files, logs, documentation
- The LLM reads resources to gather information before making decisions

### Resources vs Resource Templates

**Resources** - Use for static, enumerable resources:
- Fixed URI (e.g., `config://settings`, `docs://readme`)
- Small, known set of resources
- Example: Server configuration, system status

**Resource Templates** - Use for dynamic, parameterized resources:
- URI template with parameters (e.g., `tharsis://workspace/{workspace}/file/{path}`)
- Large or dynamic set of resources following a pattern
- The LLM constructs specific URIs by filling in parameters
- Example: Reading any file in a workspace, accessing logs for any job

## Key Features

### Flexible Tool Selection

- Enable entire toolsets: `EnabledToolsets: "job,run,workspace"`
- Enable specific tools: `EnabledTools: "get_job,get_run"` (overrides toolsets)
- Read-only mode: Skip write tools automatically

### Toolset Organization

Tools, resources, and prompts are organized into logical groups (toolsets) that can be enabled/disabled as a unit. This allows consumers to expose only the functionality they need.

### Validation

- Toolset names: 2-32 chars, lowercase letters/underscores, validated by regex `^[a-z][a-z_]{0,30}[a-z]$`
- Configuration: Required fields, instruction length limits
- Tool registration: Ensures at least one tool is enabled

## Usage Pattern

1. Define toolset metadata with name and description
2. Create tools, resources, and/or resource templates following MCP SDK patterns
3. Build toolset group by adding toolsets with their functionality
4. Create server with configuration and toolset group
5. Serve via stdio or SSE transport

## Example Implementation

See `internal/mcp/` for a complete implementation exposing Tharsis API functionality via MCP tools and resources.
