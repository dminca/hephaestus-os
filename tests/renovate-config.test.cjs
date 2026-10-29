const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const { test } = require("node:test");

test("Renovate forwards GitHub authentication to post-upgrade lock tools", () => {
  const configPath = path.resolve(__dirname, "../.github/renovate-global.js");
  const result = spawnSync(
    process.execPath,
    [
      "-e",
      `const assert = require("node:assert/strict");
       const config = require(process.argv[1]);
       assert.deepEqual(config.customEnvVariables, { GH_TOKEN: process.env.GH_TOKEN });
       assert.notEqual(config.exposeAllEnv, true);`,
      configPath,
    ],
    {
      env: { ...process.env, GH_TOKEN: "fixture-token-not-a-secret" },
      encoding: "utf8",
    },
  );
  assert.equal(result.status, 0, result.stderr);
});
