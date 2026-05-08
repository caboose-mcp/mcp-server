import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";

import { registerSkillTools } from "./tools/skills.js";

export function createServer(): McpServer {
  const server = new McpServer({
    name: "caboose-mcp-server",
    version: "0.1.0"
  });

  registerSkillTools(server);

  return server;
}
