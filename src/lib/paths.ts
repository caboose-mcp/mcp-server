import { access, realpath, stat } from "node:fs/promises";
import { homedir } from "node:os";
import path from "node:path";

const SECRET_HINTS = [
  ".env",
  ".pem",
  ".key",
  ".p12",
  ".pfx",
  "id_rsa",
  "id_ed25519",
  "secret",
  "secrets",
  "credential",
  "credentials",
  "token"
];

export function expandHome(input: string): string {
  if (input === "~") {
    return homedir();
  }
  if (input.startsWith("~/")) {
    return path.join(homedir(), input.slice(2));
  }
  return input;
}

export function normalizePath(input: string): string {
  return path.resolve(expandHome(input));
}

export function isSecretLike(filePath: string): boolean {
  const normalized = filePath.toLowerCase();
  return SECRET_HINTS.some((hint) => normalized.includes(hint));
}

export function isWithin(parent: string, child: string): boolean {
  const relative = path.relative(parent, child);
  return relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));
}

export async function existingDirectory(input: string): Promise<string | undefined> {
  const normalized = normalizePath(input);
  try {
    const info = await stat(normalized);
    if (!info.isDirectory()) {
      return undefined;
    }
    return realpath(normalized);
  } catch {
    return undefined;
  }
}

export async function realExistingPath(input: string): Promise<string> {
  await access(input);
  return realpath(input);
}

export async function assertContainedPath(
  candidate: string,
  allowedRoots: string[],
  label: string
): Promise<string> {
  const realCandidate = await realExistingPath(candidate);
  const realRoots = await Promise.all(allowedRoots.map((root) => realpath(root)));
  if (!realRoots.some((root) => isWithin(root, realCandidate))) {
    throw new Error(`${candidate} is not under an allowed ${label}`);
  }
  return realCandidate;
}

export async function assertOutputDirectory(
  outputDir: string,
  allowedRoots: string[]
): Promise<string> {
  const normalizedOutput = normalizePath(outputDir);
  const realParent = await nearestExistingParent(normalizedOutput);
  const realRoots = await Promise.all(allowedRoots.map((root) => realpath(root)));
  if (!realRoots.some((root) => isWithin(root, realParent))) {
    throw new Error(`${outputDir} is not under an allowed output root`);
  }
  return normalizedOutput;
}

async function nearestExistingParent(input: string): Promise<string> {
  let current = input;
  while (current !== path.dirname(current)) {
    try {
      const info = await stat(current);
      if (info.isDirectory()) {
        return realpath(current);
      }
      return realpath(path.dirname(current));
    } catch {
      current = path.dirname(current);
    }
  }
  return realpath(current);
}

export function splitPathEnv(value: string | undefined): string[] {
  if (!value) {
    return [];
  }
  return value
    .split(path.delimiter)
    .flatMap((part) => part.split(","))
    .map((part) => part.trim())
    .filter(Boolean);
}
