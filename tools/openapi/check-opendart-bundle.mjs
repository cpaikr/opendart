import { spawn } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const TOOL_DIR = dirname(fileURLToPath(import.meta.url));
const SPEC_DIR = fileURLToPath(new URL("../../docs/opendart/", import.meta.url));
const ROOT = join(SPEC_DIR, "openapi.yaml");
const CONFIG = join(SPEC_DIR, "redocly.yaml");
const COMMITTED_BUNDLE = join(SPEC_DIR, "generated", "openapi.bundle.yaml");
const REDOCLY = join(TOOL_DIR, "node_modules", "@redocly", "cli", "bin", "cli.js");

async function runRedocly(output) {
  await new Promise((resolve, reject) => {
    const child = spawn(
      process.execPath,
      [
        REDOCLY,
        "bundle",
        ROOT,
        "--config",
        CONFIG,
        "--output",
        output,
        "--component-renaming-conflicts-severity",
        "error",
      ],
      { cwd: TOOL_DIR, stdio: "inherit" },
    );
    child.once("error", reject);
    child.once("exit", (code, signal) => {
      if (code === 0) resolve();
      else reject(new Error(`Redocly bundle failed (${signal || `exit ${code}`})`));
    });
  });
}

async function main() {
  const temporaryDirectory = await mkdtemp(join(tmpdir(), "dartdb-opendart-bundle-"));
  try {
    const temporaryBundle = join(temporaryDirectory, "openapi.bundle.yaml");
    await runRedocly(temporaryBundle);

    let committed;
    try {
      committed = await readFile(COMMITTED_BUNDLE);
    } catch (error) {
      if (error.code === "ENOENT") {
        throw new Error("Committed OpenDART bundle is missing; run `npm run bundle:opendart`");
      }
      throw error;
    }
    const generated = await readFile(temporaryBundle);
    if (!committed.equals(generated)) {
      throw new Error("Committed OpenDART bundle is stale; run `npm run bundle:opendart`");
    }
    process.stdout.write("Committed OpenDART bundle matches a fresh bundle.\n");
  } finally {
    await rm(temporaryDirectory, { recursive: true, force: true });
  }
}

main().catch((error) => {
  process.stderr.write(`${error.stack || error.message}\n`);
  process.exitCode = 1;
});
