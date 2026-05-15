import { cwd, env } from "node:process";

import { existingDirectory, normalizePath, splitPathEnv } from "./paths.js";

export interface RootConfig {
  path: string;
  label: string;
}

export async function getSkillRoots(environment: NodeJS.ProcessEnv = env): Promise<RootConfig[]> {
  const defaultRoots = [
    environment.CODEX_HOME ? `${environment.CODEX_HOME}/skills` : undefined,
    "~/.codex/skills",
    "~/.agents/skills"
  ].filter((root): root is string => Boolean(root));

  const configuredRoots = splitPathEnv(environment.MCP_SKILL_ROOTS);
  const roots = [...defaultRoots, ...configuredRoots];
  const seen = new Set<string>();
  const resolved: RootConfig[] = [];

  for (const root of roots) {
    const existing = await existingDirectory(root);
    if (!existing || seen.has(existing)) {
      continue;
    }
    seen.add(existing);
    resolved.push({ path: existing, label: root });
  }

  return resolved;
}

export async function getTemplateOutputRoots(
  environment: NodeJS.ProcessEnv = env
): Promise<string[]> {
  const configuredRoots = splitPathEnv(environment.MCP_TEMPLATE_OUTPUT_ROOTS);
  const roots = configuredRoots.length > 0 ? configuredRoots : [cwd()];
  const resolved = await Promise.all(roots.map((root) => existingDirectory(root)));
  return resolved.filter((root): root is string => Boolean(root));
}

export function defaultTemplateOutputDir(): string {
  return normalizePath("src/tools");
}
