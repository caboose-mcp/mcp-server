import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import { describe, expect, it } from "vitest";

import { createServer } from "../src/server.js";

describe("MCP server", () => {
  it("registers the day-one skill tools", async () => {
    const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();
    const server = createServer();
    const client = new Client({ name: "test-client", version: "0.1.0" });

    await Promise.all([server.connect(serverTransport), client.connect(clientTransport)]);
    try {
      const result = await client.listTools();
      expect(result.tools.map((tool) => tool.name).sort()).toEqual([
        "skills_list",
        "skills_read",
        "skills_search",
        "tool_template_create"
      ]);
    } finally {
      await client.close();
      await server.close();
    }
  });
});
