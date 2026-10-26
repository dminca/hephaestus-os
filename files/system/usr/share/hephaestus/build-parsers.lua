local assets = vim.env.HEPHAESTUS_ASSETS or "/usr/share/hephaestus"
vim.opt.runtimepath:prepend(assets .. "/nvim/plugins/nvim-treesitter")
local treesitter = require("nvim-treesitter")
treesitter.setup({ install_dir = assets .. "/nvim/treesitter" })
local languages = {
  "bash", "c", "cpp", "css", "cue", "diff", "dockerfile", "git_config", "git_rebase",
  "gitattributes", "gitcommit", "gitignore", "go", "gomod", "gosum", "gowork", "haskell",
  "hcl", "html", "http", "javascript", "jsdoc", "json", "lua", "luadoc",
  "markdown", "markdown_inline", "nix", "python", "query", "regex", "rust", "scss",
  "svelte", "toml", "tsx", "typescript", "typst", "vim", "vimdoc", "xml", "yaml",
}
local parsers = require("nvim-treesitter.parsers")
local unsupported = {}
for _, language in ipairs(languages) do
  if not parsers[language] or parsers[language].tier == 4 then
    unsupported[#unsupported + 1] = language
  end
end
assert(#unsupported == 0, "Unavailable parsers in pinned plugin: " .. table.concat(unsupported, ", "))

local success = treesitter.install(languages, { summary = true, max_jobs = 4 }):wait(600000)
local failures = {}
for _, language in ipairs(languages) do
  local parser = assets .. "/nvim/treesitter/parser/" .. language .. ".so"
  if vim.fn.filereadable(parser) ~= 1 then
    failures[#failures + 1] = language .. ": parser file missing"
  else
    local loaded, result = pcall(vim.treesitter.language.add, language, { path = parser })
    if not loaded or not result then
      failures[#failures + 1] = language .. ": " .. tostring(result)
    end
  end
end
assert(#failures == 0, "Parser validation failed:\n" .. table.concat(failures, "\n"))
assert(success, "Treesitter installation reported a failure; see the installer output above")
io.stdout:write("All " .. #languages .. " native Treesitter parsers built\n")
