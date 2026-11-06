local config = vim.fs.dirname(vim.fn.resolve(debug.getinfo(1, "S").source:sub(2)))
vim.opt.runtimepath:prepend(config)
vim.g.hephaestus_assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.env.PATH = "/usr/libexec/hephaestus/bin:" .. (vim.env.PATH or "")
vim.treesitter.language.register("json", "jsonc")
require("hephaestus.options")
require("hephaestus.mappings")

local lazy = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if vim.fn.isdirectory(lazy) == 0 then
  local result = vim.system({
    "git", "clone", "--filter=blob:none", "--branch=stable",
    "https://github.com/folke/lazy.nvim.git", lazy,
  }, { text = true }):wait()
  if result.code ~= 0 then
    vim.notify("Could not install lazy.nvim:\n" .. (result.stderr or "")
      .. "\nRestart Neovim to retry.", vim.log.levels.ERROR)
    return
  end
end
vim.opt.runtimepath:prepend(lazy)
local lock = vim.fn.stdpath("state") .. "/lazy-lock.json"
if vim.fn.filereadable(lock) == 0 then
  vim.fn.mkdir(vim.fn.stdpath("state"), "p")
  local seed = vim.fn.readfile(config .. "/lazy-lock.json")
  assert(vim.fn.writefile(seed, lock) == 0, "Could not seed user plugin lock: " .. lock)
end
require("lazy").setup(require("hephaestus.plugins"), {
  lockfile = lock,
  defaults = { lazy = false },
  install = { missing = true },
  checker = { enabled = false },
  change_detection = { enabled = false },
  pkg = { enabled = false },
  rocks = { enabled = false },
  performance = { rtp = { reset = false } },
})
