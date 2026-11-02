const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const { checks, classify, changedPaths, gate } = require("../.github/scripts/ci-changes.cjs");
const root = path.resolve(__dirname, "..");
const workflow = readFileSync(path.join(root, ".github/workflows/build.yml"), "utf8");
const renovate = readFileSync(path.join(root, ".github/workflows/renovate.yml"), "utf8");
const all = Object.fromEntries(checks.map((name) => [name, true]));
const none = Object.fromEntries(checks.map((name) => [name, false]));

test("only explicit documentation paths skip expensive checks", () => {
  assert.deepEqual(classify(["README.md", "CONTRIBUTING.md", "docs/operations/ci.md"]), none);
  for (const input of [
    "docs/example.sh", "docs/config.yml", "docs/package.json",
    "files/system/usr/share/hephaestus/README.md", "unknown", ".github/new-file",
  ]) {
    assert.deepEqual(classify(["README.md", input]), all, input);
  }
});

test("every test, script, dependency, lock, runtime and CI input runs full validation", () => {
  for (const input of [
    "tests/ci-policy.test.cjs", "tests/profile_test.go",
    "tests/dependencies/helpers.cjs", "tests/dependencies/node-runtime.test.cjs",
    ".github/scripts/ci-changes.cjs", ".github/workflows/build.yml",
    ".github/scripts/container-retention.cjs", "tests/container-retention.test.cjs",
    ".github/workflows/renovate.yml", ".github/workflows/cleanup-images.yml",
    ".github/renovate-global.js", ".github/python-tools.txt",
    ".github/renovate/package.json", ".github/renovate/package-lock.json", ".github/renovate/.npmrc",
    "renovate.json5", "go.mod", "go.sum", "internal/lockfile/lockfile.go",
    "tools/lock-binaries/main.go", "files/scripts/install-profile.sh",
    "files/system/usr/share/hephaestus/build-assets/main.go",
    "files/system/usr/share/hephaestus/binaries.lock.json",
    "files/system/usr/share/hephaestus/sources.lock.json",
    "files/system/usr/share/hephaestus/node/package.json",
    "files/system/usr/share/hephaestus/node/package-lock.json",
    "files/system/usr/share/hephaestus/node/.npmrc",
    "files/system/usr/share/hephaestus/python/requirements.in",
    "files/system/usr/share/hephaestus/python/requirements.txt",
    "files/system/usr/share/hephaestus/python/.python-version",
    "files/system/usr/share/hephaestus/go-tools/go.mod",
    "files/system/usr/share/hephaestus/go-tools/go.sum",
    "files/system/usr/share/hephaestus/build-parsers.lua",
    "files/system/usr/share/hephaestus/check-language-servers.lua",
    "files/system/usr/share/hephaestus/profile/config/nvim/init.lua",
  ]) assert.deepEqual(classify([input]), all, input);
  assert.deepEqual(classify(["recipes/recipe.yml", "tests/profile_test.go"]), all);
});

test("known image-only and recipe changes retain applicable checks", () => {
  assert.deepEqual(classify(["cosign.pub"]), {
    ...none, tests: true, bluebuild: true,
  });
  assert.deepEqual(classify(["recipes/recipe.yml"]), {
    ...all, "node-runtimes": false,
  });
});

test("missing and empty comparisons conservatively select everything", () => {
  for (const paths of [null, undefined, []]) assert.deepEqual(classify(paths), all);
});

const before = "a".repeat(40);
const after = "b".repeat(40);

test("schedule, dispatch and unknown events always run full validation", () => {
  for (const event of ["schedule", "workflow_dispatch", "unknown"]) {
    assert.deepEqual(classify(changedPaths(event, {}, () => {
      assert.fail("non-diff events must not compare commits");
    })), all);
  }
});

test("PRs use their merge base while pushes compare exact before/after commits", () => {
  for (const [name, event, range] of [
    ["pull_request", { pull_request: { base: { sha: before }, head: { sha: after } } }, "..."],
    ["push", { before, after }, ".."],
  ]) {
    const paths = changedPaths(name, event, (...args) => {
      assert.deepEqual(args, ["diff", "--name-only", "--no-renames", "-z", `${before}${range}${after}`, "--"]);
      return "README.md\0docs/path with spaces.md\0";
    });
    assert.deepEqual(classify(paths), none);
  }
});

test("deleted or renamed substantive inputs cannot be hidden behind a documentation filename", () => {
  const paths = changedPaths("push", { before, after }, () =>
    "files/scripts/install-profile.sh\0docs/old-install.md\0");
  assert.deepEqual(classify(paths), all);
});

test("new branches, missing commits, malformed SHAs and diff errors all fall back to full", () => {
  for (const event of [
    {}, { before: "0".repeat(40), after }, { before, after: "0".repeat(40) },
    { before: "--help", after }, { before, after: "$(command)" },
  ]) {
    assert.equal(changedPaths("push", event, () => assert.fail("invalid commits reached git")), null);
  }
  assert.equal(changedPaths("pull_request", {}), null);
  assert.equal(changedPaths("push", { before, after }, () => ""), null);
  assert.equal(changedPaths("push", { before, after }, () => { throw new Error("unavailable commit"); }), null);
  assert.equal(changedPaths("push", { before, after }, () => { throw new Error("output exceeded buffer"); }), null);
});

function needsFor(selected, imageJob = "bluebuild") {
  const needs = {
    changes: { result: "success", outputs: Object.fromEntries(
      Object.entries(selected).map(([name, value]) => [name, String(value)])) },
    "image-validation": { result: "skipped" },
  };
  for (const name of checks) needs[name] = { result: selected[name] ? "success" : "skipped" };
  if (selected.bluebuild && imageJob !== "bluebuild") {
    needs.bluebuild.result = "skipped";
    needs[imageJob].result = "success";
  }
  return needs;
}

test("gate accepts docs-only skips and either successful image path", () => {
  for (const selected of [none, all, classify(["cosign.pub"]), classify(["recipes/recipe.yml"])]) {
    for (const imageJob of ["bluebuild", "image-validation"]) {
      assert.equal(gate(needsFor(selected, imageJob)), true);
    }
  }
});

test("gate rejects failures, cancellations and unexpected skips from every selected job", () => {
  for (const imageJob of ["bluebuild", "image-validation"]) {
    for (const name of ["changes", "tests", "dependencies", "node-runtimes", "parsers", imageJob]) {
      for (const result of ["failure", "cancelled", "skipped"]) {
        const needs = needsFor(all, imageJob);
        needs[name].result = result;
        assert.equal(gate(needs), false, `${name}: ${result}`);
      }
    }
  }
});

test("gate fails closed on missing outputs/results or failures in nonselected jobs", () => {
  assert.equal(gate({}), false);
  for (const name of checks) {
    const needs = needsFor(all);
    delete needs.changes.outputs[name];
    assert.equal(gate(needs), false, name);
    const missing = needsFor(all);
    delete missing[name];
    assert.equal(gate(missing), false, name);
  }
  for (const result of ["failure", "cancelled", "unknown", "success"]) {
    const needs = needsFor(none);
    needs.dependencies.result = result;
    assert.equal(gate(needs), false, result);
  }
  const both = needsFor(all);
  both["image-validation"].result = "success";
  assert.equal(gate(both), false);
});

test("gate command exits nonzero rather than masking failure", () => {
  for (const [needs, status] of [[needsFor(none), 0], [needsFor(all), 0], [{}, 1]]) {
    const result = spawnSync(process.execPath, [".github/scripts/ci-changes.cjs", "gate"], {
      cwd: root, encoding: "utf8", env: { ...process.env, NEEDS_JSON: JSON.stringify(needs) },
    });
    assert.equal(result.status, status, result.stderr);
  }
});

function job(name) {
  const content = workflow.split(`\n  ${name}:\n`)[1];
  assert.ok(content, `missing job ${name}`);
  return content.split(/\n  [a-z][\w-]*:\n/)[0];
}

test("daily rebuilds, per-ref cancellation and all eight runtime cases remain", () => {
  assert.match(workflow, /"00 06 \* \* \*"/);
  assert.doesNotMatch(workflow, /paths-ignore:|pull_request_target/);
  assert.match(workflow, /group: \$\{\{ github.workflow \}\}-\$\{\{ github.ref \|\| github.run_id \}\}\n  cancel-in-progress: true/);
  const matrix = job("node-runtimes");
  assert.match(matrix, /node: \["24.11.0", "24.15.0", "24", "24.10.0", "24.14.0", "22", "26"\]/);
  assert.match(matrix, /npm: \["declared"\]/);
  assert.match(matrix, /- node: "24"\n\s+npm: "10"/);
  assert.match(matrix, /fail-fast: false/);
  assert.match(matrix, /npm ci --omit=dev/);
  assert.match(job("dependencies"), /renovate-config-validator --strict renovate.json5/);
  for (const check of ["renovate-config", "node-runtime", "npm-install", "renovate-policy", "editor-tools"]) {
    assert.ok(job("dependencies").includes(`${check}.test.cjs`), check);
  }
  assert.match(job("parsers"), /TestParserBuildReportsCompleteFailures\|TestLanguageServerBuildValidation/);
  assert.match(job("parsers"), /nvim --headless -u NONE -l "\$HEPHAESTUS_ASSETS\/build-parsers.lua"/);
});

test("selected parser/image jobs explicitly handle skipped dependencies, and final gate always runs", () => {
  for (const name of ["parsers", "bluebuild", "image-validation"]) {
    const content = job(name);
    assert.match(content, /always\(\) && !cancelled\(\)/);
    assert.match(content, /!contains\(needs\.\*\.result, 'failure'\)/);
    assert.match(content, /!contains\(needs\.\*\.result, 'cancelled'\)/);
    assert.match(content, /needs.changes.result == 'success'/);
  }
  assert.match(job("ci"), /^    name: CI status\n/);
  assert.match(job("ci"), /if: always\(\)/);
  assert.match(job("ci"), /NEEDS_JSON: \$\{\{ toJSON\(needs\) \}\}/);
  const gateNeeds = job("ci").split("\n").find((line) => line.trimStart().startsWith("needs:"));
  for (const name of ["changes", ...checks, "image-validation"]) {
    assert.ok(gateNeeds.includes(name), `${name} missing from final gate needs`);
  }
  assert.match(job("changes"), /fetch-depth: 0/);
  assert.match(job("changes"), /persist-credentials: false/);
  assert.match(job("changes"), /node --test tests\/ci-policy.test.cjs tests\/container-retention.test.cjs/);
});

test("publishing is serialized and fork/manual branch builds have read-only credentials", () => {
  const publisher = job("bluebuild");
  assert.match(publisher, /group: ghcr-hephaestus-os\n\s+cancel-in-progress: false/);
  assert.match(publisher, /packages: write/);
  assert.match(publisher, /github.event.pull_request.head.repo.full_name == github.repository/);
  assert.match(publisher, /github.event_name != 'pull_request' && github.ref == 'refs\/heads\/main'/);
  const validation = job("image-validation");
  assert.match(validation, /permissions:\n\s+contents: read\n\s+strategy:/);
  assert.doesNotMatch(validation, /packages:|id-token:/);
  assert.match(validation, /github.event.pull_request.head.repo.full_name != github.repository/);
  assert.match(validation, /github.event_name != 'pull_request' && github.ref != 'refs\/heads\/main'/);
  assert.match(publisher, /steps: &image-steps/);
  assert.match(validation, /steps: \*image-steps/);
  for (const input of ["push", "cosign_private_key", "registry_token"]) {
    const line = publisher.split("\n").find((line) => line.trimStart().startsWith(`${input}:`));
    assert.ok(line.includes("github.event.pull_request.head.repo.full_name == github.repository"), input);
    assert.ok(line.includes("github.ref == 'refs/heads/main'"), input);
    if (input !== "push") assert.ok(line.includes("|| ''"), input);
  }
});

test("all jobs have bounded runtimes and npm caches use both explicit locks, never node_modules", () => {
  for (const name of ["changes", ...checks, "image-validation", "ci"]) {
    assert.match(job(name), /timeout-minutes: (5|10|15|20|60)\n/);
  }
  assert.match(renovate, /timeout-minutes: 45/);
  for (const content of [job("dependencies"), job("node-runtimes"), renovate]) {
    assert.match(content, /cache: npm\n\s+cache-dependency-path: \|\n\s+\.github\/renovate\/package-lock.json\n\s+files\/system\/usr\/share\/hephaestus\/node\/package-lock.json/);
    assert.doesNotMatch(content, /cache:.*node_modules/);
    assert.match(content, /npm ci --no-audit --no-fund/);
  }
  assert.match(renovate, /if: github.ref == 'refs\/heads\/main'/);
  assert.match(renovate, /renovate-config-validator --strict renovate.json5/);
});
