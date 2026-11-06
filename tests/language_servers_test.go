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
package.preload["blink.cmp"] = function()
  return { get_lsp_capabilities = function() return { completion = true } end }
end
local configured = false
package.preload["mason-lspconfig"] = function()
  return { setup = function(options)
    configured = true
    assert(vim.list_contains(options.ensure_installed, "yamlls"))
    assert(vim.list_contains(options.ensure_installed, "hls"))
    assert(vim.deep_equal(options.ensure_installed, options.automatic_enable))
    assert(not vim.list_contains(options.automatic_enable, "rust_analyzer"),
      "rustaceanvim owns Rust LSP startup")
    for _, name in ipairs(options.automatic_enable) do
      assert(vim.lsp.config[name].capabilities.completion)
    end
  end }
end
vim.lsp.config = setmetatable({}, {
  __call = function(self, name, config) rawset(self, name, config) end,
})
dofile(vim.env.HEPHAESTUS_LANGUAGES)
assert(configured)
assert(vim.lsp.config.gopls.settings.gopls.gofumpt)
assert(vim.lsp.config.pyright.settings.pyright.disableOrganizeImports)
assert(vim.lsp.config.yamlls.settings.yaml.format.enable)
assert(vim.lsp.config.yamlls.settings.yaml.kubernetesCRDStore.enable)
io.stdout:write("User language-server configuration passed\n")
`, 0o600)
	command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
	command.Env = append(os.Environ(),
		"HEPHAESTUS_LANGUAGES="+filepath.Join(repository(t),
			"files/system/usr/share/hephaestus/profile/config/nvim/lua/hephaestus/languages.lua"))
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "User language-server configuration passed") {
		t.Fatalf("user language-server configuration failed: %v\n%s", err, output)
	}
}
