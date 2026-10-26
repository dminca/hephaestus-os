return {
  { "nvim-lua/plenary.nvim" },
  { "MunifTanjim/nui.nvim" },
  { "nvim-tree/nvim-web-devicons", opts = {} },
  { "catppuccin/nvim", name = "catppuccin", priority = 1000, config = function()
    require("catppuccin").setup({
      flavour = "mocha", transparent_background = true,
      integrations = { native_lsp = { enabled = true, underlines = {
        errors = { "undercurl" }, hints = { "undercurl" },
        warnings = { "undercurl" }, information = { "undercurl" },
      } } },
    })
    vim.cmd.colorscheme("catppuccin")
  end },
  { "folke/snacks.nvim", priority = 900, config = function() require("hephaestus.snacks") end },
  { "folke/todo-comments.nvim", opts = {} },
  { "karb94/neoscroll.nvim", opts = {} },
  { "mrjones2014/smart-splits.nvim", opts = {} },
  { "christoomey/vim-tmux-navigator" },
  { "folke/flash.nvim", opts = { modes = { char = { enabled = false } } } },
  { "folke/which-key.nvim", opts = {
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
  } },
  { "numToStr/Comment.nvim", opts = { toggler = { line = "<leader>/" }, opleader = { line = "<leader>/" } } },
  { "kylechui/nvim-surround", opts = {} },
  { "windwp/nvim-autopairs", opts = {} },
  { "cappyzawa/trim.nvim", opts = {} },
  { "mg979/vim-visual-multi" },
  { "arnamak/stay-centered.nvim", opts = {} },
  { "direnv/direnv.vim" },
  { "chrishrb/gx.nvim", opts = {}, config = function(_, opts)
    require("gx").setup(opts)
    vim.keymap.set({ "n", "x" }, "gx", "<cmd>Browse<cr>")
  end },
  { "anuvyklack/middleclass" },
  { "anuvyklack/animation.nvim" },
  { "anuvyklack/windows.nvim", opts = { ignore = { filetype = { "snacks_picker_list", "snacks_layout_box" } } } },
  { "NvChad/nvim-colorizer.lua", opts = {
    filetypes = { "*" }, user_default_options = {
      names = true, RRGGBBAA = true, AARRGGBB = true, rgb_fn = true, hsl_fn = true,
      css = true, css_fn = true, tailwind = true, mode = "virtualtext", always_update = true,
    },
  } },
  { "stevearc/dressing.nvim", opts = { input = { mappings = { n = { q = "Close", k = "HistoryPrev", j = "HistoryNext" } } } } },
  { "ethanholz/nvim-lastplace", opts = {} },
  { "j-hui/fidget.nvim", opts = {
    progress = { display = { progress_icon = { pattern = "moon" } } },
    notification = { window = { relative = "editor", winblend = 0, border = "none" } },
  } },
  { "ThePrimeagen/harpoon", opts = {} },
  { "akinsho/bufferline.nvim", opts = { options = {
    diagnostics = "nvim_lsp", always_show_bufferline = true,
    hover = { enabled = true, delay = 200, reveal = { "close" } },
  } } },
  { "folke/noice.nvim", opts = {
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
  } },
  { "sindrets/diffview.nvim", opts = {
    file_panel = { win_config = { position = "right", width = 30 } },
    keymaps = {
      view = { { "n", "<leader>e", function() require("diffview.actions").toggle_files() end } },
      file_panel = { { "n", "<leader>e", function() require("diffview.actions").toggle_files() end } },
    },
  } },
  { "akinsho/git-conflict.nvim", opts = { default_mappings = true } },
  { "lewis6991/gitsigns.nvim", opts = {
    current_line_blame = true, preview_config = { border = "rounded" },
  }, config = function(_, opts) require("gitsigns").setup(opts); require("hephaestus.git") end },
  { "nvim-lualine/lualine.nvim", config = function() require("hephaestus.statusline") end },
  { "rmagatti/auto-session", opts = {
    auto_restore = true, git_use_branch_name = true,
    session_lens = { picker_opts = { preview = true }, mappings = { delete_session = { "n", "d" } } },
  } },
  { "glacambre/firenvim", init = function()
    vim.g.firenvim_config = {
      globalSettings = { cmdline = "neovim", content = "text", priority = 0, selector = "textarea", takeover = "never" },
      localSettings = { [".*"] = { takeover = "never", cmdline = "neovim" } },
    }
  end },
  { "niksingh710/leetcode.nvim", cmd = "Leet", opts = { lang = "python3" } },
  { "L3MON4D3/LuaSnip", opts = {} },
  { "saghen/blink.cmp", opts = {
    fuzzy = { implementation = "lua" },
    snippets = { preset = "luasnip" },
    completion = { menu = { border = "rounded" }, list = { selection = { preselect = false } } },
    keymap = {
      ["<C-j>"] = { "select_next", "fallback" }, ["<C-k>"] = { "select_prev", "fallback" },
      ["<C-l>"] = { "snippet_forward", "fallback" }, ["<C-h>"] = { "snippet_backward", "fallback" },
      ["<C-u>"] = { "scroll_documentation_up", "fallback" }, ["<C-d>"] = { "scroll_documentation_down", "fallback" },
      ["<C-space>"] = { "select_and_accept", "fallback" },
    },
  } },
  { "nvim-treesitter/nvim-treesitter", config = function()
    local directory = vim.g.hephaestus_assets .. "/nvim/treesitter"
    require("nvim-treesitter").setup({ install_dir = directory })
    vim.api.nvim_create_autocmd("FileType", {
      callback = function(event)
        local language = vim.treesitter.language.get_lang(vim.bo[event.buf].filetype)
        if language and vim.fn.filereadable(directory .. "/parser/" .. language .. ".so") == 1 then
          vim.treesitter.start(event.buf, language)
          vim.bo[event.buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
        end
      end,
    })
  end },
  { "nvim-treesitter/nvim-treesitter-context", opts = { max_lines = 4, min_window_height = 40 } },
  { "echasnovski/mini.ai", opts = {} },
  { "tpope/vim-sleuth" },
  { "neovim/nvim-lspconfig", config = function() require("hephaestus.languages") end },
  { "stevearc/conform.nvim", opts = {
    default_format_opts = { lsp_format = "prefer" },
    formatters_by_ft = {
      ["_"] = { "squeeze_blanks", "trim_whitespace", "trim_newlines" },
      xml = { "xmllint" }, yaml = { "yamlfix" }, json = { "fixjson" }, go = { "goimports" },
      lua = { "stylua" }, nix = { "nixfmt" }, terraform = { "tofu_fmt" },
      sh = { "shellcheck", "shellharden", "shfmt" }, bash = { "shellcheck", "shellharden", "shfmt" },
      zsh = { "shellcheck", "shellharden", "shfmt" }, toml = { "taplo" }, typst = { "typstyle" },
    },
    formatters = { squeeze_blanks = { command = "cat" }, tofu_fmt = { command = "tofu", args = { "fmt", "-" }, stdin = true } },
  }, config = function(_, opts)
    require("conform").setup(opts)
    vim.api.nvim_create_autocmd("BufWritePre", {
      pattern = "*.go", callback = function() require("conform").format({ async = false }) end,
    })
  end },
  { "folke/trouble.nvim", opts = {} },
  { "rachartier/tiny-inline-diagnostic.nvim", opts = {} },
  { "nvimdev/lspsaga.nvim", opts = {
    lightbulb = { enable = false, virtual_text = false }, outline = { keys = { jump = "<cr>" } },
    ui = { border = "rounded" }, scroll_preview = { scroll_down = "<C-d>", scroll_up = "<C-u>" },
  } },
  { "kevinhwang91/promise-async" },
  { "kevinhwang91/nvim-ufo", opts = {
    provider_selector = function() return { "lsp", "indent" } end,
    preview = { mappings = { close = "q", switch = "K" } },
  } },
  { "mrcjkb/rustaceanvim", init = function()
    vim.g.rustaceanvim = { server = { auto_attach = function() return vim.fn.executable("rust-analyzer") == 1 end } }
  end },
  { "saecki/crates.nvim", opts = {} },
  { "HakonHarnes/img-clip.nvim", opts = {} },
  { "iamcco/markdown-preview.nvim", ft = "markdown", init = function()
    vim.g.mkdp_echo_preview_url = 1
    -- Keep the preview bound to loopback rather than exposing it to the LAN.
    vim.g.mkdp_open_to_the_world = 0
    vim.g.mkdp_browserfunc = "MkdpNoOp"
    vim.cmd("function! MkdpNoOp(url)\nendfunction")
  end },
  { "MeanderingProgrammer/render-markdown.nvim", opts = {
    ignore = function(buf)
      return vim.api.nvim_buf_get_name(buf or 0):match("/leetcode/") ~= nil
        or vim.bo[buf or 0].filetype == "leetcode.nvim"
    end,
    link = { wiki = { body = function(ctx)
      if (ctx.text or ""):match("^[%d%s,%-%[%]%.\"']+$") then return false end
    end } },
  } },
  { "ellisonleao/glow.nvim", cmd = "Glow", opts = {} },
  { "chomosuke/typst-preview.nvim", opts = {
    dependencies_bin = { tinymist = "/usr/bin/tinymist", websocat = "/usr/bin/websocat" },
  } },
  { "mistweaverco/kulala.nvim", ft = { "http", "rest" }, opts = {
    global_keymaps = true, global_keymaps_prefix = "<leader>R", kulala_keymaps_prefix = "",
  } },
  { "windwp/nvim-ts-autotag", opts = {} },
  { "folke/ts-comments.nvim", opts = {} },
  { "mistricky/codesnap.nvim", cmd = { "CodeSnap", "CodeSnapSave" }, opts = {
    snapshot_config = {
      background = { stops = { { color = "#00000000" }, { color = "#00000000" } } },
      theme = "vercel@https://raw.githubusercontent.com/Railly/one-hunter-vscode/refs/heads/main/themes/OneHunter-Vercel-color-theme.json",
      watermark = { content = "" },
    },
  } },
  { "vyfor/cord.nvim", event = "VeryLazy", opts = { advanced = { server = {
    auto_update = false, executable_path = "/usr/libexec/hephaestus/cord",
  } } } },
  { "amitds1997/remote-nvim.nvim", cmd = "RemoteNvim", opts = {} },
}
