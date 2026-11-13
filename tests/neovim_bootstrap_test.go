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
	for _, mode := range []string{"fresh", "existing", "linked", "failed", "old-version"} {
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
if mode == "old-version" then
  local has = vim.fn.has
  vim.fn.has = function(feature) return feature == "nvim-0.12" and 0 or has(feature) end
end
local added, configured = false, false
local lock = vim.fn.stdpath("config") .. "/nvim-pack-lock.json"
vim.fn.mkdir(vim.fn.stdpath("config"), "p")
if mode == "existing" then vim.fn.writefile({ '{"user-choice":{}}' }, lock) end
local hooks
local create = vim.api.nvim_create_autocmd
vim.api.nvim_create_autocmd = function(event, opts)
  if event == "PackChanged" then hooks = opts.callback end
  return create(event, opts)
end
vim.pack.add = function(specs, opts)
  assert(hooks, "build hooks must exist before first native package call")
  assert(opts.confirm == false and opts.load == true)
  local names = {}
  for _, spec in ipairs(specs) do
    assert(spec.src:match("^https://github.com/"))
    assert(spec.version and not spec.dir)
    local name = spec.name or spec.src:match("[^/]+$")
    assert(not names[name], "duplicate plugin " .. name)
    names[name] = true
  end
  assert(names.catppuccin and names["nvim-lspconfig"] and not names["mason-lspconfig.nvim"])
  assert(vim.g.mkdp_open_to_the_world == 0 and vim.g.firenvim_config)
  added = true
  if mode == "failed" then error("network unavailable") end
end
package.preload["hephaestus.plugins"] = function() configured = true; return {} end
local ok, err = pcall(dofile, vim.env.HEPHAESTUS_TEST_CONFIG .. "/init.lua")
if mode == "old-version" then
  assert(not ok and tostring(err):find("requires Neovim 0.12") and not added and not configured)
elseif mode == "failed" then
  assert(not ok and tostring(err):find("network unavailable") and not configured)
else
  assert(ok, err)
  assert(added and configured)
  if mode == "existing" then
    assert(vim.fn.readfile(lock)[1] == '{"user-choice":{}}', "user lock was overwritten")
  end
  local builds = 0
  vim.system = function(args, opts)
    assert(args[1] == "make" and opts.cwd == "/plugin")
    builds = builds + 1
    return { wait = function() return { code = 1, stderr = "build unavailable" } end }
  end
  hooks({ data = { spec = { name = "codesnap.nvim" }, kind = "delete", path = "/plugin" } })
  assert(builds == 0)
  local built, failure = pcall(hooks, {
    data = { spec = { name = "codesnap.nvim" }, kind = "update", path = "/plugin" },
  })
  assert(not built and tostring(failure):find("CodeSnap build failed") and builds == 1)
end
io.stdout:write("User plugin bootstrap passed\n")
`, 0o600)
			command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
			command.Env = append(os.Environ(),
				"HEPHAESTUS_TEST_MODE="+mode, "HEPHAESTUS_TEST_CONFIG="+config,
				"XDG_CONFIG_HOME="+filepath.Join(directory, "xdg-config"),
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
