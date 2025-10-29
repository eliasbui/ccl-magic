# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the application
task build

# Run with CLI arguments
task run -- --help

# Install locally
task install

# Development mode with profiling enabled
task dev
```

### Testing and Quality
```bash
# Run all tests
task test

# Run tests with specific flags
task test -- -v -race

# Run linter
task lint

# Fix linting issues automatically
task lint:fix

# Format code
task fmt
```

### Schema Generation
```bash
# Generate JSON schema for configuration
task schema
```

## High-Level Architecture

CCL-MAGIC is a terminal-based AI assistant built in Go using a layered architecture:

### Core Components
- **Application Layer** (`internal/app/`) - Main orchestrator coordinating all subsystems
- **Agent System** (`internal/agent/`) - AI conversation engine using the `fantasy` framework
- **Terminal UI** (`internal/tui/`) - Bubble Tea-based user interface with component architecture
- **Configuration System** (`internal/config/`) - Hierarchical configuration management
- **Data Layer** (`internal/db/`) - SQLite persistence with migration system
- **Tool System** (`internal/agent/tools/`) - Extensible framework for tool execution
- **External Integrations** - LSP client and MCP (Model Context Protocol) support

### Key Architectural Patterns
- **Service Layer Pattern** - Each major functionality encapsulated as a service
- **Event-Driven Architecture** - Pub/sub pattern for loose coupling via central event bus
- **Tool Plugin System** - Extensible tools implementing `fantasy.AgentTool` interface
- **Provider Abstraction** - Unified interface for multiple LLM providers

### Agent System
The agent system (`internal/agent/`) is the core AI engine:
- **Coordinators** manage agent lifecycle and tool building
- **Session Agents** handle individual conversations with streaming support
- **Multi-provider support** for OpenAI, Anthropic, Google, Azure, Bedrock, OpenRouter
- **Context management** with conversation history and automatic summarization
- **Tool integration** with permission-based execution

### Tool Framework
Located in `internal/agent/tools/`, the tool system provides:
- **File Operations** - `edit`, `view`, `write`, `glob`, `grep`, `ls`
- **System Integration** - `bash`, `download`, `fetch`
- **LSP Integration** - Diagnostics, references, code analysis
- **MCP Support** - External tool integration via Model Context Protocol

### TUI Architecture
The terminal UI (`internal/tui/`) uses a component-based approach:
- **Components** - Reusable UI elements (chat, dialogs, completions)
- **Pages** - Main application pages (chat is primary)
- **Dialogs** - Modal dialogs for commands, permissions, settings
- **Navigation** - Keyboard-driven with comprehensive key bindings

### Configuration System
Configuration follows this priority order:
1. `.ccl-magic.json` (project root)
2. `ccl-magic.json` (project root)
3. `$HOME/.config/ccl-magic/ccl-magic.json` (global)

The system supports:
- **Environment variable expansion** using `$(echo $VAR)` syntax
- **Provider-specific settings** for different LLM providers
- **Agent definitions** with per-agent tool and model assignments
- **Context paths** for project-specific context files

## Important Implementation Details

### Database Schema
- Uses SQLite with migrations in `internal/db/migrations/`
- Key tables: sessions, messages, files
- Automatic schema versioning and migration system

### Permission System
- Granular tool-level permissions
- Session-based permission persistence
- YOLO mode for automated workflows
- Directory-based permission boundaries

### LSP Integration
- Multi-language support with automatic file type detection
- Root marker detection for project identification
- Code intelligence features: diagnostics, completions, hover info

### MCP Integration
- Three transport types: stdio, HTTP, SSE
- Dynamic tool discovery from MCP servers
- Environment variable expansion for configuration

## Environment Variables
Key environment variables for development:
- `CCL_MAGIC_PROFILE` - Enable pprof profiling
- `CCL_MAGIC_DISABLE_METRICS` - Opt out of metrics collection
- `CCL_MAGIC_DISABLE_PROVIDER_AUTO_UPDATE` - Disable automatic provider updates
- Provider-specific API keys (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)

## Testing Strategy
- Unit tests located alongside source files
- Integration tests in `testdata/` directories
- Use `task test` to run the full test suite
- Test files follow Go conventions with `_test.go` suffix

## Configuration Examples
When working with configuration, refer to the existing JSON schema (`schema.json`) and example configurations in the repository for proper structure and available options.