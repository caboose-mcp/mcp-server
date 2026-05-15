# mcp-server

TypeScript MCP server for personal but reusable agent tooling. The first version is local-first over stdio and exposes safe tools for discovering personal skills and creating new MCP tool templates.

## Requirements

- [mise](https://mise.jdx.dev/)
- Bun, installed through `mise install`

## Setup

```sh
mise trust
mise install
bun install
```

## Commands

```sh
mise run dev        # run stdio MCP server
mise run test       # run unit tests
mise run typecheck  # TypeScript type check
mise run lint       # ESLint + Prettier checks
mise run build      # compile to dist/
```

## MCP Client Config

Use the compiled binary for stable local clients:

```sh
mise run build
```

```json
{
  "mcpServers": {
    "caboose-mcp-server": {
      "command": "node",
      "args": ["/absolute/path/to/mcp-server/dist/index.js"]
    }
  }
}
```

For local development:

```json
{
  "mcpServers": {
    "caboose-mcp-server": {
      "command": "bun",
      "args": ["run", "/absolute/path/to/mcp-server/src/index.ts"]
    }
  }
}
```

## Skill Roots

The server indexes `SKILL.md` files under these default roots when they exist:

- `$CODEX_HOME/skills`
- `~/.codex/skills`
- `~/.agents/skills`

Add more roots with `MCP_SKILL_ROOTS` using your OS path delimiter, for example:

```sh
MCP_SKILL_ROOTS="/home/caboose/dev/ai-skills:/other/skills" mise run dev
```

Template creation is constrained to the current working directory by default. Add allowed output roots with `MCP_TEMPLATE_OUTPUT_ROOTS`.

## Tools

| Tool                   | Purpose                                                                |
| ---------------------- | ---------------------------------------------------------------------- |
| `skills_list`          | List discovered skills with name, description, path, and source root.  |
| `skills_read`          | Read a skill's `SKILL.md` after root containment checks.               |
| `skills_search`        | Search discovered skill metadata and `SKILL.md` content.               |
| `tool_template_create` | Create a starter TypeScript MCP tool module in an allowed output root. |

## Next Milestone

Add repo-crawler tools by wrapping or porting the existing `repo-agent-guidance-generator` inventory flow.
