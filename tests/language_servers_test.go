package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLanguageServerBuildValidation(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required for LSP validation tests")
	}

	script := filepath.Join(repository(t), "files/system/usr/share/hephaestus/check-language-servers.lua")
	for _, mode := range []string{"installed", "missing"} {
		t.Run(mode, func(t *testing.T) {
			driver := filepath.Join(t.TempDir(), "driver.lua")
			content := `
vim.g.hephaestus_language_server_commands = {
  lua_ls = "lua-language-server",
  nil_ls = "nil",
}
vim.fn.executable = function() return vim.env.HEPHAESTUS_LSP_TEST == "installed" and 1 or 0 end
dofile(vim.env.HEPHAESTUS_LSP_SCRIPT)
`
			if err := os.WriteFile(driver, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
			command.Env = append(os.Environ(), "HEPHAESTUS_LSP_TEST="+mode, "HEPHAESTUS_LSP_SCRIPT="+script)
			output, err := command.CombinedOutput()
			if mode == "installed" {
				if err != nil || !strings.Contains(string(output), "All 2 configured") {
					t.Fatalf("expected successful validation: %v\n%s", err, output)
				}
			} else if err == nil || !strings.Contains(string(output), "lua_ls: missing lua-language-server") ||
				!strings.Contains(string(output), "nil_ls: missing nil") {
				t.Fatalf("expected every missing server to be reported: %v\n%s", err, output)
			}
		})
	}
}

func TestImageChecksLanguageServersAfterInstallation(t *testing.T) {
	root := repository(t)
	dnf := readRecipeModule(t, "dnf")
	if !slices.Contains(dnf.Install.Packages, "nix-core") {
		t.Fatal("nil's builtin generation requires the nix-core package")
	}

	script := readFile(t, filepath.Join(root, "files/scripts/install-profile.sh"))
	check := strings.Index(script, `-l "$assets/check-language-servers.lua"`)
	if check < 0 {
		t.Fatal("image build must check the installed Neovim language servers")
	}
	for _, install := range []string{
		`"$temporary/hephaestus-assets" binaries`,
		"npm ci",
		"--rev dbbb5c24f9215476992aa1bd305f8e5398e89654",
		"--rev d324490e45bcda1b664f28b6f88da8580deda76b",
	} {
		position := strings.Index(script, install)
		if position < 0 || position >= check {
			t.Fatalf("missing installation before validation: %s", install)
		}
	}
}

func TestLanguageServerFunctionCommands(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required for LSP configuration tests")
	}
	root := repository(t)
	driver := filepath.Join(t.TempDir(), "driver.lua")
	writeFile(t, driver, `
local dynamic = {
  ts_ls = "typescript-language-server", tailwindcss = "tailwindcss-language-server",
  jsonls = "vscode-json-language-server", html = "vscode-html-language-server",
  eslint = "vscode-eslint-language-server", cssls = "vscode-css-language-server",
  biome = "biome", yamlls = "yaml-language-server", svelte = "svelteserver",
}
package.preload["blink.cmp"] = function()
  return { get_lsp_capabilities = function() return {} end }
end
vim.lsp.config = setmetatable({}, {
  __call = function(self, name, config)
    config.cmd = dynamic[name] and function() end or { name }
    rawset(self, name, config)
  end,
})
vim.lsp.enable = function() end
vim.fn.executable = function() return 1 end
dofile(vim.env.HEPHAESTUS_LANGUAGES)
for name, command in pairs(dynamic) do
  assert(vim.g.hephaestus_language_server_commands[name] == command,
    "Wrong command for " .. name)
end
dofile(vim.env.HEPHAESTUS_LSP_SCRIPT)
`, 0o600)
	command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
	command.Env = append(os.Environ(),
		"HEPHAESTUS_LANGUAGES="+filepath.Join(root, "files/system/usr/share/hephaestus/profile/config/nvim/lua/hephaestus/languages.lua"),
		"HEPHAESTUS_LSP_SCRIPT="+filepath.Join(root, "files/system/usr/share/hephaestus/check-language-servers.lua"),
	)
	if output, ok := runCommand(t, command); !ok || !strings.Contains(output, "configured language server commands are installed") {
		t.Fatalf("function-valued commands must resolve before validation:\n%s", output)
	}
}
