import { mkdtemp, mkdir, writeFile, symlink } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { listSkills, readSkillFile, searchSkills } from "../src/lib/skills.js";

async function makeRoot(): Promise<string> {
  return mkdtemp(path.join(tmpdir(), "mcp-skills-"));
}

async function writeSkill(root: string, relativeDir: string, body: string): Promise<string> {
  const dir = path.join(root, relativeDir);
  await mkdir(dir, { recursive: true });
  const skillFile = path.join(dir, "SKILL.md");
  await writeFile(skillFile, body);
  return skillFile;
}

describe("skill discovery", () => {
  it("lists SKILL.md files with frontmatter metadata", async () => {
    const root = await makeRoot();
    await writeSkill(
      root,
      "repo-agent-guidance-generator",
      `---
name: repo-agent-guidance-generator
description: Crawls repositories and proposes agent guidance.
---

# Repo Agent Guidance Generator
`
    );

    const skills = await listSkills([{ path: root, label: "fixture" }]);

    expect(skills).toEqual([
      expect.objectContaining({
        name: "repo-agent-guidance-generator",
        description: "Crawls repositories and proposes agent guidance.",
        rootLabel: "fixture"
      })
    ]);
  });

  it("skips secret-like paths during discovery", async () => {
    const root = await makeRoot();
    await writeSkill(root, "safe-skill", "---\nname: safe\n---\n# Safe\n");
    await writeSkill(root, "secret-skill", "---\nname: secret\n---\n# Secret\n");

    const skills = await listSkills([{ path: root, label: "fixture" }]);

    expect(skills.map((skill) => skill.name)).toEqual(["safe"]);
  });

  it("reads only SKILL.md files contained in configured roots", async () => {
    const root = await makeRoot();
    const outside = await makeRoot();
    const skillFile = await writeSkill(root, "safe-skill", "# Safe skill");
    const outsideFile = await writeSkill(outside, "outside-skill", "# Outside");

    await expect(readSkillFile(skillFile, [{ path: root, label: "fixture" }])).resolves.toBe(
      "# Safe skill"
    );
    await expect(readSkillFile(outsideFile, [{ path: root, label: "fixture" }])).rejects.toThrow(
      "not under an allowed skill root"
    );
  });

  it("rejects symlinks that resolve outside configured roots", async () => {
    const root = await makeRoot();
    const outside = await makeRoot();
    const outsideFile = await writeSkill(outside, "outside-skill", "# Outside");
    const link = path.join(root, "linked-skill");
    await symlink(path.dirname(outsideFile), link);

    await expect(
      readSkillFile(path.join(link, "SKILL.md"), [{ path: root, label: "fixture" }])
    ).rejects.toThrow("not under an allowed skill root");
  });

  it("searches metadata and content case-insensitively", async () => {
    const root = await makeRoot();
    await writeSkill(root, "repo-agent", "---\nname: repo-agent\n---\n# Repo crawler\n");
    await writeSkill(root, "other", "---\nname: other\n---\n# Email assistant\n");

    const results = await searchSkills("CRAWLER", [{ path: root, label: "fixture" }]);

    expect(results).toHaveLength(1);
    expect(results[0]?.skill.name).toBe("repo-agent");
    expect(results[0]?.excerpt).toContain("crawler");
  });
});
