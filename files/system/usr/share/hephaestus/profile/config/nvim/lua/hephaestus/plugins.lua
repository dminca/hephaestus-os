require("nvim-web-devicons").setup()
require("catppuccin").setup({
  flavour = "mocha", transparent_background = true,
  integrations = { native_lsp = { enabled = true, underlines = {
    errors = { "undercurl" }, hints = { "undercurl" },
    warnings = { "undercurl" }, information = { "undercurl" },
  } } },
})
vim.cmd.colorscheme("catppuccin")
require("hephaestus.snacks")
require("todo-comments").setup()
require("neoscroll").setup()
require("smart-splits").setup()
require("flash").setup({ modes = { char = { enabled = false } } })
require("which-key").setup({
  preset = "helix", icons = { colors = false },
  spec = {
    { "<leader>b", group = "buffers" }, { "<leader><Tab>", group = "tabs" },
    { "<leader>q", group = "quit/session" }, { "<leader>i", group = "insert" },
    { "<leader>u", group = "UI" }, { "<leader>o", group = "open" },
    { "<leader>d", group = "diff" }, { "<leader>c", group = "quickfix" },
    { "<leader>s", group = "search" }, { "<leader>f", group = "file/find" },
    { "<leader>g", group = "git" }, { "<leader>gh", group = "hunks" },
    { "<leader>l", group = "LSP" }, { "<leader>p", group = "preview" },
    { "<leader>R", group = "HTTP requests" },
  },
})
require("Comment").setup({ toggler = { line = "<leader>/" }, opleader = { line = "<leader>/" } })
require("nvim-surround").setup()
require("nvim-autopairs").setup()
require("trim").setup()
require("stay-centered").setup()
require("gx").setup()
vim.keymap.set({ "n", "x" }, "gx", "<cmd>Browse<cr>")
require("windows").setup({ ignore = { filetype = { "snacks_picker_list", "snacks_layout_box" } } })
require("colorizer").setup({
  filetypes = { "*" }, user_default_options = {
    names = true, RRGGBBAA = true, AARRGGBB = true, rgb_fn = true, hsl_fn = true,
    css = true, css_fn = true, tailwind = true, mode = "virtualtext", always_update = true,
  },
})
require("dressing").setup({ input = { mappings = { n = { q = "Close", k = "HistoryPrev", j = "HistoryNext" } } } })
require("nvim-lastplace").setup()
require("fidget").setup({
  progress = { display = { progress_icon = { pattern = "moon" } } },
  notification = { window = { relative = "editor", winblend = 0, border = "none" } },
})
require("harpoon"):setup()
require("bufferline").setup({ options = {
  diagnostics = "nvim_lsp", always_show_bufferline = true,
  hover = { enabled = true, delay = 200, reveal = { "close" } },
} })
require("noice").setup({
  presets = { bottom_search = true },
  routes = {
    { filter = { event = "msg_show", kind = { "shell_out" } }, view = "notify", opts = { level = "info", title = "stdout" } },
    { filter = { event = "msg_show", kind = { "shell_err" } }, view = "notify", opts = { level = "error", title = "stderr" } },
  },
  views = {
    cmdline_popup = { position = { row = -2, col = "50%" }, border = { style = "none" } },
    cmdline_popupmenu = { position = { row = -5, col = "50%" } },
  },
  lsp = {
    override = {
      ["vim.lsp.util.convert_input_to_markdown_lines"] = true,
      ["vim.lsp.util.stylize_markdown"] = true, ["cmp.entry.get_documentation"] = true,
    },
    hover = { enabled = false }, message = { enabled = false },
    signature = { enabled = false }, progress = { enabled = false },
  },
})
require("diffview").setup({
  file_panel = { win_config = { position = "right", width = 30 } },
  keymaps = {
    view = { { "n", "<leader>e", function() require("diffview.actions").toggle_files() end } },
    file_panel = { { "n", "<leader>e", function() require("diffview.actions").toggle_files() end } },
  },
})
require("git-conflict").setup({ default_mappings = true })
require("gitsigns").setup({ current_line_blame = true, preview_config = { border = "rounded" } })
require("hephaestus.git")
require("hephaestus.statusline")
require("auto-session").setup({
  auto_restore = true, git_use_branch_name = true,
  session_lens = { picker_opts = { preview = true }, mappings = { delete_session = { "n", "d" } } },
})
require("leetcode").setup({ lang = "python3" })
require("luasnip").setup()
require("blink.cmp").setup({
  fuzzy = { implementation = "lua" },
  snippets = { preset = "luasnip" },
  completion = { menu = { border = "rounded" }, list = { selection = { preselect = false } } },
  keymap = {
    ["<C-j>"] = { "select_next", "fallback" }, ["<C-k>"] = { "select_prev", "fallback" },
    ["<C-l>"] = { "snippet_forward", "fallback" }, ["<C-h>"] = { "snippet_backward", "fallback" },
    ["<C-u>"] = { "scroll_documentation_up", "fallback" }, ["<C-d>"] = { "scroll_documentation_down", "fallback" },
    ["<C-space>"] = { "select_and_accept", "fallback" },
  },
})
require("mason").setup()
require("hephaestus.languages")
require("hephaestus.treesitter").setup()
require("treesitter-context").setup({ max_lines = 4, min_window_height = 40 })
require("mini.ai").setup()
require("conform").setup({
  default_format_opts = { lsp_format = "prefer" },
  formatters_by_ft = {
    ["_"] = { "squeeze_blanks", "trim_whitespace", "trim_newlines" },
    xml = { "xmllint" }, json = { "fixjson" }, go = { "goimports" },
    lua = { "stylua" }, nix = { "nixfmt" }, terraform = { "tofu_fmt" },
    sh = { "shellcheck", "shellharden", "shfmt" }, bash = { "shellcheck", "shellharden", "shfmt" },
    zsh = { "shellcheck", "shellharden", "shfmt" }, toml = { "taplo" }, typst = { "typstyle" },
  },
  formatters = { squeeze_blanks = { command = "cat" }, tofu_fmt = { command = "tofu", args = { "fmt", "-" }, stdin = true } },
})
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = "*.go", callback = function() require("conform").format({ async = false }) end,
})
require("trouble").setup()
require("tiny-inline-diagnostic").setup()
require("lspsaga").setup({
  lightbulb = { enable = false, virtual_text = false }, outline = { keys = { jump = "<cr>" } },
  ui = { border = "rounded" }, scroll_preview = { scroll_down = "<C-d>", scroll_up = "<C-u>" },
})
require("ufo").setup({
  provider_selector = function() return { "lsp", "indent" } end,
  preview = { mappings = { close = "q", switch = "K" } },
})
require("crates").setup()
require("img-clip").setup()
require("render-markdown").setup({
  ignore = function(buf)
    return vim.api.nvim_buf_get_name(buf or 0):match("/leetcode/") ~= nil
      or vim.bo[buf or 0].filetype == "leetcode.nvim"
  end,
  link = { wiki = { body = function(ctx)
    if (ctx.text or ""):match("^[%d%s,%-%[%]%.\"']+$") then return false end
  end } },
})
require("glow").setup()
require("typst-preview").setup()
local http_ready = false
local function setup_http()
  if http_ready or vim.fn.executable("tree-sitter") ~= 1 then return end
  require("kulala").setup({ global_keymaps = true, global_keymaps_prefix = "<leader>R", kulala_keymaps_prefix = "" })
  http_ready = true
end
vim.api.nvim_create_autocmd("FileType", {
  pattern = { "http", "rest" },
  callback = function()
    setup_http()
    if not http_ready then
      vim.notify("HTTP tooling is waiting for tree-sitter. See :Mason; retry :MasonToolsInstall.", vim.log.levels.WARN)
    end
  end,
})
vim.api.nvim_create_autocmd("User", { pattern = "MasonToolsUpdateCompleted", callback = setup_http })
setup_http()
require("nvim-ts-autotag").setup()
require("ts-comments").setup()
require("codesnap").setup({ snapshot_config = {
  background = { stops = { { color = "#00000000" }, { color = "#00000000" } } },
  theme = "vercel@https://raw.githubusercontent.com/Railly/one-hunter-vscode/refs/heads/main/themes/OneHunter-Vercel-color-theme.json",
  watermark = { content = "" },
} })
require("cord").setup({ advanced = { server = { auto_update = false } } })
require("remote-nvim").setup()
