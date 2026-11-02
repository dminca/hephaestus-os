const assert = require("node:assert/strict");
const { test } = require("node:test");
const { readFileSync } = require("node:fs");
const { createHash } = require("node:crypto");
const {
  planRetention, pages, client, execute, summary,
} = require("../.github/scripts/container-retention.cjs");

const imageType = "application/vnd.oci.image.manifest.v1+json";
const indexType = "application/vnd.oci.image.index.v1+json";
const hash = (id) => `sha256:${id.toString(16).padStart(64, "0")}`;
const image = () => ({ schemaVersion: 2, mediaType: imageType, config: {}, layers: [] });
const index = (...ids) => ({
  schemaVersion: 2, mediaType: indexType,
  manifests: ids.map((id) => ({ digest: hash(id) })),
});
const version = (id, tags = [], day = 1) => ({
  id, name: hash(id), created_at: `2026-10-${String(day).padStart(2, "0")}T00:00:00Z`,
  metadata: { container: { tags } },
});
const signatureTag = (id) => `sha256-${hash(id).slice(7)}.sig`;
const pull = (number) => ({ number, head: { sha: `head-${number}` }, updated_at: "2026-10-30T00:00:00Z" });

function fixture() {
  const versions = [
    version(1, ["latest", "44", "20261001"]),
    version(2), version(3), version(4),
    version(5, [signatureTag(1)]), version(6, [signatureTag(3)]),
    version(7, ["pr-19-43"], 2), version(8, ["pr-19-44", "abc1234-44"], 3),
    version(9, ["pr-20-44"], 3), version(10, ["pr-18-44"], 3),
    version(11, [signatureTag(10)]), version(12, ["br-old-44"]),
    version(13), version(14, [signatureTag(4)]),
    version(15, [`sha256-${hash(1).slice(7)}.att`]),
    version(16, [`sha256-${hash(1).slice(7)}`]),
  ];
  const manifests = new Map(versions.map((value) => [value.name, image()]));
  manifests.set(hash(1), index(2));
  manifests.set(hash(2), index(3, 4));
  manifests.set(hash(13), { ...image(), subject: { digest: hash(1) }, artifactType: "test/sbom" });
  const pulls = [pull(19), pull(20), pull(27)];
  return { versions, pulls, manifests };
}

function fakeService(data = fixture()) {
  const deleted = [];
  let inventoryReads = 0;
  return {
    deleted,
    data,
    onInventory: () => {},
    async api(url, method = "GET") {
      if (method === "DELETE") {
        const id = Number(url.split("/").at(-1));
        if (this.failDelete) throw new Error("DELETE: HTTP 403 (package admin required)");
        deleted.push(id);
        data.versions = data.versions.filter((value) => value.id !== id);
        return new Response(null, { status: 204 });
      }
      if (url.includes("/pulls?")) return Response.json(data.pulls);
      this.onInventory(++inventoryReads, data);
      return Response.json(data.versions);
    },
    async manifest(value) { return data.manifests.get(value); },
    async registry() { return null; },
  };
}

test("retains latest main and latest PR roots with nested children, aliases and attachments", () => {
  const data = fixture();
  const plan = planRetention(data.versions, data.pulls, data.manifests);
  assert.deepEqual(plan.roots, [
    { name: "main/latest", digest: hash(1) },
    { name: "PR 19", digest: hash(8) },
    { name: "PR 20", digest: hash(9) },
  ]);
  assert.deepEqual(plan.remove.map((value) => value.id), [7, 10, 11, 12]);
  assert(plan.keep.some((value) => value.id === 14), "recursive attestation signature remains");
  assert(plan.keep.some((value) => value.id === 13), "subject-based referrer remains");
  assert.match(summary(plan, false), /dry-run/);
  assert.match(summary(plan, false), /outside retained image graphs/);
});

test("shared digest and multiple aliases are retained as a single package version", () => {
  const data = fixture();
  data.versions[0].metadata.container.tags.push("pr-20-44");
  data.versions = data.versions.filter((value) => value.id !== 9);
  const plan = planRetention(data.versions, [pull(20)], data.manifests);
  assert.equal(plan.roots[0].digest, plan.roots[1].digest);
  assert.equal(plan.keep.filter((value) => value.id === 1).length, 1);
});

test("preserves referrers returned by the OCI API and dependencies of attachments", () => {
  const data = fixture();
  data.manifests.set(hash(12), index(7));
  const plan = planRetention(data.versions, data.pulls, data.manifests, new Map([[hash(1), [hash(12)]]]));
  assert(plan.keep.some((value) => value.id === 12));
  assert(plan.keep.some((value) => value.id === 7));
});

test("rejects missing latest, missing protected children, unknown formats and ambiguous versions", () => {
  const data = fixture();
  assert.throws(() => planRetention(data.versions.slice(1), [], data.manifests), /exactly one.*latest/);
  assert.throws(() => planRetention(data.versions.filter((value) => value.id !== 3), [], data.manifests), /missing from package inventory/);
  assert.throws(() => planRetention([...data.versions, data.versions[0]], [], data.manifests), /Duplicate/);
  const manifests = new Map(data.manifests);
  manifests.set(hash(12), { mediaType: "unknown/manifest" });
  assert.throws(() => planRetention(data.versions, [], manifests), /Unsupported manifest type/);
  assert.throws(() => planRetention(data.versions, [], data.manifests, new Map([[hash(1), [hash(99)]]])), /missing from package inventory/);
});

test("fingerprint ignores list ordering but changes with aliases, roots or PR state", () => {
  const data = fixture();
  const fingerprint = planRetention(data.versions, data.pulls, data.manifests).fingerprint;
  assert.equal(planRetention([...data.versions].reverse(), data.pulls, data.manifests).fingerprint, fingerprint);
  assert.notEqual(planRetention(data.versions, [pull(19)], data.manifests).fingerprint, fingerprint);
  data.versions[0].metadata.container.tags.push("new-alias");
  assert.notEqual(planRetention(data.versions, data.pulls, data.manifests).fingerprint, fingerprint);
});

test("paginates complete inventories and rejects loops, invalid responses and foreign links", async () => {
  const path = "/users/dminca/packages/container/hephaestus-os/versions";
  const url = `https://api.github.com${path}?per_page=100`;
  const responses = [
    Response.json([1], { headers: { link: `<${url}&page=2>; rel="next"` } }),
    Response.json([2]),
  ];
  assert.deepEqual(await pages(async () => responses.shift(), url, "https://api.github.com", path), [1, 2]);
  for (const link of [
    `<${url}>; rel="next"`,
    '<https://example.com/versions>; rel="next"',
    '<https://api.github.com/another-endpoint>; rel="next"',
  ]) {
    await assert.rejects(pages(async () => Response.json([], { headers: { link } }), url, "https://api.github.com", path), /Pagination loop|Unsafe pagination/);
  }
  await assert.rejects(pages(async () => Response.json({ message: "no access" }), url, "https://api.github.com", path), /complete list/);
  const exactPage = Array.from({ length: 100 }, (_, i) => i);
  assert.equal((await pages(async () => Response.json(exactPage), url, "https://api.github.com", path)).length, 100);
});

test("dry-run verifies signatures and never deletes; failed replacement aborts all deletion", async () => {
  const service = fakeService();
  let verified = false;
  await execute(service, { verify: () => { verified = true; }, report: () => {} });
  assert(verified);
  assert.deepEqual(service.deleted, []);
  await assert.rejects(execute(service, {
    apply: true, verify: () => { throw new Error("Unsigned replacement"); }, report: () => {},
  }), /Unsigned replacement/);
  assert.deepEqual(service.deleted, []);
});

test("apply requires an unchanged reviewed fingerprint and deletes only stale versions", async () => {
  const service = fakeService();
  await assert.rejects(execute(service, { apply: true, expectedPlan: "wrong", verify: () => {}, report: () => {} }), /Reviewed plan changed/);
  assert.deepEqual(service.deleted, []);
  const data = fixture();
  const fingerprint = planRetention(data.versions, data.pulls, data.manifests).fingerprint;
  await execute(service, { apply: true, expectedPlan: fingerprint, verify: () => {}, report: () => {} });
  assert.deepEqual(service.deleted, [7, 10, 11, 12]);
});

test("aborts on concurrent package or PR changes before the first deletion", async () => {
  for (const mutate of [
    (data) => { data.pulls.pop(); },
    (data) => { data.versions[0].metadata.container.tags.push("concurrent-tag"); },
  ]) {
    const service = fakeService();
    service.onInventory = (read, data) => { if (read === 2) mutate(data); };
    await assert.rejects(execute(service, { apply: true, verify: () => {}, report: () => {} }), /state changed/);
    assert.deepEqual(service.deleted, []);
  }
});

test("deletion and discovery failures remain explicit; post-apply signature verification runs", async () => {
  const service = fakeService();
  service.failDelete = true;
  await assert.rejects(execute(service, { apply: true, verify: () => {}, report: () => {} }), /package admin required/);
  const discoveryFailure = fakeService();
  discoveryFailure.registry = async () => { throw new Error("referrers HTTP 403"); };
  await assert.rejects(execute(discoveryFailure, { apply: true, verify: () => {}, report: () => {} }), /referrers HTTP 403/);
  assert.deepEqual(discoveryFailure.deleted, []);
  const success = fakeService();
  let verifications = 0;
  await execute(success, { apply: true, verify: () => { verifications++; }, report: () => {} });
  assert.equal(verifications, 2);
});

test("OCI discovery paginates and recursively protects new referrers", async () => {
  const service = fakeService();
  service.registry = async (path) => {
    if (!path.includes(hash(1))) return null;
    if (path.includes("page=2")) return Response.json(index(7));
    return Response.json(index(12), {
      headers: { link: `<https://ghcr.io/v2/dminca/hephaestus-os/referrers/${hash(1)}?page=2>; rel="next"` },
    });
  };
  const plan = await execute(service, { verify: () => {}, report: () => {} });
  assert(plan.keep.some((value) => value.id === 12));
  assert(plan.keep.some((value) => value.id === 7));
  assert(!plan.remove.some((value) => value.id === 12));
});

test("incomplete or malformed discovery aborts before deletion", async () => {
  for (const response of [
    Response.json({ mediaType: "unknown/referrers", manifests: [] }),
    Response.json(index(99)),
  ]) {
    const service = fakeService();
    service.registry = async () => response.clone();
    await assert.rejects(execute(service, { apply: true, verify: () => {}, report: () => {} }), /Invalid referrers|missing from package inventory/);
    assert.deepEqual(service.deleted, []);
  }
});

test("client surfaces permission errors and verifies registry content digests", async () => {
  const document = JSON.stringify(image());
  const name = `sha256:${createHash("sha256").update(document).digest("hex")}`;
  const fetcher = async (url) => {
    if (url.includes("/token?")) return Response.json({ token: "fixture-token" });
    if (url.includes("/manifests/")) return new Response(document);
    if (url.includes("/referrers/")) return new Response(null, { status: 404 });
    return new Response(null, { status: 403 });
  };
  const service = await client("fixture-token", fetcher);
  assert.deepEqual(await service.manifest(name), image());
  await assert.rejects(service.manifest(hash(1)), /digest mismatch/);
  await assert.rejects(service.api("/test"), /HTTP 403/);
  assert.equal(await service.registry("/referrers/test", true), null);
});

test("cleanup is trusted, serialized, dry-run by default, and manual apply needs review", () => {
  const workflow = readFileSync(".github/workflows/container-retention.yml", "utf8");
  assert.match(workflow, /pull_request_target:\s+types: \[closed\]/);
  assert.match(workflow, /ref: main/);
  assert.match(workflow, /persist-credentials: false/);
  assert.match(workflow, /group: ghcr-hephaestus-os/);
  assert.match(workflow, /cancel-in-progress: false/);
  assert.match(workflow, /CONTAINER_RETENTION_AUTO_APPLY/);
  assert.match(workflow, /--apply --expected-plan "\$EXPECTED_PLAN"/);
  assert.doesNotMatch(workflow, /pull_request\.head|workflow_run\.head_sha|download-artifact/);
});
