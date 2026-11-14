package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateSystemProvenanceMatchesBuild(t *testing.T) {
	root := repository(t)
	verifier := readFile(t, filepath.Join(root, "files/system/usr/libexec/secureblue/verify-provenance"))
	for _, expected := range []string{
		"ghcr.io/dminca/hephaestus-os:*",
		"expected_source='https://github.com/dminca/hephaestus-os'",
		"latest)\n    branch='main'",
		"--level 3",
		"--param \"trusted_builders:[${expected_source}/.github/workflows/build.yml@refs/heads/${branch}]\"",
	} {
		if !strings.Contains(verifier, expected) {
			t.Errorf("provenance verifier is missing %q", expected)
		}
	}

	workflow := readFile(t, filepath.Join(root, ".github/workflows/build.yml"))
	for _, expected := range []string{
		"IMAGE_REF: ${{ steps.image_manifest_metadata.outputs.IMAGE_REF }}",
		"IMAGE_DIGEST: ${{ steps.image_manifest_metadata.outputs.IMAGE_DIGEST }}",
		"slsa-framework/actions/.github/workflows/attest_actions.yml@v0.2.0",
		"watch-jobs: bluebuild",
		"artifact-name: slsa-provenance-hephaestus-os",
		"oras push \"${artifact_ref}\" provenance.intoto.json:application/vnd.dsse.envelope.v1+json",
	} {
		if !strings.Contains(workflow, expected) {
			t.Errorf("build workflow is missing %q", expected)
		}
	}
}
