import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { createToolTemplate } from "../src/lib/templates.js";

describe("tool template creation", () => {
  it("creates a typed MCP tool template inside an allowed output root", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "mcp-template-"));
    const outputDir = path.join(root, "src", "tools");

    const result = await createToolTemplate(
      {
        name: "repo_summary",
        description: "Summarize a repository"
      },
      [root],
      outputDir
    );

    const text = await readFile(result.path, "utf8");
    expect(result.path).toBe(path.join(outputDir, "repo_summary.ts"));
    expect(text).toContain("registerRepoSummaryTool");
    expect(text).toContain("Summarize a repository");
  });

  it("scaffolds types.ts alongside the new tool when it does not exist", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "mcp-template-"));
    const outputDir = path.join(root, "src", "tools");

    await createToolTemplate({ name: "my_tool", description: "A tool" }, [root], outputDir);

    const typesText = await readFile(path.join(outputDir, "types.ts"), "utf8");
    expect(typesText).toContain("ToolRegistrar");
  });

  it("does not overwrite an existing types.ts", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "mcp-template-"));
    const outputDir = root;
    await mkdir(outputDir, { recursive: true });
    const typesPath = path.join(outputDir, "types.ts");
    const original = "// existing types\n";
    await writeFile(typesPath, original);

    await createToolTemplate(
      { name: "another_tool", description: "Another tool" },
      [root],
      outputDir
    );

    const typesText = await readFile(typesPath, "utf8");
    expect(typesText).toBe(original);
  });

  it("rejects output paths outside allowed roots", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "mcp-template-"));
    const outside = await mkdtemp(path.join(tmpdir(), "mcp-outside-"));
    await mkdir(outside, { recursive: true });

    await expect(
      createToolTemplate({ name: "bad_tool", description: "Bad" }, [root], outside)
    ).rejects.toThrow("not under an allowed output root");
  });

  it("rejects invalid tool names", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "mcp-template-"));

    await expect(
      createToolTemplate({ name: "../bad", description: "Bad" }, [root], root)
    ).rejects.toThrow("Tool names");
  });
});
