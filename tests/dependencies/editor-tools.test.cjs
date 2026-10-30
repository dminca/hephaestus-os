const assert = require("node:assert/strict");
const { chmodSync, mkdtempSync, rmSync, writeFileSync } = require("node:fs");
const { tmpdir } = require("node:os");
const path = require("node:path");
const { test } = require("node:test");
const { assets, run } = require("./helpers.cjs");
const { checkTools, executables } = require(path.join(assets, "node/check-tools.cjs"));
const bin = path.join(assets, "node/node_modules/.bin");

test("all bundled editor executables are present and executable", () => {
  checkTools(bin);
});

for (const name of ["tsc", "typescript-language-server", "biome", "pyright"]) {
  test(`${name} starts on the target platform`, () => {
    const result = run(path.join(bin, name), ["--version"]);
    assert.equal(result.status, 0, result.output);
    assert.match(result.output, /\d+\.\d+/);
  });
}

test("a missing tsserver fails the same check used by the image build", (t) => {
  const directory = mkdtempSync(path.join(tmpdir(), "hephaestus-bins-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  for (const executable of executables.filter((name) => name !== "tsserver")) {
    const filename = path.join(directory, executable);
    writeFileSync(filename, "");
    chmodSync(filename, 0o755);
  }
  assert.throws(() => checkTools(directory), /Missing executable: tsserver/);
});
