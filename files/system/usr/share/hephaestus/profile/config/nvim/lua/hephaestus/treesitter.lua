local M = {}
M.languages = {
  "bash", "c", "cpp", "css", "cue", "diff", "dockerfile", "git_config", "git_rebase",
  "gitattributes", "gitcommit", "gitignore", "go", "gomod", "gosum", "gowork", "haskell",
  "hcl", "html", "http", "javascript", "jsdoc", "json", "lua", "luadoc",
  "markdown", "markdown_inline", "nix", "python", "query", "regex", "rust", "scss",
  "svelte", "toml", "tsx", "typescript", "typst", "vim", "vimdoc", "xml", "yaml",
}

function M.setup()
  local treesitter = require("nvim-treesitter")
  local directory = vim.fn.stdpath("data") .. "/site"
  treesitter.setup({ install_dir = directory })
  local function highlight(buf)
    local language = vim.treesitter.language.get_lang(vim.bo[buf].filetype)
    if language and vim.fn.filereadable(directory .. "/parser/" .. language .. ".so") == 1 then
      vim.treesitter.start(buf, language)
      vim.bo[buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
    end
  end
  local installing = false
  local function install()
    if installing or vim.fn.executable("tree-sitter") ~= 1 then return end
    installing = true
    treesitter.install(M.languages, { summary = true, max_jobs = 4 }):await(vim.schedule_wrap(function(err, success)
      installing = false
      if err or not success then
        vim.notify("Treesitter installation failed: " .. tostring(err or "see :messages")
          .. "\nRestart to retry, or use :TSInstall <language> for a missing parser.", vim.log.levels.ERROR)
        return
      end
      for _, buf in ipairs(vim.api.nvim_list_bufs()) do
        if vim.api.nvim_buf_is_loaded(buf) then highlight(buf) end
      end
    end))
  end
  vim.api.nvim_create_autocmd("FileType", {
    callback = function(event) highlight(event.buf) end,
  })
  vim.api.nvim_create_autocmd("User", {
    pattern = "MasonToolsUpdateCompleted", callback = install,
  })
  vim.schedule(install)
end

return M
