import { z } from "zod";

import { getSkillRoots, getTemplateOutputRoots } from "../lib/config.js";
import { listSkills, readSkillFile, searchSkills } from "../lib/skills.js";
import { createToolTemplate } from "../lib/templates.js";
import type { ToolRegistrar } from "./types.js";

export const registerSkillTools: ToolRegistrar = (server) => {
  server.registerTool(
    "skills_list",
    {
      title: "List Skills",
      description: "List discovered personal skills from configured skill roots.",
      inputSchema: {
        limit: z.number().int().positive().max(500).optional()
      }
    },
    async ({ limit }) => {
      const roots = await getSkillRoots();
      const skills = await listSkills(roots);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(limit ? skills.slice(0, limit) : skills, null, 2)
          }
        ]
      };
    }
  );

  server.registerTool(
    "skills_read",
    {
      title: "Read Skill",
      description: "Read a SKILL.md file after validating it is under a configured skill root.",
      inputSchema: {
        path: z.string().min(1).describe("Absolute or home-relative path to a SKILL.md file")
      }
    },
    async ({ path }) => {
      const roots = await getSkillRoots();
      const text = await readSkillFile(path, roots);
      return { content: [{ type: "text", text }] };
    }
  );

  server.registerTool(
    "skills_search",
    {
      title: "Search Skills",
      description: "Search skill names, descriptions, and SKILL.md content.",
      inputSchema: {
        query: z.string().min(1),
        limit: z.number().int().positive().max(100).optional()
      }
    },
    async ({ query, limit }) => {
      const roots = await getSkillRoots();
      const results = await searchSkills(query, roots, limit);
      return { content: [{ type: "text", text: JSON.stringify(results, null, 2) }] };
    }
  );

  server.registerTool(
    "tool_template_create",
    {
      title: "Create Tool Template",
      description: "Create a starter TypeScript MCP tool module in an allowed output root.",
      inputSchema: {
        name: z.string().min(1).describe("snake_case tool name"),
        description: z.string().min(1),
        outputDir: z.string().min(1).describe("Directory where the tool .ts file should be written")
      }
    },
    async ({ name, description, outputDir }) => {
      const allowedRoots = await getTemplateOutputRoots();
      const result = await createToolTemplate({ name, description }, allowedRoots, outputDir);
      return { content: [{ type: "text", text: JSON.stringify(result, null, 2) }] };
    }
  );
};
