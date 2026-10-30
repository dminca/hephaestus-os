const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { before, test } = require("node:test");
const { root, renovateRequire } = require("./helpers.cjs");
const semver = renovateRequire("semver");
const config = JSON.parse(readFileSync(path.join(root, "renovate.json5"), "utf8"));
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

test("effective TypeScript policy allows 6 and rejects 7", async () => {
  const result = await policy("typescript", "files/system/usr/share/hephaestus/node/package.json");
  assert.ok(semver.satisfies("6.0.3", result.allowedVersions));
  assert.equal(semver.satisfies("7.0.2", result.allowedVersions), false);
});

test("TypeScript and Svelte stay separate from unrelated npm and Renovate updates", async () => {
  const editor = "files/system/usr/share/hephaestus/node/package.json";
  for (const name of ["typescript", "svelte-language-server", "typescript-language-server"]) {
    assert.equal((await policy(name, editor)).groupName, "TypeScript and Svelte compatibility");
  }
  assert.notEqual((await policy("pyright", editor)).groupName, "TypeScript and Svelte compatibility");
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

test("custom managers extract both Rust revisions and every BlueBuild module digest", async () => {
  const { extractPackageFile } = await load("modules/manager/custom/regex/index.js");
  const script = readFileSync(path.join(root, "files/scripts/install-profile.sh"), "utf8");
  const rust = config.customManagers.find((manager) => manager.description === "Track locked Rust source revisions");
  const sources = extractPackageFile(script, "files/scripts/install-profile.sh", rust);
  assert.deepEqual(sources.deps.map((dep) => dep.depName), ["oxalica/nil", "nerdypepper/statix"]);
  for (const dep of sources.deps) assert.match(dep.currentDigest, /^[a-f0-9]{40}$/);
  const modules = config.customManagers.find((manager) => manager.description === "Track immutable BlueBuild module images");
  const recipe = readFileSync(path.join(root, "recipes/recipe.yml"), "utf8");
  const images = extractPackageFile(recipe, "recipes/recipe.yml", modules);
  assert.equal(images.deps.length, 5);
  for (const dep of images.deps) assert.match(dep.currentDigest, /^sha256:[a-f0-9]{64}$/);
});

test("Rust revision updates resolve the default branch without treating HEAD as a named ref", async () => {
  const { extractPackageFile } = await load("modules/manager/custom/regex/index.js");
  const { GitRefsDatasource } = await load("modules/datasource/git-refs/index.js");
  const filename = "files/scripts/install-profile.sh";
  const manager = config.customManagers.find((entry) => entry.description === "Track locked Rust source revisions");
  const extracted = extractPackageFile(readFileSync(path.join(root, filename), "utf8"), filename, manager);
  assert.ok(extracted.deps.length > 0);
  const digest = "a".repeat(40);
  const datasource = new GitRefsDatasource();
  datasource.getRawRefs = async () => [
    { type: "", value: "HEAD", hash: digest },
    { type: "heads", value: "main", hash: digest },
  ];
  assert.equal(await datasource.getDigest({}, "HEAD"), null);
  for (const dependency of extracted.deps) {
    assert.equal(dependency.currentValue, undefined);
    assert.equal(await datasource.getDigest(dependency, dependency.currentValue), digest);
  }
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

test("Renovate discovers yamlfix through its compiled Python requirements", async () => {
  const { extractAllPackageFiles } = await load("modules/manager/pip-compile/extract.js");
  const files = await extractAllPackageFiles({}, ["files/system/usr/share/hephaestus/python/requirements.txt"]);
  const yamlfix = files.flatMap((file) => file.deps).find((dep) => dep.depName === "yamlfix");
  assert.equal(yamlfix.currentValue, "==1.19.1");
});

test("the Python lock compiler keeps compatible Click flag defaults", async () => {
  const { applyPackageRules } = await load("util/package-rules/index.js");
  const result = await applyPackageRules({
    manager: "pip_requirements", datasource: "pypi", packageName: "click", depName: "click",
    packageFile: ".github/python-tools.txt", packageRules: config.packageRules,
  });
  assert.ok(semver.satisfies("8.4.2", result.allowedVersions));
  assert.equal(semver.satisfies("8.5.0", result.allowedVersions), false);
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
