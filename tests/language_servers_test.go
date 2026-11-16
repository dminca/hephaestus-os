package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserLanguageServers(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required for LSP configuration tests")
	}
	driver := filepath.Join(t.TempDir(), "driver.lua")
	writeFile(t, driver, `
vim.opt.runtimepath:prepend(vim.env.HEPHAESTUS_CONFIG)
package.preload["blink.cmp"] = function()
  return { get_lsp_capabilities = function() return { completion = true } end }
end
local installed, enabled, messages = {}, {}, {}
vim.notify = function(message) messages[#messages + 1] = message end
vim.fn.executable = function(name) return installed[name] and 1 or 0 end
local options
package.preload["mason-tool-installer"] = function()
  return { setup = function(opts) options = opts end, run_on_start = function() end }
end
local events = {}
package.preload["mason-registry"] = function()
  return { on = function(_, event, callback) events[event] = callback end }
end
package.preload["mason-lspconfig"] = function() error("mason-lspconfig must not be used") end
vim.lsp.config = setmetatable({}, {
  __call = function(self, name, config)
    config.cmd = config.cmd or { name }
    rawset(self, name, config)
  end,
})
vim.lsp.enable = function(name)
  assert(vim.lsp.config[name].capabilities.completion)
  enabled[name] = true
end
dofile(vim.env.HEPHAESTUS_CONFIG .. "/lua/hephaestus/languages.lua")
assert(options and options.auto_update == false and options.run_on_start)
assert(options.integrations["mason-lspconfig"] == false)
local tools = {}
for _, tool in ipairs(options.ensure_installed) do
  assert(not tools[tool[1]], "duplicate Mason package")
  tools[tool[1]] = tool
end
assert(tools["marksman"].condition())
assert(not tools["gopls"].condition() and not tools["haskell-language-server"].condition())
assert(not tools["yaml-language-server"].condition() and not tools["nil"].condition())
assert(not tools["goimports"].condition() and not tools["fixjson"].condition())
assert(#messages >= 6 and messages[1]:find("missing"))
for _, binary in ipairs({ "go", "node", "npm", "ghcup", "ghc", "cargo", "rustc", "cc" }) do
  installed[binary] = true
end
for _, tool in pairs(tools) do assert(tool.condition(), tool[1]) end
assert(vim.tbl_isempty(enabled), "missing servers must not be enabled")
installed.gopls, installed["yaml-language-server"], installed.hls = true, true, true
vim.api.nvim_exec_autocmds("User", { pattern = "MasonToolsUpdateCompleted" })
vim.wait(100, function() return enabled.gopls end)
assert(enabled.gopls and enabled.yamlls and enabled.hls)
installed.svelteserver = true
events["package:install:success"]()
vim.wait(100, function() return enabled.svelte end)
assert(enabled.svelte, "manual Mason installs must enable ready servers too")
assert(not enabled.rust_analyzer, "rustaceanvim owns Rust LSP startup")
assert(vim.lsp.config.gopls.settings.gopls.gofumpt)
assert(vim.lsp.config.pyright.settings.pyright.disableOrganizeImports)
assert(vim.lsp.config.yamlls.settings.yaml.format.enable)
assert(vim.lsp.config.yamlls.settings.yaml.kubernetesCRDStore.enable)
events["package:install:failed"]({ name = "marksman" }, "download failed")
vim.wait(50)
assert(messages[#messages]:find("marksman installation failed"))
io.stdout:write("User language-server configuration passed\n")
`, 0o600)
	command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
	command.Env = append(os.Environ(),
		"HEPHAESTUS_CONFIG="+filepath.Join(repository(t),
			"files/system/usr/share/hephaestus/profile/config/nvim"))
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "User language-server configuration passed") {
		t.Fatalf("user language-server configuration failed: %v\n%s", err, output)
	}
}
