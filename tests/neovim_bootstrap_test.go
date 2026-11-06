package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserPluginBootstrap(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required for bootstrap tests")
	}
	for _, mode := range []string{"fresh", "existing", "linked", "failed"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			config := filepath.Join(repository(t), "files/system/usr/share/hephaestus/profile/config/nvim")
			if mode == "linked" {
				link := filepath.Join(directory, "config")
				if err := os.Symlink(config, link); err != nil {
					t.Fatal(err)
				}
				config = link
			}
			driver := filepath.Join(directory, "driver.lua")
			writeFile(t, driver, `
local mode = vim.env.HEPHAESTUS_TEST_MODE
local cloned, configured = false, false
local lazy = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
local lock = vim.fn.stdpath("state") .. "/lazy-lock.json"
vim.fn.mkdir(vim.fn.stdpath("state"), "p")
if mode == "existing" then
  vim.fn.mkdir(lazy, "p")
  vim.fn.writefile({ '{"user-choice":{}}' }, lock)
end
vim.system = function(args)
  assert(args[1] == "git" and args[#args] == lazy)
  cloned = true
  return { wait = function()
    if mode ~= "failed" then vim.fn.mkdir(lazy, "p") end
    return { code = mode == "failed" and 1 or 0, stderr = "network unavailable" }
  end }
end
local messages = {}
vim.notify = function(message) messages[#messages + 1] = message end
package.preload["lazy"] = function()
  return { setup = function(specs, options)
    configured = true
    assert(options.install.missing)
    assert(options.lockfile == lock)
    assert(specs[1][1] and not specs[1].dir)
    assert(vim.fn.filereadable(lock) == 1)
  end }
end
dofile(vim.env.HEPHAESTUS_TEST_CONFIG .. "/init.lua")
if mode == "failed" then
  assert(not configured and #messages == 1 and messages[1]:find("Restart Neovim"))
else
  assert(configured and #messages == 0)
  assert(cloned == (mode == "fresh" or mode == "linked"))
  if mode == "existing" then
    assert(vim.fn.readfile(lock)[1] == '{"user-choice":{}}', "user lock was overwritten")
  end
end
io.stdout:write("User plugin bootstrap passed\n")
`, 0o600)
			command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
			command.Env = append(os.Environ(),
				"HEPHAESTUS_TEST_MODE="+mode, "HEPHAESTUS_TEST_CONFIG="+config,
				"XDG_DATA_HOME="+filepath.Join(directory, "data"),
				"XDG_STATE_HOME="+filepath.Join(directory, "state"),
				"XDG_CACHE_HOME="+filepath.Join(directory, "cache"))
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "User plugin bootstrap passed") {
				t.Fatalf("plugin bootstrap failed: %v\n%s", err, output)
			}
		})
	}
}
