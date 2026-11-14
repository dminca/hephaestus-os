package tests

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
	var build struct {
		Jobs map[string]struct {
			Uses string `yaml:"uses"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(workflow), &build); err != nil {
		t.Fatalf("parse build workflow: %v", err)
	}
	reference := build.Jobs["provenance"].Uses
	if !regexp.MustCompile(`^slsa-framework/actions/\.github/workflows/attest_actions\.yml@(v[0-9]+\.[0-9]+\.[0-9]+|[0-9a-f]{40})$`).MatchString(reference) {
		t.Errorf("provenance job must use the SLSA attestation workflow at a release tag or full commit SHA, got %q", reference)
	}
	for _, expected := range []string{
		"IMAGE_REF: ${{ steps.image_manifest_metadata.outputs.IMAGE_REF }}",
		"IMAGE_DIGEST: ${{ steps.image_manifest_metadata.outputs.IMAGE_DIGEST }}",
		"watch-jobs: bluebuild",
		"artifact-name: slsa-provenance-hephaestus-os",
		"oras push \"${artifact_ref}\" provenance.intoto.json:application/vnd.dsse.envelope.v1+json",
	} {
		if !strings.Contains(workflow, expected) {
			t.Errorf("build workflow is missing %q", expected)
		}
	}
}
