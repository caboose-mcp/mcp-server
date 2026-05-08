import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

import { assertOutputDirectory } from "./paths.js";

export interface CreateToolTemplateInput {
  name: string;
  description: string;
}

export interface CreatedToolTemplate {
  path: string;
  text: string;
}

const TOOL_NAME_PATTERN = /^[a-z][a-z0-9_]*$/;

export async function createToolTemplate(
  input: CreateToolTemplateInput,
  allowedOutputRoots: string[],
  outputDir: string
): Promise<CreatedToolTemplate> {
  const name = input.name.trim();
  if (!TOOL_NAME_PATTERN.test(name)) {
    throw new Error("Tool names must use snake_case and start with a lowercase letter");
  }
  if (!input.description.trim()) {
    throw new Error("Description is required");
  }
  if (allowedOutputRoots.length === 0) {
    throw new Error("No allowed output roots exist");
  }

  const safeOutputDir = await assertOutputDirectory(outputDir, allowedOutputRoots);
  await mkdir(safeOutputDir, { recursive: true });

  const target = path.join(safeOutputDir, `${name}.ts`);
  const text = renderToolTemplate(name, input.description.trim());
  await writeFile(target, text, { flag: "wx" });

  return { path: target, text };
}

function renderToolTemplate(name: string, description: string): string {
  const exportName = `register${toPascalCase(name)}Tool`;
  return `import { z } from "zod";

import type { ToolRegistrar } from "./types.js";

const inputSchema = {
  message: z.string().min(1).describe("Input text for this tool")
};

export const ${exportName}: ToolRegistrar = (server) => {
  server.registerTool(
    "${name}",
    {
      title: "${toTitle(name)}",
      description: ${JSON.stringify(description)},
      inputSchema
    },
    async ({ message }) => ({
      content: [{ type: "text", text: message }]
    })
  );
};
`;
}

function toPascalCase(name: string): string {
  return name
    .split("_")
    .map((part) => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
    .join("");
}

function toTitle(name: string): string {
  return name
    .split("_")
    .map((part) => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
    .join(" ");
}
