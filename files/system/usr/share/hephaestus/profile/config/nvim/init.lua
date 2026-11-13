local config = vim.fs.dirname(vim.fn.resolve(debug.getinfo(1, "S").source:sub(2)))
vim.opt.runtimepath:prepend(config)
vim.g.hephaestus_assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.env.PATH = "/usr/libexec/hephaestus/bin:" .. (vim.env.PATH or "")
vim.treesitter.language.register("json", "jsonc")
require("hephaestus.options")
require("hephaestus.mappings")

assert(vim.fn.has("nvim-0.12") == 1, "Hephaestus requires Neovim 0.12+ with vim.pack")
require("hephaestus.packages").setup()
require("hephaestus.plugins")
