local assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.opt.runtimepath:prepend(assets .. "/nvim/plugins/nvim-treesitter")
local treesitter = require("nvim-treesitter")
treesitter.setup({ install_dir = assets .. "/nvim/treesitter" })
local languages = {
  "bash", "c", "cpp", "css", "cue", "diff", "dockerfile", "git_config", "git_rebase",
  "gitattributes", "gitcommit", "gitignore", "go", "gomod", "gosum", "gowork", "haskell",
  "hcl", "html", "http", "javascript", "jsdoc", "json", "jsonc", "lua", "luadoc",
  "markdown", "markdown_inline", "nix", "python", "query", "regex", "rust", "scss",
  "svelte", "toml", "tsx", "typescript", "typst", "vim", "vimdoc", "xml", "yaml",
}
treesitter.install(languages):wait(600000)
for _, language in ipairs(languages) do
  assert(vim.fn.filereadable(assets .. "/nvim/treesitter/parser/" .. language .. ".so") == 1,
    "Parser build failed: " .. language)
end
io.stdout:write("All " .. #languages .. " native Treesitter parsers built\n")
