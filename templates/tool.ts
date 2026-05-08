import { z } from "zod";

import type { ToolRegistrar } from "./types.js";

const inputSchema = {
  message: z.string().min(1).describe("Input text for this tool")
};

export const registerExampleTool: ToolRegistrar = (server) => {
  server.registerTool(
    "example_tool",
    {
      title: "Example Tool",
      description: "Describe what this tool does",
      inputSchema
    },
    async ({ message }) => ({
      content: [{ type: "text", text: message }]
    })
  );
};
