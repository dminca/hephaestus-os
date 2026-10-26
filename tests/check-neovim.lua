local root = assert(vim.env.HEPHAESTUS_REPO, "Set HEPHAESTUS_REPO to the checkout")
local errors = {}
local notify = vim.notify
vim.notify = function(message, level, opts)
  if level == vim.log.levels.ERROR then table.insert(errors, tostring(message)) end
  notify(message, level, opts)
end
local config = root .. "/files/system/usr/share/hephaestus/profile/config/nvim"
for _, file in ipairs(vim.fn.glob(config .. "/**/*.lua", false, true)) do
  assert(loadfile(file))
end
dofile(config .. "/init.lua")
vim.api.nvim_exec_autocmds("VimEnter", {})
vim.wait(1500)
assert(vim.g.mapleader == " ")
assert(vim.o.tabstop == 2 and vim.o.shiftwidth == 2)
assert(vim.o.relativenumber and vim.o.undofile and not vim.o.swapfile)
assert(vim.fn.maparg("jk", "i") == "<Esc>")
assert(vim.fn.maparg("<leader>gg", "n") ~= "")
assert(vim.fn.maparg("<leader>ff", "n") ~= "")
assert(vim.fn.maparg("<leader>lf", "n") ~= "")
assert(vim.fn.maparg("<leader>gs", "n") ~= "")
assert(vim.fn.exists(":DiffviewOpen") == 2)
assert(vim.fn.exists(":AutoSession") == 2)
assert(require("lazy.core.config").options.install.missing == false)
assert(not require("lazy.core.config").options.checker.enabled)
assert(require("conform").list_formatters_for_buffer(0))
assert(vim.lsp.config.gopls.settings.gopls.gofumpt)
assert(vim.lsp.config.pyright.settings.pyright.disableOrganizeImports)
assert(vim.lsp.config.yamlls.settings.yaml.kubernetesCRDStore.enable)
assert(#errors == 0, table.concat(errors, "\n"))
assert(vim.v.errmsg == "", vim.v.errmsg)
io.stdout:write("Native nvix startup, options, mappings, plugins, LSP settings, and offline policy passed\n")
vim.cmd("qa!")
