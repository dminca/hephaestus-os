local M = {}

-- Native LSP names differ from Mason registry package names.
M.servers = {
  bashls = { "bash-language-server", requires = { "node", "npm" } },
  biome = { "biome", requires = { "node", "npm" } },
  cssls = { "css-lsp", requires = { "node", "npm" } },
  cue = { "cue", requires = { "go" } },
  emmet_ls = { "emmet-ls", requires = { "node", "npm" } },
  eslint = { "eslint-lsp", requires = { "node", "npm" } },
  gopls = { "gopls", requires = { "go" } },
  hls = { "haskell-language-server", requires = { "ghcup", "ghc" } },
  html = { "html-lsp", requires = { "node", "npm" } },
  jsonls = { "json-lsp", requires = { "node", "npm" } },
  lua_ls = { "lua-language-server" },
  markdown_oxide = { "markdown-oxide" },
  marksman = { "marksman" },
  nil_ls = { "nil", requires = { "cargo", "rustc", "cc" } },
  pyright = { "pyright", requires = { "node", "npm" } },
  ruff = { "ruff" },
  rust_analyzer = { "rust-analyzer", requires = { "cargo", "rustc" } },
  svelte = { "svelte-language-server", requires = { "node", "npm" } },
  tailwindcss = { "tailwindcss-language-server", requires = { "node", "npm" } },
  taplo = { "taplo" },
  tinymist = { "tinymist" },
  tofu_ls = { "tofu-ls" },
  ts_ls = { "typescript-language-server", requires = { "node", "npm" } },
  typos_lsp = { "typos-lsp" },
  yamlls = { "yaml-language-server", requires = { "node", "npm" } },
}

function M.ready(tool)
  for _, executable in ipairs(tool.requires or {}) do
    if vim.fn.executable(executable) ~= 1 then return false, executable end
  end
  return true
end

function M.setup()
  local tools = vim.tbl_values(M.servers)
  vim.list_extend(tools, {
    { "tree-sitter-cli" }, { "goimports", requires = { "go" } }, { "stylua" },
    { "nixfmt" }, { "shellcheck" }, { "shellharden" }, { "shfmt" },
    { "typstyle" }, { "fixjson", requires = { "node", "npm" } },
  })
  table.sort(tools, function(a, b) return a[1] < b[1] end)
  local entries = {}
  for _, tool in ipairs(tools) do
    entries[#entries + 1] = {
      tool[1],
      condition = function()
        local ready, missing = M.ready(tool)
        if not ready then
          vim.notify(tool[1] .. " skipped: missing " .. missing
            .. ". Install the prerequisite in your user environment, then :MasonToolsInstall.", vim.log.levels.WARN)
        end
        return ready
      end,
    }
  end
  local registry = require("mason-registry")
  registry:on("package:install:failed", function(package, err)
    vim.schedule(function()
      vim.notify(package.name .. " installation failed: " .. tostring(err) .. ". See :MasonLog.", vim.log.levels.ERROR)
    end)
  end)
  registry:on("update:failed", function(err)
    vim.schedule(function()
      vim.notify("Mason registry refresh failed: " .. tostring(err) .. ". See :MasonLog.", vim.log.levels.ERROR)
    end)
  end)
  local installer = require("mason-tool-installer")
  installer.setup({
    ensure_installed = entries, auto_update = false, run_on_start = true,
    integrations = { ["mason-lspconfig"] = false, ["mason-null-ls"] = false, ["mason-nvim-dap"] = false },
  })
  if vim.v.vim_did_enter == 1 then installer.run_on_start() end
end

return M
