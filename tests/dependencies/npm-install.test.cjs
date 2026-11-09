const assert = require("node:assert/strict");
const { rmSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const { root, assets, renovateRequire, run, npmProject } = require("./helpers.cjs");
const semver = renovateRequire("semver");
const editor = path.join(assets, "node");

test("both npm manifests agree with their committed locks", () => {
  for (const directory of [editor, path.join(root, ".github/renovate")]) {
    const declared = require(path.join(directory, "package.json"));
    const locked = require(path.join(directory, "package-lock.json")).packages[""];
    assert.deepEqual(locked.dependencies, declared.dependencies, directory);
    assert.deepEqual(locked.engines, declared.engines, directory);
  }
});

test("Renovate runs on a Node range supported by its installed release", () => {
  const declared = require(path.join(root, ".github/renovate/package.json"));
  const renovate = renovateRequire("renovate/package.json");
  assert.ok(semver.subset(declared.engines.node, renovate.engines.node));
  assert.ok(semver.satisfies(process.versions.node, renovate.engines.node));
  const result = run(path.join(root, ".github/renovate/node_modules/.bin/renovate"), ["--version"]);
  assert.equal(result.status, 0, result.output);
  assert.match(result.output, new RegExp(declared.dependencies.renovate.replaceAll(".", "\\.")));
});

test("declared Node minima satisfy every dependency in both locked graphs", () => {
  for (const directory of [editor, path.join(root, ".github/renovate")]) {
    const declared = require(path.join(directory, "package.json"));
    const minimum = semver.minVersion(declared.engines.node);
    const lock = require(path.join(directory, "package-lock.json"));
    for (const [name, dependency] of Object.entries(lock.packages)) {
      if (dependency.engines?.node) {
        assert.ok(semver.satisfies(minimum, dependency.engines.node),
          `${name || declared.name} requires Node ${dependency.engines.node}, not ${minimum}`);
      }
    }
    for (const unsupported of ["22.22.2", "26.13.0"]) {
      assert.equal(semver.satisfies(unsupported, declared.engines.node), false,
        `${declared.name} must reject Node ${unsupported}`);
    }
  }
});

test("engine-strict rejects an npm version outside the declared range", (t) => {
  const project = npmProject(editor, { name: "npm-fixture", engines: { npm: ">999" } });
  t.after(() => rmSync(project, { recursive: true, force: true }));
  const result = run("npm", ["ci", "--offline", "--ignore-scripts", "--no-audit", "--no-fund"], project);
  assert.notEqual(result.status, 0, result.output);
  assert.match(result.output, /EBADENGINE/, result.output);
});
