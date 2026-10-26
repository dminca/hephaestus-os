package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserBuildReportsCompleteFailures(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required; the Fedora parser CI job runs these checks")
	}
	versionCheck := exec.Command(nvim, "--headless", "-u", "NONE",
		"+lua if vim.fn.has('nvim-0.11') == 0 then vim.cmd('cquit') end", "+qa!")
	if err := versionCheck.Run(); err != nil {
		t.Skip("Neovim 0.11 or newer is required; the Fedora parser CI job runs these checks")
	}
	script := filepath.Join(repository(t), "files/system/usr/share/hephaestus/build-parsers.lua")
	fixtures := []struct {
		name    string
		mode    string
		message string
	}{
		{name: "success", mode: "success", message: "All 42 native Treesitter parsers built"},
		{name: "unknown parsers", mode: "unknown", message: "Unavailable parsers in pinned plugin: bash, c"},
		{name: "unsupported parser", mode: "unsupported", message: "Unavailable parsers in pinned plugin: bash"},
		{name: "missing parsers", mode: "missing", message: "bash: parser file missing\nc: parser file missing"},
		{name: "unloadable parser", mode: "unloadable", message: "bash: incompatible parser ABI"},
		{name: "parser rejected", mode: "rejected", message: "bash: false"},
		{name: "installer failure", mode: "failed", message: "Treesitter installation reported a failure"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			driver := filepath.Join(t.TempDir(), "driver.lua")
			content := `
local mode = vim.env.HEPHAESTUS_TEST_MODE
local languages
package.preload["nvim-treesitter.parsers"] = function()
  return setmetatable({}, { __index = function(_, language)
    if mode == "unknown" and (language == "bash" or language == "c") then return nil end
    return { tier = mode == "unsupported" and language == "bash" and 4 or 1 }
  end })
end
package.preload["nvim-treesitter"] = function()
  return {
    setup = function() end,
    install = function(requested, options)
      assert(mode ~= "unknown" and mode ~= "unsupported", "invalid parsers reached installation")
      assert(options.summary and options.max_jobs == 4, "missing bounded build settings")
      for _, language in ipairs(requested) do
        assert(language ~= "jsonc", "JSONC is a filetype, not a parser")
      end
      languages = requested
      return { wait = function(_, timeout)
        assert(timeout == 600000)
        return mode ~= "failed"
      end }
    end,
  }
end
vim.fn.filereadable = function(filename)
  if mode == "missing" and (filename:match("/bash%.so$") or filename:match("/c%.so$")) then return 0 end
  return 1
end
vim.treesitter.language.add = function(language)
  assert(vim.list_contains(languages, language), "unrequested parser checked")
  if mode == "unloadable" and language == "bash" then error("incompatible parser ABI", 0) end
  if mode == "rejected" and language == "bash" then return false end
  return true
end
dofile(vim.env.HEPHAESTUS_TEST_SCRIPT)
`
			if err := os.WriteFile(driver, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
			command.Env = append(os.Environ(),
				"HEPHAESTUS_TEST_MODE="+fixture.mode,
				"HEPHAESTUS_TEST_SCRIPT="+script,
			)
			output, err := command.CombinedOutput()
			if fixture.mode == "success" && err != nil {
				t.Fatalf("parser build failed: %v\n%s", err, output)
			}
			if fixture.mode != "success" && err == nil {
				t.Fatalf("parser failure did not stop the build:\n%s", output)
			}
			normalized := strings.ReplaceAll(string(output), "\r\n", "\n")
			if !strings.Contains(normalized, fixture.message) {
				t.Fatalf("expected %q, got:\n%s", fixture.message, output)
			}
		})
	}
}
