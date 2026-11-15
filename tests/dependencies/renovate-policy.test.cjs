const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { before, test } = require("node:test");
const { root, renovateRequire } = require("./helpers.cjs");
const semver = renovateRequire("semver");
const config = renovateRequire("json5").parse(readFileSync(path.join(root, "renovate.json5"), "utf8"));
const renovateRoot = path.dirname(renovateRequire.resolve("renovate/package.json"));
const load = (file) => import(pathToFileURL(path.join(renovateRoot, "dist", file)));

before(async () => {
  const { init } = await load("logger/index.js");
  await init();
  const { GlobalConfig } = await load("config/global.js");
  GlobalConfig.set({ localDir: root });
});

async function policy(packageName, packageFile, depType = "dependencies") {
  const { applyPackageRules } = await load("util/package-rules/index.js");
  return applyPackageRules({
    manager: "npm", datasource: "npm", packageName, depName: packageName,
    packageFile, depType, updateType: "major", packageRules: config.packageRules,
  });
}

test("system preview tools stay separate from Renovate updates", async () => {
  assert.equal((await policy("@mermaid-js/mermaid-cli", "files/system/usr/share/hephaestus/node/package.json")).groupName,
    "npm major upgrades");
  assert.equal((await policy("renovate", ".github/renovate/package.json")).groupName, "Renovate runtime");
});

test("runtime policy rejects automatic Node 26 and npm 12 updates", async () => {
  for (const file of [".github/renovate/package.json", "files/system/usr/share/hephaestus/node/package.json"]) {
    const node = await policy("node", file, "engines");
    const manifest = require(path.join(root, file));
    assert.ok(semver.satisfies(semver.minVersion(manifest.engines.node), node.allowedVersions));
    assert.equal(semver.satisfies("26.13.0", node.allowedVersions), false);
    const npm = await policy("npm", file, "packageManager");
    assert.ok(semver.satisfies("11.19.0", npm.allowedVersions));
    assert.equal(semver.satisfies("12.0.0", npm.allowedVersions), false);
  }
});

test("custom managers extract every BlueBuild module digest", async () => {
  const { extractPackageFile } = await load("modules/manager/custom/regex/index.js");
  const modules = config.customManagers.find((manager) => manager.description === "Track immutable BlueBuild module images");
  const recipe = readFileSync(path.join(root, "recipes/recipe.yml"), "utf8");
  const images = extractPackageFile(recipe, "recipes/recipe.yml", modules);
  assert.equal(images.deps.length, 6);
  for (const dep of images.deps) assert.match(dep.currentDigest, /^sha256:[a-f0-9]{64}$/);
});

test("SLSA workflow and signer use the same release and reviewed update group", async () => {
  const regex = await load("modules/manager/custom/regex/index.js");
  const actions = await load("modules/manager/github-actions/index.js");
  const { applyPackageRules } = await load("util/package-rules/index.js");
  const manager = config.customManagers.find((entry) =>
    entry.description === "Track the SLSA provenance signer release and workflow digest");
  assert.ok(manager, "the verifier signer needs Renovate coverage");
  const filename = manager.managerFilePatterns[0];
  const content = readFileSync(path.join(root, filename), "utf8");
  const signer = regex.extractPackageFile(content, filename, manager);
  assert.equal(signer.deps.length, 1);
  const workflowFile = ".github/workflows/build.yml";
  const workflow = await actions.extractPackageFile(
    readFileSync(path.join(root, workflowFile), "utf8"), workflowFile, {});
  const dependency = workflow.deps.find((dep) => dep.depName === "slsa-framework/actions");
  assert.ok(dependency);
  assert.equal(signer.deps[0].datasource, dependency.datasource);
  assert.equal(signer.deps[0].currentValue, dependency.currentValue);
  assert.equal(signer.deps[0].currentDigest, dependency.currentDigest);
  for (const updateType of ["digest", "patch", "minor", "major"]) {
    for (const [dep, managerName, packageFile] of [
      [signer.deps[0], "custom.regex", filename],
      [dependency, "github-actions", workflowFile],
    ]) {
      const resolved = await applyPackageRules({
        ...dep, manager: managerName, packageName: dep.depName,
        packageFile, updateType, packageRules: config.packageRules,
      });
      assert.equal(resolved.groupName, "SLSA workflow and signer");
      assert.equal(resolved.automerge, false);
    }
  }
  assert.equal(regex.extractPackageFile(
    content.replace("slsa-framework/actions/.github/workflows/attest_actions.yml",
      "untrusted/actions/.github/workflows/attest_actions.yml"),
    filename, manager), null, "unrelated signer identities must not be extracted");
});

test("BlueBuild CLI pins agree and the Nushell image includes a digest", async () => {
  const { extractPackageFile } = await load("modules/manager/custom/regex/index.js");
  const cli = config.customManagers.find((manager) => manager.description === "Track pinned BlueBuild CLI versions");
  const versions = [];
  for (const filename of ["recipes/recipe.yml", ".github/workflows/build.yml"]) {
    const extracted = extractPackageFile(readFileSync(path.join(root, filename), "utf8"), filename, cli);
    assert.equal(extracted.deps.length, 1);
    assert.equal(extracted.deps[0].depName, "blue-build/cli");
    versions.push(extracted.deps[0].currentValue);
  }
  assert.equal(versions[0], versions[1]);
  const nushell = config.customManagers.find((manager) => manager.description === "Track immutable Nushell runtime image");
  const filename = "recipes/recipe.yml";
  const extracted = extractPackageFile(readFileSync(path.join(root, filename), "utf8"), filename, nushell);
  assert.equal(extracted.deps.length, 1);
  assert.equal(extracted.deps[0].depName, "ghcr.io/blue-build/nushell-image");
  assert.ok(semver.valid(extracted.deps[0].currentValue));
  assert.match(extracted.deps[0].currentDigest, /^sha256:[a-f0-9]{64}$/);
});

test("Renovate discovers each npm project's lock, runtime, and strict npmrc", async () => {
  const { extractPackageFile } = await load("modules/manager/npm/extract/index.js");
  for (const filename of [".github/renovate/package.json", "files/system/usr/share/hephaestus/node/package.json"]) {
    const extracted = await extractPackageFile(readFileSync(path.join(root, filename), "utf8"), filename, {});
    assert.equal(extracted.managerData.npmLock, filename.replace("package.json", "package-lock.json"));
    assert.match(extracted.npmrc, /engine-strict=true/);
    assert.match(extracted.npmrc, /strict-peer-deps=true/);
  }
});

test("artifact generation failures set a failing Renovate status", async (t) => {
  const { default: adapters } = await load("modules/platform/api.js");
  const { setPlatformApi } = await load("modules/platform/index.js");
  const { setArtifactErrorStatus } = await load("workers/repository/update/branch/artifacts.js");
  const original = adapters.get("github");
  const statuses = [];
  adapters.set("github", {
    getBranchStatusCheck: async () => null,
    setBranchStatus: async (status) => statuses.push(status),
  });
  setPlatformApi("github");
  t.after(() => {
    adapters.set("github", original);
    setPlatformApi("github");
  });
  const settings = {
    branchName: "renovate/fixture",
    statusCheckNames: { artifactError: "renovate/artifacts" },
  };
  await setArtifactErrorStatus({ ...settings, artifactErrors: [] });
  assert.deepEqual(statuses, []);
  await setArtifactErrorStatus({
    ...settings,
    artifactErrors: [{ fileName: "package-lock.json", stderr: "fixture: lock resolution failed" }],
  });
  assert.deepEqual(statuses, [{
    branchName: settings.branchName,
    context: "renovate/artifacts",
    description: "Artifact file update failure",
    state: "red",
  }]);
});
