const { execFileSync } = require("node:child_process");
const { appendFileSync, readFileSync } = require("node:fs");

const checks = ["tests", "dependencies", "node-runtimes", "parsers", "bluebuild"];

function classify(paths) {
  const selected = Object.fromEntries(checks.map((name) => [name, false]));
  if (!paths?.length) return Object.fromEntries(checks.map((name) => [name, true]));
  for (const path of paths) {
    // Only explicit documentation locations are cheap; unknown inputs fail open.
    if (/^[^/]+\.md$/.test(path) || /^docs\/.*\.md$/.test(path)) continue;
    selected.tests = true;
    selected.bluebuild = true;
    if (path === "cosign.pub") continue;
    selected.dependencies = true;
    selected.parsers = true;
    // Recipes affect Renovate extraction and Fedora/parser alignment, not Node engines.
    if (/^recipes\/[^/]+\.ya?ml$/.test(path)) continue;
    selected["node-runtimes"] = true;
  }
  return selected;
}

function changedPaths(eventName, event, git = (...args) =>
  execFileSync("git", args, { encoding: "utf8" })) {
  let base;
  let head;
  let range;
  if (eventName === "pull_request") {
    base = event.pull_request?.base?.sha;
    head = event.pull_request?.head?.sha;
    range = "...";
  } else if (eventName === "push") {
    base = event.before;
    head = event.after;
    range = "..";
  } else {
    return null;
  }
  if (![base, head].every((sha) => /^[a-f0-9]{40}$/.test(sha || "") && !/^0+$/.test(sha))) return null;
  try {
    const output = git("diff", "--name-only", "--no-renames", "-z", `${base}${range}${head}`, "--");
    // Empty comparisons must not suppress daily/manual or new-branch validation.
    return output ? output.split("\0").filter(Boolean) : null;
  } catch (error) {
    console.warn(`CI file comparison unavailable; running full validation: ${error.message}`);
    return null;
  }
}

function gate(needs) {
  if (needs.changes?.result !== "success") return false;
  if (Object.values(needs).some(({ result }) => !["success", "skipped"].includes(result))) return false;
  for (const name of checks) {
    const selected = needs.changes.outputs?.[name];
    if (!["true", "false"].includes(selected)) return false;
    if (name === "bluebuild") {
      const results = [needs.bluebuild?.result, needs["image-validation"]?.result].sort();
      const expected = selected === "true" ? ["skipped", "success"] : ["skipped", "skipped"];
      if (JSON.stringify(results) !== JSON.stringify(expected)) return false;
    } else if (needs[name]?.result !== (selected === "true" ? "success" : "skipped")) {
      return false;
    }
  }
  return true;
}

if (require.main === module) {
  if (process.argv[2] === "gate") {
    if (!gate(JSON.parse(process.env.NEEDS_JSON))) {
      console.error("CI failed: a required job failed, was cancelled, or was unexpectedly skipped.");
      process.exitCode = 1;
    }
  } else {
    const event = JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, "utf8"));
    const selected = classify(changedPaths(process.env.GITHUB_EVENT_NAME, event));
    appendFileSync(process.env.GITHUB_OUTPUT,
      Object.entries(selected).map(([name, value]) => `${name}=${value}\n`).join(""));
  }
}

module.exports = { checks, classify, changedPaths, gate };
