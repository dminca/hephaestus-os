const assert = require("node:assert/strict");
const { rmSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const { root, assets, renovateRequire, run, npmProject } = require("./helpers.cjs");
const semver = renovateRequire("semver");

for (const directory of [path.join(root, ".github/renovate"), path.join(assets, "node")]) {
  const manifest = require(path.join(directory, "package.json"));
  test(`${manifest.name} enforces Node and npm engines with npm ci`, (t) => {
    const project = npmProject(directory, { name: "runtime-fixture", engines: manifest.engines });
    t.after(() => rmSync(project, { recursive: true, force: true }));
    const npm = run("npm", ["--version"]).output.trim();
    const supported = semver.satisfies(process.versions.node, manifest.engines.node)
      && semver.satisfies(npm, manifest.engines.npm);
    const result = run("npm", ["ci", "--offline", "--ignore-scripts", "--no-audit", "--no-fund"], project);
    if (supported) {
      assert.equal(result.status, 0, result.output);
    } else {
      assert.notEqual(result.status, 0, result.output);
      assert.match(result.output, /EBADENGINE/, result.output);
    }
  });
}
