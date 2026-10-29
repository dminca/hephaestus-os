package tests

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const flowStub = `#!/usr/bin/env bash
set -euo pipefail
command=$(basename "$0")
printf '%s %s\n' "$command" "$*" >> "$CALL_LOG"
case "$command" in
  uname)
    if [[ "$1" == -s ]]; then
      printf 'Linux\n'
    else
      printf 'x86_64\n'
    fi
    ;;
  curl)
    url=
    output_file=
    while [[ $# -gt 0 ]]; do
      case "$1" in
        https://*) url=$1 ;;
        -o)
          shift
          output_file=$1
          ;;
      esac
      shift
    done
    if [[ "$url" == *.sig ]]; then
      printf 'c2lnbmF0dXJl' > "$output_file"
    else
      printf 'fixture RPM' > "$output_file"
    fi
    ;;
  rpm)
    if [[ "$1" == -qp ]]; then
      printf 'flow-beta:%s:1:x86_64\n' "$FLOW_RPM_VERSION"
    else
      printf '%s:1:x86_64\n' "$FLOW_RPM_VERSION"
    fi
    ;;
  minisign)
    if [[ "${FAIL_SIGNATURE:-}" == 1 ]]; then exit 1; fi
    ;;
  dnf5)
    if [[ "${FAIL_DNF:-}" == 1 ]]; then exit 1; fi
    ;;
  gst-inspect-1.0)
    if [[ "${FAIL_CODEC:-}" == 1 ]]; then
      printf 'No such element or plugin avdec_h264\n' >&2
      exit 1
    fi
    ;;
esac
`

type flowBuildOptions struct {
	corruptChecksum bool
	environment     []string
	version         string
}

func runFlowBuild(t *testing.T, options flowBuildOptions) (output, calls string, success bool) {
	t.Helper()
	root := t.TempDir()
	stub := filepath.Join(root, "stub")
	writeFile(t, stub, flowStub, 0o755)
	for _, command := range []string{"uname", "curl", "rpm", "minisign", "dnf5", "gst-inspect-1.0"} {
		if err := os.Symlink(stub, filepath.Join(root, command)); err != nil {
			t.Fatal(err)
		}
	}
	script := readFile(t, filepath.Join(repository(t), "files/scripts/install-flow-desktop.sh"))
	versionPattern := regexp.MustCompile(`(?m)^version=v([^\s]+)`)
	versionMatch := versionPattern.FindStringSubmatch(script)
	if versionMatch == nil {
		t.Fatal("expected a pinned Flow release version")
	}
	rpmVersion := versionMatch[1]
	if options.version != "" {
		rpmVersion = options.version
		script = versionPattern.ReplaceAllLiteralString(script, "version=v"+rpmVersion)
	}
	if !options.corruptChecksum {
		checksum := fmt.Sprintf("checksum=%x", sha256.Sum256([]byte("fixture RPM")))
		pattern := regexp.MustCompile(`checksum=[a-f0-9]{64}`)
		if len(pattern.FindAllString(script, -1)) != 1 {
			t.Fatal("expected exactly one pinned RPM checksum")
		}
		script = pattern.ReplaceAllString(script, checksum)
	}
	// Keep installation paths inside the disposable fixture, never on the host.
	writeFile(t, filepath.Join(root, "flow"), "", 0o755)
	writeFile(t, filepath.Join(root, "Flow Beta.desktop"), "", 0o644)
	script = strings.ReplaceAll(script, "/usr/bin/flow", filepath.Join(root, "flow"))
	script = strings.ReplaceAll(script, "/usr/share/applications/Flow Beta.desktop", filepath.Join(root, "Flow Beta.desktop"))
	log := filepath.Join(root, "calls")
	command := exec.Command("bash", "-c", script)
	command.Env = append(os.Environ(),
		"PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"),
		"CALL_LOG="+log, "FAIL_SIGNATURE=", "FAIL_DNF=", "FAIL_CODEC=",
		"FLOW_RPM_VERSION="+rpmVersion,
	)
	command.Env = append(command.Env, options.environment...)
	output, success = runCommand(t, command)
	return output, readFile(t, log), success
}

func assertCommandCalled(t *testing.T, calls, command string) {
	t.Helper()
	if !strings.Contains(calls, command+" ") {
		t.Fatalf("expected %s to run; command log:\n%s", command, calls)
	}
}

func assertCommandNotCalled(t *testing.T, calls, command string) {
	t.Helper()
	if strings.Contains(calls, command+" ") {
		t.Fatalf("%s should not run; command log:\n%s", command, calls)
	}
}

func TestFlowDesktop(t *testing.T) {
	t.Run("UpdatedReleaseInstalled", func(t *testing.T) {
		output, calls, ok := runFlowBuild(t, flowBuildOptions{version: "0.1.0-beta3"})
		if !ok {
			t.Fatal(output)
		}
		assertCommandCalled(t, calls, "dnf5")
		assertCommandCalled(t, calls, "gst-inspect-1.0")
		if !strings.Contains(calls, "Flow_0.1.0-beta3_linux_x86_64.rpm") {
			t.Fatalf("expected updated release package:\n%s", calls)
		}
	})

	t.Run("VerifiedPackageInstalledWithoutTrustOverrides", func(t *testing.T) {
		output, calls, ok := runFlowBuild(t, flowBuildOptions{})
		if !ok {
			t.Fatal(output)
		}
		assertCommandCalled(t, calls, "minisign")
		assertCommandCalled(t, calls, "dnf5")
		signaturePosition := strings.Index(calls, "minisign ")
		installationPosition := strings.Index(calls, "dnf5 ")
		if signaturePosition >= installationPosition {
			t.Fatalf("signature must be verified before installation:\n%s", calls)
		}
		for _, required := range []string{"--setopt=install_weak_deps=False", "gst-inspect-1.0 avdec_h264"} {
			if !strings.Contains(calls, required) {
				t.Fatalf("missing %q:\n%s", required, calls)
			}
		}
		script := readFile(t, filepath.Join(repository(t), "files/scripts/install-flow-desktop.sh"))
		for _, override := range []string{"--nogpgcheck", "--enablerepo", "gpgcheck=0", "localpkg_gpgcheck"} {
			if strings.Contains(script, override) {
				t.Fatalf("unexpected trust override: %s", override)
			}
		}
	})

	t.Run("ChecksumFailurePreventsInstallation", func(t *testing.T) {
		output, calls, ok := runFlowBuild(t, flowBuildOptions{corruptChecksum: true})
		if ok {
			t.Fatalf("build succeeded despite checksum failure:\n%s", output)
		}
		assertCommandCalled(t, calls, "curl")
		assertCommandNotCalled(t, calls, "minisign")
		assertCommandNotCalled(t, calls, "dnf5")
	})

	t.Run("SignatureFailurePreventsInstallation", func(t *testing.T) {
		output, calls, ok := runFlowBuild(t, flowBuildOptions{environment: []string{"FAIL_SIGNATURE=1"}})
		if ok {
			t.Fatalf("build succeeded despite signature failure:\n%s", output)
		}
		assertCommandCalled(t, calls, "minisign")
		assertCommandNotCalled(t, calls, "dnf5")
	})

	t.Run("DependencyFailureStopsBuild", func(t *testing.T) {
		output, calls, ok := runFlowBuild(t, flowBuildOptions{environment: []string{"FAIL_DNF=1"}})
		if ok {
			t.Fatalf("build succeeded despite dependency failure:\n%s", output)
		}
		assertCommandCalled(t, calls, "dnf5")
		assertCommandNotCalled(t, calls, "gst-inspect-1.0")
	})

	t.Run("MissingDecoderFailsExplicitly", func(t *testing.T) {
		output, _, ok := runFlowBuild(t, flowBuildOptions{environment: []string{"FAIL_CODEC=1"}})
		if ok {
			t.Fatalf("build succeeded without a decoder:\n%s", output)
		}
		if !strings.Contains(output, "requires an available H.264 decoder") {
			t.Fatalf("expected explicit decoder error:\n%s", output)
		}
	})
}

func TestFlowDesktopPinUpdater(t *testing.T) {
	for _, failSignature := range []bool{false, true} {
		t.Run(fmt.Sprintf("signature-failure-%t", failSignature), func(t *testing.T) {
			root := t.TempDir()
			script := `version=v0.1.0-beta2 # renovate: datasource=github-releases depName=Flow-Tube/Flow-Desktop
checksum=` + strings.Repeat("a", 64) + `
public_key=RWQpaRLd6yhk18G30wcoSsrlLXi8bjGYFcC4gMCiYs/Ms5xKMe3P7qpR
`
			mkdir(t, filepath.Join(root, "files/scripts"))
			mkdir(t, filepath.Join(root, "tools"))
			writeFile(t, filepath.Join(root, "files/scripts/install-flow-desktop.sh"), script, 0o644)
			updateScript := readFile(t, filepath.Join(repository(t), "tools/update-flow-desktop-pin.sh"))
			writeFile(t, filepath.Join(root, "tools/update-flow-desktop-pin.sh"), updateScript, 0o644)

			bin := filepath.Join(root, "bin")
			mkdir(t, bin)
			writeFile(t, filepath.Join(bin, "curl"), `#!/usr/bin/env bash
set -euo pipefail
url=
output=
while [[ $# -gt 0 ]]; do
  case "$1" in
    https://*) url=$1 ;;
    -o) shift; output=$1 ;;
  esac
  shift
done
if [[ "$url" == *.sig ]]; then
  printf 'c2lnbmF0dXJl' > "$output"
else
  printf 'fixture RPM' > "$output"
fi
`, 0o755)
			minisign := "#!/usr/bin/env bash\n[[ \"${FAIL_SIGNATURE:-false}\" != true ]]\n"
			writeFile(t, filepath.Join(bin, "minisign"), minisign, 0o755)
			command := exec.Command("bash", "tools/update-flow-desktop-pin.sh")
			command.Dir = root
			command.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				fmt.Sprintf("FAIL_SIGNATURE=%t", failSignature),
			)
			output, ok := runCommand(t, command)
			if failSignature {
				if ok {
					t.Fatalf("invalid signature was accepted: %s", output)
				}
				if !strings.Contains(readFile(t, filepath.Join(root, "files/scripts/install-flow-desktop.sh")), strings.Repeat("a", 64)) {
					t.Fatal("signature failure modified the pinned checksum")
				}
				return
			}
			if !ok {
				t.Fatal(output)
			}
			want := fmt.Sprintf("%x", sha256.Sum256([]byte("fixture RPM")))
			if !strings.Contains(readFile(t, filepath.Join(root, "files/scripts/install-flow-desktop.sh")), "checksum="+want) {
				t.Fatal("verified release checksum was not written")
			}
		})
	}
}
