import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { parse } from "yaml";

const workflowPath = new URL("../.github/workflows/release-please.yml", import.meta.url);
const configPath = new URL("../release-please-config.json", import.meta.url);
const manifestPath = new URL("../.release-please-manifest.json", import.meta.url);

async function readReleaseFiles() {
  const [workflowSource, configSource, manifestSource] = await Promise.all([
    readFile(workflowPath, "utf8"),
    readFile(configPath, "utf8"),
    readFile(manifestPath, "utf8"),
  ]);

  return {
    workflowSource,
    workflow: parse(workflowSource),
    config: JSON.parse(configSource),
    manifest: JSON.parse(manifestSource),
  };
}

test("configures one simple root release with a SemVer manifest", async () => {
  const { config, manifest } = await readReleaseFiles();

  assert.deepEqual(Object.keys(manifest), ["."]);
  assert.match(manifest["."], /^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/);
  assert.deepEqual(Object.keys(config.packages), ["."]);
  assert.equal(config.packages["."].releaseType, undefined);
  assert.equal(config.packages["."]["release-type"], "simple");
  assert.equal(config.packages["."]["package-name"], "opendart-spec");
  assert.equal(config.packages["."]["include-component-in-tag"], false);
  assert.equal(config.packages["."]["include-v-in-tag"], true);
  assert.equal(config.packages["."].draft, true);
  assert.equal(config.packages["."]["force-tag-creation"], true);
});

test("pins release actions and validates before creating a release", async () => {
  const { workflow, workflowSource } = await readReleaseFiles();
  const job = workflow.jobs["release-please"];
  const uses = job.steps.filter((step) => step.uses).map((step) => step.uses);
  const verifyIndex = job.steps.findIndex((step) => step.run === "npm run verify:opendart");
  const draftIndex = job.steps.findIndex((step) => step.id === "draft");
  const releaseIndex = job.steps.findIndex((step) => step.id === "release");

  assert.deepEqual(workflow.on.push.branches, ["main"]);
  assert.equal(Object.hasOwn(workflow.on, "workflow_dispatch"), false);
  assert.equal(workflow.concurrency.group, "release-please");
  assert.deepEqual(workflow.permissions, {
    contents: "write",
    issues: "write",
    "pull-requests": "write",
  });
  assert.ok(uses.every((value) => /@[0-9a-f]{40}$/.test(value)));
  assert.ok(verifyIndex >= 0 && verifyIndex < draftIndex && draftIndex < releaseIndex);
  assert.match(job.steps[draftIndex].run, /gh release view/);
  assert.match(job.steps[releaseIndex].if, /recovering != 'true'/);
  assert.doesNotMatch(workflowSource, /npm publish|--clobber/);
});

test("uploads only the versioned bundle and its checksum after release", async () => {
  const { workflow } = await readReleaseFiles();
  const steps = workflow.jobs["release-please"].steps;
  const prepare = steps.find((step) => step.name === "Prepare release assets");
  const upload = steps.find((step) => step.name === "Upload release assets");
  const publish = steps.find((step) => step.name === "Publish immutable release");
  const releaseCondition = "steps.release.outputs.release_created == 'true'";
  const recoveryCondition = "steps.draft.outputs.recovering == 'true'";

  assert.match(prepare.if, new RegExp(releaseCondition.replaceAll(".", "\\.")));
  assert.match(prepare.if, new RegExp(recoveryCondition.replaceAll(".", "\\.")));
  assert.match(prepare.run, /sha256sum openapi\.bundle\.yaml/);
  assert.match(upload.if, new RegExp(releaseCondition.replaceAll(".", "\\.")));
  assert.match(upload.if, new RegExp(recoveryCondition.replaceAll(".", "\\.")));
  assert.match(upload.run, /release-assets\/openapi\.bundle\.yaml \\/);
  assert.match(upload.run, /release-assets\/openapi\.bundle\.yaml\.sha256/);
  assert.match(upload.run, /gh release download/);
  assert.match(upload.run, /cmp -s/);
  assert.match(publish.if, new RegExp(releaseCondition.replaceAll(".", "\\.")));
  assert.match(publish.if, new RegExp(recoveryCondition.replaceAll(".", "\\.")));
  assert.match(publish.run, /gh release edit .* --draft=false --latest/);
  assert.ok(steps.indexOf(prepare) < steps.indexOf(upload));
  assert.ok(steps.indexOf(upload) < steps.indexOf(publish));
});
