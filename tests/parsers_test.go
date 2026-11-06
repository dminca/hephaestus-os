package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserParserInstallation(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim is required for parser installation tests")
	}
	for _, mode := range []string{"success", "failed", "delayed"} {
		t.Run(mode, func(t *testing.T) {
			driver := filepath.Join(t.TempDir(), "driver.lua")
			writeFile(t, driver, `
local mode = vim.env.HEPHAESTUS_TEST_MODE
local ready = mode ~= "delayed"
local installs, starts = 0, 0
local notifications = {}
vim.notify = function(message, level)
  assert(level == vim.log.levels.ERROR)
  notifications[#notifications + 1] = message
end
vim.fn.executable = function() return ready and 1 or 0 end
vim.fn.filereadable = function(path)
  assert(path:find(vim.fn.stdpath("data"), 1, true), "parser path is not user-owned")
  return 1
end
vim.treesitter.language.get_lang = function() return "lua" end
vim.treesitter.start = function() starts = starts + 1 end
package.preload["nvim-treesitter"] = function()
  return {
    setup = function(options)
      assert(options.install_dir == vim.fn.stdpath("data") .. "/site")
    end,
    install = function(languages, options)
      installs = installs + 1
      assert(#languages == 42 and options.max_jobs == 4)
      return { await = function(_, callback)
        vim.defer_fn(function() callback(nil, mode ~= "failed") end, 10)
      end }
    end,
  }
end
dofile(vim.env.HEPHAESTUS_TEST_SCRIPT).setup()
vim.wait(30)
if mode == "delayed" then
  assert(installs == 0, "installation started before tree-sitter CLI was ready")
  ready = true
  vim.api.nvim_exec_autocmds("User", { pattern = "MasonToolsUpdateCompleted" })
  vim.wait(30)
end
assert(installs == 1)
if mode == "failed" then
  assert(#notifications == 1 and notifications[1]:find(":TSInstall"))
else
  assert(#notifications == 0 and starts > 0)
end
io.stdout:write("User parser installation passed\n")
`, 0o600)
			command := exec.Command(nvim, "--headless", "-u", "NONE", "-l", driver)
			command.Env = append(os.Environ(),
				"HEPHAESTUS_TEST_MODE="+mode,
				"HEPHAESTUS_TEST_SCRIPT="+filepath.Join(repository(t), "files/system/usr/share/hephaestus/profile/config/nvim/lua/hephaestus/treesitter.lua"))
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "User parser installation passed") {
				t.Fatalf("parser installation failed: %v\n%s", err, output)
			}
		})
	}
}
