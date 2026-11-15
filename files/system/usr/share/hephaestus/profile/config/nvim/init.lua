local config = vim.fs.dirname(vim.fn.resolve(debug.getinfo(1, "S").source:sub(2)))
vim.opt.runtimepath:prepend(config)
vim.g.hephaestus_assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.env.PATH = "/usr/libexec/hephaestus/bin:" .. (vim.env.PATH or "")
vim.treesitter.language.register("json", "jsonc")
require("hephaestus.options")
require("hephaestus.mappings")

assert(vim.fn.has("nvim-0.12") == 1, "Hephaestus requires Neovim 0.12+ with vim.pack")
local user_config = vim.fn.stdpath("config")
assert(vim.fn.resolve(user_config) ~= vim.fn.resolve(vim.g.hephaestus_assets .. "/profile/config/nvim"),
  "Neovim's configuration directory points to the read-only image profile: " .. user_config
    .. "\nRun hephaestus-profile --replace to back up the old link and create a writable directory."
    .. "\nKeep bash environment lockdown enabled; only init.lua and lua/ should link to the image.")
require("hephaestus.packages").setup()
require("hephaestus.plugins")
