local commands = {
  biome = { "biome", "lsp-proxy" },
  cssls = { "vscode-css-language-server", "--stdio" },
  eslint = { "vscode-eslint-language-server", "--stdio" },
  html = { "vscode-html-language-server", "--stdio" },
  jsonls = { "vscode-json-language-server", "--stdio" },
  svelte = { "svelteserver", "--stdio" },
  tailwindcss = { "tailwindcss-language-server", "--stdio" },
  ts_ls = { "typescript-language-server", "--stdio" },
  yamlls = { "yaml-language-server", "--stdio" },
}
local servers = {
  gopls = { settings = { gopls = {
    gofumpt = true, usePlaceholders = true, completeUnimported = true,
    semanticTokens = true, staticcheck = true,
  } } },
  cue = {}, hls = {}, nil_ls = {}, tofu_ls = {}, ruff = {},
  lua_ls = { settings = { Lua = { diagnostics = {
    disable = { "miss-name" }, globals = { "vim", "cmp", "Snacks" },
  } } } },
  markdown_oxide = {}, marksman = {},
  pyright = { settings = { pyright = { disableOrganizeImports = true }, python = { analysis = { ignore = { "*" } } } } },
  bashls = {}, taplo = {}, tinymist = {}, ts_ls = {}, tailwindcss = {}, svelte = {},
  jsonls = {}, html = {}, eslint = {}, emmet_ls = {}, cssls = {}, biome = {},
  typos_lsp = { init_options = { diagnosticSeverity = "Hint" } },
  yamlls = { settings = { yaml = {
    format = { enable = true }, validate = true, completion = true, hover = true,
    schemaStore = { enable = true, url = "https://www.schemastore.org/api/json/catalog.json" },
    kubernetesCRDStore = { enable = true }, schemas = { kubernetes = "/*.yaml" },
  } } },
}
for name, config in pairs(servers) do
  config.cmd = commands[name]
  config.capabilities = require("blink.cmp").get_lsp_capabilities()
  vim.lsp.config(name, config)
end
local tools = require("hephaestus.tools")
local function enable_ready()
  for name in pairs(servers) do
    local config = vim.lsp.config[name]
    assert(type(config.cmd) == "table" and type(config.cmd[1]) == "string",
      "Expected an executable command in the native LSP definition for " .. name)
    if tools.ready(assert(tools.servers[name], "Missing Mason package for " .. name))
      and vim.fn.executable(config.cmd[1]) == 1 then
      vim.lsp.enable(name)
    end
  end
  if tools.ready(tools.servers.rust_analyzer) and vim.fn.executable("rust-analyzer") == 1 then
    for _, buf in ipairs(vim.api.nvim_list_bufs()) do
      if vim.api.nvim_buf_is_loaded(buf) and vim.bo[buf].filetype == "rust"
        and #vim.lsp.get_clients({ bufnr = buf, name = "rust-analyzer" }) == 0 then
        require("rustaceanvim.lsp").start(buf)
      end
    end
  end
end
vim.api.nvim_create_autocmd("User", {
  pattern = "MasonToolsUpdateCompleted", callback = function() vim.schedule(enable_ready) end,
})
require("mason-registry"):on("package:install:success", function() vim.schedule(enable_ready) end)
tools.setup()
enable_ready()
vim.api.nvim_create_autocmd("LspAttach", {
  callback = function(event)
    local client = vim.lsp.get_client_by_id(event.data.client_id)
    if client and client:supports_method("textDocument/inlayHint") then
      vim.lsp.inlay_hint.enable(true, { bufnr = event.buf })
    end
    vim.keymap.set("n", "<leader>lj", function() vim.diagnostic.jump({ count = 1 }) end, { buffer = event.buf })
    vim.keymap.set("n", "<leader>lk", function() vim.diagnostic.jump({ count = -1 }) end, { buffer = event.buf })
  end,
})
vim.api.nvim_create_autocmd("FileType", {
  pattern = "markdown",
  callback = function(event)
    vim.keymap.set("n", "<leader>pg", "<cmd>Glow<cr>", { buffer = event.buf, desc = "Glow preview" })
    vim.keymap.set("n", "<leader>pb", function()
      vim.cmd("MarkdownPreview")
      vim.defer_fn(function()
        local messages = vim.api.nvim_exec2("messages", { output = true }).output
        local url
        for candidate in messages:gmatch("https?://[%d%.]+:%d+/%S*") do url = candidate end
        if url then
          url = url:gsub("https?://[%d%.]+", "http://localhost")
          vim.fn.setreg("+", url)
          vim.notify("URL copied to clipboard: " .. url)
        else
          vim.notify("Markdown preview did not report a URL", vim.log.levels.ERROR)
        end
      end, 500)
    end, { buffer = event.buf, desc = "Browser preview and copy URL" })
    vim.keymap.set("n", "<leader>pp", function()
      if vim.fn.executable("pandoc") == 0 then
        vim.notify("PDF export requires pandoc and a PDF engine", vim.log.levels.ERROR)
        return
      end
      local file = vim.api.nvim_buf_get_name(0)
      vim.system({ "pandoc", file, "-o", vim.fn.fnamemodify(file, ":r") .. ".pdf" }, {}, vim.schedule_wrap(function(result)
        vim.notify(result.code == 0 and "PDF exported" or result.stderr,
          result.code == 0 and vim.log.levels.INFO or vim.log.levels.ERROR)
      end))
    end, { buffer = event.buf, desc = "Print PDF" })
  end,
})
vim.keymap.set("n", "<leader>o<cr>", function()
  local row, column = unpack(vim.api.nvim_win_get_cursor(0))
  local line = vim.api.nvim_get_current_line()
  local start = 1
  while true do
    local first, last = line:find("[A-Za-z0-9_/%-]+", start)
    if not first then return end
    if column + 1 >= first and column + 1 <= last then
      vim.api.nvim_win_set_cursor(0, { row, first - 1 })
      vim.cmd("normal! v")
      vim.api.nvim_win_set_cursor(0, { row, last - 1 })
      return
    end
    start = last + 1
  end
end, { desc = "Select note under cursor" })
