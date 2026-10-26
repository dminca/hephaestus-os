local config = vim.fs.dirname(vim.fn.resolve(debug.getinfo(1, "S").source:sub(2)))
vim.opt.runtimepath:prepend(config)
vim.g.hephaestus_assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.env.PATH = "/usr/libexec/hephaestus/bin:" .. (vim.env.PATH or "")
vim.treesitter.language.register("json", "jsonc")
require("hephaestus.options")
require("hephaestus.mappings")

local plugins = vim.g.hephaestus_assets .. "/nvim/plugins"
if vim.fn.isdirectory(plugins .. "/lazy.nvim") == 0 then
  error("Hephaestus Neovim plugins are missing; run the image build's install-profile.sh")
end
vim.opt.runtimepath:prepend(plugins .. "/lazy.nvim")
local specs = require("hephaestus.plugins")
for _, spec in ipairs(specs) do
  spec.dir = plugins .. "/" .. spec[1]:match("[^/]+$")
  spec[1] = nil
end
require("lazy").setup(specs, {
  root = vim.fn.stdpath("data") .. "/hephaestus-lazy",
  defaults = { lazy = false },
  install = { missing = false },
  checker = { enabled = false },
  change_detection = { enabled = false },
  pkg = { enabled = false },
  rocks = { enabled = false },
  performance = { rtp = { reset = false } },
})
