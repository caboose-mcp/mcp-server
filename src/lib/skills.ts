import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

import { assertContainedPath, existingDirectory, isSecretLike, normalizePath } from "./paths.js";

export interface SkillRoot {
  path: string;
  label: string;
}

export interface SkillRecord {
  name: string;
  description: string;
  path: string;
  directory: string;
  root: string;
  rootLabel: string;
}

export interface SkillSearchResult {
  skill: SkillRecord;
  excerpt: string;
}

interface Frontmatter {
  body: string;
  fields: Record<string, string>;
}

const SKIP_DIRS = new Set([
  ".git",
  ".hg",
  ".svn",
  ".cache",
  ".next",
  ".nuxt",
  ".pytest_cache",
  ".ruff_cache",
  ".tox",
  ".venv",
  "venv",
  "env",
  "node_modules",
  "vendor",
  "dist",
  "build",
  "target",
  "coverage",
  "__pycache__"
]);

export async function listSkills(roots: SkillRoot[]): Promise<SkillRecord[]> {
  const records: SkillRecord[] = [];

  for (const root of roots) {
    const existingRoot = await existingDirectory(root.path);
    if (!existingRoot) {
      continue;
    }
    for await (const skillPath of walkSkillFiles(existingRoot)) {
      const text = await readFile(skillPath, "utf8");
      const frontmatter = parseFrontmatter(text);
      records.push({
        name: frontmatter.fields.name || path.basename(path.dirname(skillPath)),
        description: frontmatter.fields.description || "",
        path: skillPath,
        directory: path.dirname(skillPath),
        root: existingRoot,
        rootLabel: root.label
      });
    }
  }

  return records.sort((a, b) => a.name.localeCompare(b.name) || a.path.localeCompare(b.path));
}

export async function readSkillFile(skillFile: string, roots: SkillRoot[]): Promise<string> {
  if (path.basename(skillFile) !== "SKILL.md") {
    throw new Error("Only SKILL.md files can be read");
  }

  const allowedRoots = await resolveRoots(roots);
  if (allowedRoots.length === 0) {
    throw new Error("No allowed skill roots exist");
  }

  const realSkillPath = await assertContainedPath(
    normalizePath(skillFile),
    allowedRoots,
    "skill root"
  );
  if (isSecretLike(realSkillPath)) {
    throw new Error(`${skillFile} is a secret-like path and will not be read`);
  }

  return readFile(realSkillPath, "utf8");
}

export async function searchSkills(
  query: string,
  roots: SkillRoot[],
  limit = 20
): Promise<SkillSearchResult[]> {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    throw new Error("Query is required");
  }

  const results: SkillSearchResult[] = [];

  for (const root of roots) {
    const existingRoot = await existingDirectory(root.path);
    if (!existingRoot) {
      continue;
    }
    for await (const skillPath of walkSkillFiles(existingRoot)) {
      const content = await readFile(skillPath, "utf8");
      const frontmatter = parseFrontmatter(content);
      const name = frontmatter.fields.name || path.basename(path.dirname(skillPath));
      const description = frontmatter.fields.description || "";
      const haystack = `${name}\n${description}\n${content}`.toLowerCase();
      if (haystack.indexOf(needle) === -1) {
        continue;
      }
      results.push({
        skill: {
          name,
          description,
          path: skillPath,
          directory: path.dirname(skillPath),
          root: existingRoot,
          rootLabel: root.label
        },
        excerpt: makeExcerpt(content, needle)
      });
      if (results.length >= limit) {
        return results;
      }
    }
  }

  return results;
}

async function resolveRoots(roots: SkillRoot[]): Promise<string[]> {
  const resolved = await Promise.all(roots.map((root) => existingDirectory(root.path)));
  return resolved.filter((root): root is string => Boolean(root));
}

async function* walkSkillFiles(dir: string): AsyncGenerator<string> {
  if (isSecretLike(dir)) {
    return;
  }

  const entries = await readdir(dir, { withFileTypes: true });
  if (entries.some((entry) => entry.isFile() && entry.name === "SKILL.md")) {
    yield path.join(dir, "SKILL.md");
    return;
  }

  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue;
    }
    const child = path.join(dir, entry.name);
    if (SKIP_DIRS.has(entry.name) || isSecretLike(child)) {
      continue;
    }
    yield* walkSkillFiles(child);
  }
}

function parseFrontmatter(text: string): Frontmatter {
  if (!text.startsWith("---\n")) {
    return { fields: {}, body: text };
  }

  const closeIndex = text.indexOf("\n---", 4);
  if (closeIndex === -1) {
    return { fields: {}, body: text };
  }

  const raw = text.slice(4, closeIndex);
  const fields: Record<string, string> = {};
  for (const line of raw.split("\n")) {
    const match = /^([A-Za-z0-9_-]+):\s*(.*)$/.exec(line);
    if (!match) {
      continue;
    }
    fields[match[1] ?? ""] = (match[2] ?? "").replace(/^["']|["']$/g, "").trim();
  }

  return { fields, body: text.slice(closeIndex + 4) };
}

function makeExcerpt(content: string, needle: string): string {
  const lowered = content.toLowerCase();
  const index = lowered.indexOf(needle);
  if (index === -1) {
    return "";
  }
  const start = Math.max(0, index - 60);
  const end = Math.min(content.length, index + needle.length + 100);
  return content.slice(start, end).replace(/\s+/g, " ").trim();
}
