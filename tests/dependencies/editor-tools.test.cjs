const assert = require("node:assert/strict");
const { chmodSync, mkdtempSync, readFileSync, rmSync, writeFileSync } = require("node:fs");
const { tmpdir } = require("node:os");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { test } = require("node:test");
const { root, assets, run } = require("./helpers.cjs");
const { checkTools, executables } = require(path.join(assets, "node/check-tools.cjs"));
const bin = path.join(assets, "node/node_modules/.bin");

test("all bundled editor executables are present and executable", () => {
  checkTools(bin);
});

test("bundled YAML server formats YAML without a Python formatter", async () => {
  const { YAMLFormatter } = await import(pathToFileURL(path.join(
    assets, "node/node_modules/yaml-language-server/out/src/languageservice/services/yamlFormatter.js",
  )));
  const formatter = new YAMLFormatter();
  formatter.configure({ format: true });
  const text = "items: [one,two]\n";
  const edits = await formatter.format({
    getText: () => text,
    positionAt: () => ({ line: 1, character: 0 }),
  }, { tabSize: 2 });
  assert.equal(edits.length, 1);
  assert.equal(edits[0].newText, "items: [one, two]\n");
  const plugins = readFileSync(path.join(assets, "profile/config/nvim/lua/hephaestus/plugins.lua"), "utf8");
  const languages = readFileSync(path.join(assets, "profile/config/nvim/lua/hephaestus/languages.lua"), "utf8");
  const installer = readFileSync(path.join(root, "files/scripts/install-profile.sh"), "utf8");
  assert.match(plugins, /lsp_format = "prefer"/);
  assert.match(languages, /yamlls =[\s\S]*format = \{ enable = true \}/);
  assert.doesNotMatch(plugins + installer, /yamlfix|python\/install\.sh/);
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
