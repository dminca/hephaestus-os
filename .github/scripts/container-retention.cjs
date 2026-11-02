const { createHash } = require("node:crypto");
const { appendFileSync } = require("node:fs");
const { execFileSync } = require("node:child_process");
const { parseArgs } = require("node:util");

const repository = "dminca/hephaestus-os";
const packagePath = "/users/dminca/packages/container/hephaestus-os";
const registryPath = "/v2/dminca/hephaestus-os";
const digestPattern = /^sha256:[a-f0-9]{64}$/;
const indexTypes = new Set([
  "application/vnd.oci.image.index.v1+json",
  "application/vnd.docker.distribution.manifest.list.v2+json",
]);
const imageTypes = new Set([
  "application/vnd.oci.image.manifest.v1+json",
  "application/vnd.docker.distribution.manifest.v2+json",
  "application/vnd.oci.artifact.manifest.v1+json",
]);

function digest(value) {
  if (!digestPattern.test(value)) throw new Error(`Unsupported digest: ${value}`);
  return value;
}

function references(manifest) {
  if (!manifest || (!indexTypes.has(manifest.mediaType) && !imageTypes.has(manifest.mediaType))) {
    throw new Error(`Unsupported manifest type: ${manifest?.mediaType}`);
  }
  if (indexTypes.has(manifest.mediaType) && !Array.isArray(manifest.manifests)) {
    throw new Error("Index is missing its manifests");
  }
  return [
    ...(manifest.manifests || []),
    ...(manifest.subject ? [manifest.subject] : []),
  ].map((entry) => digest(entry.digest));
}

function selectRoots(versions, pulls) {
  const tagged = (tag) => versions.filter((version) => version.metadata.container.tags.includes(tag));
  const main = tagged("latest");
  if (main.length !== 1) throw new Error("Expected exactly one version tagged latest");
  const roots = [{ name: "main/latest", digest: main[0].name }];
  for (const pull of [...pulls].sort((a, b) => a.number - b.number)) {
    const pattern = new RegExp(`^pr-${pull.number}-[0-9]+$`);
    const candidates = versions.filter((version) => version.metadata.container.tags.some((tag) => pattern.test(tag)));
    candidates.sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at) || b.id - a.id);
    if (candidates.length) roots.push({ name: `PR ${pull.number}`, digest: candidates[0].name });
  }
  return roots;
}

function planRetention(versions, pulls, manifests, referrers = new Map()) {
  const byDigest = new Map();
  const byId = new Set();
  const attachments = new Map();
  const attach = (subject, artifact) => {
    if (!attachments.has(subject)) attachments.set(subject, new Set());
    attachments.get(subject).add(artifact);
  };
  for (const version of versions) {
    digest(version.name);
    if (!Number.isSafeInteger(version.id) || version.id <= 0 || byId.has(version.id) || byDigest.has(version.name)) {
      throw new Error("Duplicate or invalid package version");
    }
    if (!Number.isFinite(Date.parse(version.created_at)) || !Array.isArray(version.metadata?.container?.tags)) {
      throw new Error("Incomplete package version metadata");
    }
    byDigest.set(version.name, version);
    byId.add(version.id);
    for (const tag of version.metadata.container.tags) {
      if (typeof tag !== "string" || !/^[\w][\w.-]{0,127}$/.test(tag)) throw new Error("Invalid registry tag");
      const attachment = tag.match(/^sha256-([a-f0-9]{64})(?:\.[\w.-]+)?$/);
      if (attachment) attach(`sha256:${attachment[1]}`, version.name);
    }
    const manifest = manifests.get(version.name);
    references(manifest);
    if (manifest.subject) attach(digest(manifest.subject.digest), version.name);
  }
  for (const [subject, children] of referrers) {
    for (const child of children) attach(digest(subject), digest(child));
  }
  const roots = selectRoots(versions, pulls);
  const reasons = new Map();
  const visit = (value, reason) => {
    if (!byDigest.has(value)) throw new Error(`Protected digest missing from package inventory: ${value}`);
    if (reasons.has(value)) return;
    reasons.set(value, reason);
    for (const child of references(manifests.get(value))) visit(child, `manifest dependency of ${value}`);
    for (const child of attachments.get(value) || []) visit(child, `attachment of ${value}`);
  };
  for (const root of roots) visit(root.digest, root.name);
  const keep = versions.filter((version) => reasons.has(version.name));
  const remove = versions.filter((version) => !reasons.has(version.name));
  const snapshot = JSON.stringify({
    versions: versions.map((version) => ({
      id: version.id, name: version.name, created_at: version.created_at,
      updated_at: version.updated_at, tags: [...version.metadata.container.tags].sort(),
    })).sort((a, b) => a.id - b.id),
    pulls: pulls.map((pull) => ({
      number: pull.number, sha: pull.head?.sha, updated_at: pull.updated_at,
    })).sort((a, b) => a.number - b.number),
    keep: keep.map((version) => version.name).sort(),
  });
  return { roots, keep, remove, reasons, fingerprint: createHash("sha256").update(snapshot).digest("hex") };
}

function nextPage(response, current, origin, prefix) {
  const link = response.headers.get("link") || "";
  const match = link.match(/<([^>]+)>;\s*rel="next"/);
  if (!match) {
    if (link.includes('rel="next"')) throw new Error("Malformed pagination header");
    return null;
  }
  const next = new URL(match[1], current);
  if (next.origin !== origin || next.pathname !== prefix) throw new Error("Unsafe pagination URL");
  return next.href;
}

async function pages(request, initial, origin, prefix, extract = (value) => value) {
  const result = [];
  const visited = new Set();
  let url = initial;
  while (url) {
    if (visited.has(url)) throw new Error("Pagination loop");
    visited.add(url);
    const response = await request(url);
    const items = extract(await response.json());
    if (!Array.isArray(items)) throw new Error("Expected a complete list response");
    result.push(...items);
    url = nextPage(response, url, origin, prefix);
  }
  return result;
}

async function client(token, fetcher = fetch) {
  const request = async (url, headers, method = "GET", allowMissing = false) => {
    const response = await fetcher(url, { method, headers, signal: AbortSignal.timeout(30_000) });
    if (allowMissing && response.status === 404) return null;
    if (!response.ok) throw new Error(`${method} ${new URL(url).pathname}: HTTP ${response.status}`);
    return response;
  };
  const apiHeaders = {
    Authorization: `Bearer ${token}`,
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  const api = (path, method) => request(new URL(path, "https://api.github.com").href, apiHeaders, method);
  const auth = await request(
    `https://ghcr.io/token?service=ghcr.io&scope=repository:${repository}:pull`,
    { Authorization: `Basic ${Buffer.from(`dminca:${token}`).toString("base64")}` },
  );
  const registryToken = (await auth.json()).token;
  if (!registryToken) throw new Error("Registry did not return a pull token");
  const registryHeaders = {
    Authorization: `Bearer ${registryToken}`,
    Accept: [...indexTypes, ...imageTypes].join(", "),
  };
  const registry = (path, allowMissing = false) =>
    request(new URL(path, "https://ghcr.io").href, registryHeaders, "GET", allowMissing);
  const manifest = async (value) => {
    const response = await registry(`${registryPath}/manifests/${digest(value)}`);
    const text = await response.text();
    const actual = `sha256:${createHash("sha256").update(text).digest("hex")}`;
    if (actual !== value) throw new Error(`Registry digest mismatch for ${value}`);
    const document = JSON.parse(text);
    references(document);
    return document;
  };
  return { api, registry, manifest };
}

async function loadPlan(service) {
  const versions = await pages(
    (url) => service.api(url), `https://api.github.com${packagePath}/versions?per_page=100`,
    "https://api.github.com", `${packagePath}/versions`,
  );
  const pulls = await pages(
    (url) => service.api(url), `https://api.github.com/repos/${repository}/pulls?state=open&per_page=100`,
    "https://api.github.com", `/repos/${repository}/pulls`,
  );
  if (pulls.some((pull) => !Number.isSafeInteger(pull.number) || pull.number <= 0)) throw new Error("Invalid PR inventory");
  const manifests = new Map();
  for (let start = 0; start < versions.length; start += 8) {
    await Promise.all(versions.slice(start, start + 8).map(async (version) => {
      manifests.set(version.name, await service.manifest(version.name));
    }));
  }
  // Packages inventory also covers legacy tags and subject-based referrers on
  // registries that return 404 for the OCI referrers API.
  const referrers = new Map();
  let plan = planRetention(versions, pulls, manifests, referrers);
  const checked = new Set();
  while (plan.keep.some((version) => !checked.has(version.name))) {
    for (const version of plan.keep) {
      if (checked.has(version.name)) continue;
      checked.add(version.name);
      const path = `${registryPath}/referrers/${version.name}`;
      const first = await service.registry(path, true);
      if (!first) continue;
      let firstPage = true;
      const children = await pages(
        async (url) => {
          if (firstPage) { firstPage = false; return first; }
          return service.registry(url);
        },
        `https://ghcr.io${path}`, "https://ghcr.io", path,
        (value) => {
          if (!indexTypes.has(value.mediaType) || !Array.isArray(value.manifests)) throw new Error("Invalid referrers index");
          return value.manifests;
        },
      );
      referrers.set(version.name, children.map((child) => digest(child.digest)));
    }
    plan = planRetention(versions, pulls, manifests, referrers);
  }
  return plan;
}

function summary(plan, apply) {
  return [
    `## Container retention: ${apply ? "apply" : "dry-run"}`,
    "",
    `Plan fingerprint: \`${plan.fingerprint}\``,
    "",
    ...plan.roots.map((root) => `- ${root.name}: \`${root.digest}\``),
    "",
    `Keep ${plan.keep.length} package versions; delete ${plan.remove.length}. Supporting manifests and signatures count as versions, not additional retained images.`,
    "",
    "| Action | Version | Digest | Tags | Reason |",
    "|---|---|---|---|---|",
    ...[...plan.keep, ...plan.remove].map((version) =>
      `| ${plan.reasons.has(version.name) ? "keep" : "delete"} | ${version.id} | ${version.name} | ${version.metadata.container.tags.join(", ")} | ${plan.reasons.get(version.name) || "outside retained image graphs"} |`),
    "",
    "Strict retention removes registry rollback to older builds. Dry-run proves read access, not package-admin deletion permission.",
    "",
  ].join("\n");
}

function verifyRoots(plan) {
  for (const root of plan.roots) {
    try {
      execFileSync("cosign", ["verify", "--key", "cosign.pub", `ghcr.io/${repository}@${root.digest}`], {
        stdio: ["ignore", "ignore", "pipe"],
        timeout: 120_000,
      });
    } catch (error) {
      throw new Error(`Signature verification failed for ${root.name} (${root.digest}): ${error.message}`);
    }
  }
}

async function execute(service, { apply = false, expectedPlan, verify = verifyRoots, report = console.log } = {}) {
  const plan = await loadPlan(service);
  if (expectedPlan && expectedPlan !== plan.fingerprint) throw new Error("Reviewed plan changed; run and review a new dry-run");
  verify(plan);
  report(summary(plan, apply));
  if (!apply) return plan;
  const refreshed = await loadPlan(service);
  if (refreshed.fingerprint !== plan.fingerprint) throw new Error("Inventory or PR state changed; refusing deletion");
  for (const version of plan.remove) {
    await service.api(`${packagePath}/versions/${version.id}`, "DELETE");
    report(`Deleted version ${version.id} (${version.name}).`);
  }
  const after = await loadPlan(service);
  verify(after);
  if (after.remove.length || after.keep.length !== plan.keep.length ||
      after.roots.some((root, index) => root.digest !== plan.roots[index]?.digest)) {
    throw new Error("Post-deletion inventory differs from the reviewed retention result");
  }
  report(`Deleted ${plan.remove.length} versions; retained roots and signatures verified.`);
  return after;
}

async function main() {
  const { values } = parseArgs({
    options: { apply: { type: "boolean", default: false }, "expected-plan": { type: "string" } },
  });
  const apply = values.apply;
  const expectedPlan = values["expected-plan"];
  if (expectedPlan !== undefined && !/^[a-f0-9]{64}$/.test(expectedPlan)) throw new Error("Expected a valid reviewed plan fingerprint");
  if (process.env.GITHUB_REPOSITORY && process.env.GITHUB_REPOSITORY !== repository) throw new Error("Refusing to operate outside the configured repository");
  if (!process.env.GH_TOKEN) throw new Error("GH_TOKEN with package read access is required; apply additionally requires package admin access");
  const service = await client(process.env.GH_TOKEN);
  await execute(service, {
    apply, expectedPlan,
    report: (text) => {
      console.log(text);
      if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${text}\n`);
    },
  });
}

module.exports = { references, selectRoots, planRetention, pages, client, loadPlan, execute, summary };
if (require.main === module) main().catch((error) => { console.error(error.message); process.exitCode = 1; });
