const { spawnSync } = require("node:child_process");
const { mkdtempSync, writeFileSync, copyFileSync, realpathSync } = require("node:fs");
const { tmpdir } = require("node:os");
const path = require("node:path");
const { createRequire } = require("node:module");

const root = path.resolve(__dirname, "../..");
const assets = path.join(root, "files/system/usr/share/hephaestus");
const renovateRequire = createRequire(path.join(root, ".github/renovate/package.json"));

function run(command, args, cwd = root, env = {}) {
  if (command === "npm") {
    const location = spawnSync("which", ["npm"], { encoding: "utf8" });
    if (location.status !== 0) throw new Error("npm is required for dependency tests");
    args = [realpathSync(location.stdout.trim()), ...args];
    command = process.execPath;
  }
  const result = spawnSync(command, args, {
    cwd,
    env: { ...process.env, ...env },
    encoding: "utf8",
    timeout: 120_000,
  });
  if (result.error) throw result.error;
  return { status: result.status, output: result.stdout + result.stderr };
}

function npmProject(directory, manifest) {
  const project = mkdtempSync(path.join(tmpdir(), "hephaestus-npm-"));
  copyFileSync(path.join(directory, ".npmrc"), path.join(project, ".npmrc"));
  writeFileSync(path.join(project, "package.json"), JSON.stringify(manifest));
  writeFileSync(path.join(project, "package-lock.json"), JSON.stringify({
    name: manifest.name,
    lockfileVersion: 3,
    packages: { "": manifest },
  }));
  return project;
}

module.exports = { root, assets, renovateRequire, run, npmProject };
