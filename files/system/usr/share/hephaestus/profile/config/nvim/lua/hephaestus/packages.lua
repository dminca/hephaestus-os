local M = {}

M.specs = {
  { src = "https://github.com/nvim-lua/plenary.nvim", version = "74b06c6c75e4eeb3108ec01852001636d85a932b" },
  { src = "https://github.com/MunifTanjim/nui.nvim", version = "10fc361835c856ba4233ef5ea135b919bf3dce97" },
  { src = "https://github.com/nvim-tree/nvim-web-devicons", version = "ec309a0eaff6ea967e58c275996155ab5c16938d" },
  { src = "https://github.com/catppuccin/nvim", name = "catppuccin", version = "edefef779ab08ce1a4a404713e3012b0d202bd35" },
  { src = "https://github.com/folke/snacks.nvim", version = "882c996cf28183f4d63640de0b4c02ec886d01f2" },
  { src = "https://github.com/folke/todo-comments.nvim", version = "31e3c38ce9b29781e4422fc0322eb0a21f4e8668" },
  { src = "https://github.com/karb94/neoscroll.nvim", version = "c8d29979cb0cb3a2437a8e0ae683fd82f340d3b8" },
  { src = "https://github.com/mrjones2014/smart-splits.nvim", version = "dd40c3a03ef71bcdf77fb4b14668cc1320293f48" },
  { src = "https://github.com/christoomey/vim-tmux-navigator", version = "e41c431a0c7b7388ae7ba341f01a0d217eb3a432" },
  { src = "https://github.com/folke/flash.nvim", version = "5f0f270fdc7c5b0c21d903ee85b9cb06f2ac636a" },
  { src = "https://github.com/folke/which-key.nvim", version = "3aab2147e74890957785941f0c1ad87d0a44c15a" },
  { src = "https://github.com/numToStr/Comment.nvim", version = "e30b7f2008e52442154b66f7c519bfd2f1e32acb" },
  { src = "https://github.com/kylechui/nvim-surround", version = "8b47db616ef658b8fc27e61db2896aa2f40134de" },
  { src = "https://github.com/windwp/nvim-autopairs", version = "3172f40c16563db871a58f20d205c710ad782311" },
  { src = "https://github.com/cappyzawa/trim.nvim", version = "765360a6f6ac732f4c78c5c694f4b892a55b53ec" },
  { src = "https://github.com/mg979/vim-visual-multi", version = "a6975e7c1ee157615bbc80fc25e4392f71c344d4" },
  { src = "https://github.com/arnamak/stay-centered.nvim", version = "e1a63ccaf2584e97c0ef8e64f9654c9a80d983f6" },
  { src = "https://github.com/direnv/direnv.vim", version = "ab2a7e08dd630060cd81d7946739ac7442a4f269" },
  { src = "https://github.com/chrishrb/gx.nvim", version = "ba9c408fc0130fc4548760c3933a81b58fc50de8" },
  { src = "https://github.com/anuvyklack/middleclass", version = "9fab4d5bca67262614960960ca35c4740eb2be2c" },
  { src = "https://github.com/anuvyklack/animation.nvim", version = "fb77091ab72ec9971aee0562e7081182527aaa6a" },
  { src = "https://github.com/anuvyklack/windows.nvim", version = "c7492552b23d0ab30325e90b56066ec51242adc8" },
  { src = "https://github.com/NvChad/nvim-colorizer.lua", version = "72a05f62c52241bc7441c820eb53946f92b2e6a4" },
  { src = "https://github.com/stevearc/dressing.nvim", version = "2d7c2db2507fa3c4956142ee607431ddb2828639" },
  { src = "https://github.com/ethanholz/nvim-lastplace", version = "0bb6103c506315044872e0f84b1f736c4172bb20" },
  { src = "https://github.com/j-hui/fidget.nvim", version = "9e0201673e08e997e7cf52afca5565c70bd117f3" },
  { src = "https://github.com/ThePrimeagen/harpoon", version = "87b1a3506211538f460786c23f98ec63ad9af4e5" }, -- branch: harpoon2
  { src = "https://github.com/akinsho/bufferline.nvim", version = "655133c3b4c3e5e05ec549b9f8cc2894ac6f51b3" },
  { src = "https://github.com/folke/noice.nvim", version = "7bfd942445fb63089b59f97ca487d605e715f155" },
  { src = "https://github.com/sindrets/diffview.nvim", version = "4516612fe98ff56ae0415a259ff6361a89419b0a" },
  { src = "https://github.com/akinsho/git-conflict.nvim", version = "a1badcd070d176172940eb55d9d59029dad1c5a6" },
  { src = "https://github.com/lewis6991/gitsigns.nvim", version = "1e9d0fbfe7ff848bc402d2c2e2854fb4eff2dce7" },
  { src = "https://github.com/nvim-lualine/lualine.nvim", version = "221ce6b2d999187044529f49da6554a92f740a96" },
  { src = "https://github.com/rmagatti/auto-session", version = "02a5588f677970a9bc784115c62bef9e5855499b" },
  { src = "https://github.com/glacambre/firenvim", version = "302c105ff79c51d3d362eb37210f95f65285926d" },
  { src = "https://github.com/niksingh710/leetcode.nvim", version = "cab77da208b0abca7070ef4213bc4cea1a339ada" }, -- held
  { src = "https://github.com/L3MON4D3/LuaSnip", version = "0abc8f390b278c3b4aabc4c004ac8a088b65cf24" },
  { src = "https://github.com/saghen/blink.cmp", version = "v1.10.2" },
  { src = "https://github.com/nvim-treesitter/nvim-treesitter", version = "c6dd314086f7b471bf6c9110092a94ce1c06d220" }, -- held
  { src = "https://github.com/nvim-treesitter/nvim-treesitter-context", version = "8decd73460d26ef0a0befb745d022dd1fc2c38b8" },
  { src = "https://github.com/echasnovski/mini.ai", version = "a4f5648dedf435e6306205135d1babb563b55d28" },
  { src = "https://github.com/tpope/vim-sleuth", version = "be69bff86754b1aa5adcbb527d7fcd1635a84080" },
  { src = "https://github.com/mason-org/mason.nvim", version = "2a6940af80375532e5e9e7c1f2fc6319a1b7a69d" },
  { src = "https://github.com/WhoIsSethDaniel/mason-tool-installer.nvim", version = "443f1ef8b5e6bf47045cb2217b6f748a223cf7dc" },
  { src = "https://github.com/neovim/nvim-lspconfig", version = "4b2b045b8ceb035f5c8bc9cab19b95706bfa6a19" },
  { src = "https://github.com/stevearc/conform.nvim", version = "016802de402556da54c36bd7359b441266b01cdd" },
  { src = "https://github.com/folke/trouble.nvim", version = "bd67efe408d4816e25e8491cc5ad4088e708a69a" },
  { src = "https://github.com/rachartier/tiny-inline-diagnostic.nvim", version = "46ae9b3c8f3039e96ff4497d190ad0aee091f30d" },
  { src = "https://github.com/nvimdev/lspsaga.nvim", version = "cf6fc9473bba1d332eda9887855ea29ed9b37701" },
  { src = "https://github.com/kevinhwang91/promise-async", version = "119e8961014c9bfaf1487bf3c2a393d254f337e2" },
  { src = "https://github.com/kevinhwang91/nvim-ufo", version = "ab3eb124062422d276fae49e0dd63b3ad1062cfc" },
  { src = "https://github.com/mrcjkb/rustaceanvim", version = "b7f84928e254bce1cf9ca6afda5f1cfd5ba01591" },
  { src = "https://github.com/saecki/crates.nvim", version = "7039bc1d130e424232a7aa64ceddb170a154ff5b" },
  { src = "https://github.com/HakonHarnes/img-clip.nvim", version = "99848daf801789dbebe2192b475eb887d375002c" },
  { src = "https://github.com/iamcco/markdown-preview.nvim", version = "a923f5fc5ba36a3b17e289dc35dc17f66d0548ee" },
  { src = "https://github.com/MeanderingProgrammer/render-markdown.nvim", version = "8e8795368391036643b622875a6da6a51bcd33f4" },
  { src = "https://github.com/ellisonleao/glow.nvim", version = "5d5954b2f22e109d4a6eba8b2618c5b96e4ee7a2" },
  { src = "https://github.com/chomosuke/typst-preview.nvim", version = "1c2e19486397be1c580b560fc50ee36abe329c46" },
  { src = "https://github.com/mistweaverco/kulala.nvim", version = "125751ea9f41c1ee966c5c7e3032e3539dd2f33e" },
  { src = "https://github.com/windwp/nvim-ts-autotag", version = "d7220b55e16ed3c04e88086256def5d282429493" },
  { src = "https://github.com/folke/ts-comments.nvim", version = "a59d6092213447450191122c9346f309161504cb" },
  { src = "https://github.com/mistricky/codesnap.nvim", version = "fccc5c7a2a10ad04da83d0263d50dde7f6da2430" },
  { src = "https://github.com/vyfor/cord.nvim", version = "85f295e43550bc968b66d95d8bfd6e12bb17df08" },
  { src = "https://github.com/amitds1997/remote-nvim.nvim", version = "bd8c33df43484948c9c92a58aea28b844331ba2f" },
}

function M.setup()
  vim.g.firenvim_config = {
    globalSettings = { cmdline = "neovim", content = "text", priority = 0, selector = "textarea", takeover = "never" },
    localSettings = { [".*"] = { takeover = "never", cmdline = "neovim" } },
  }
  vim.g.rustaceanvim = { server = { auto_attach = function()
    local tools = require("hephaestus.tools")
    return tools.ready(tools.servers.rust_analyzer) and vim.fn.executable("rust-analyzer") == 1
  end } }
  vim.g.mkdp_echo_preview_url = 1
  vim.g.mkdp_open_to_the_world = 0
  vim.g.mkdp_browserfunc = "MkdpNoOp"
  vim.cmd("function! MkdpNoOp(url)\nendfunction")

  vim.api.nvim_create_autocmd("PackChanged", {
    callback = function(event)
      if event.data.kind ~= "install" and event.data.kind ~= "update" then return end
      local name, path = event.data.spec.name, event.data.path
      if name == "codesnap.nvim" then
        local result = vim.system({ "make" }, { cwd = path, text = true }):wait()
        if result.code ~= 0 then
          error("CodeSnap build failed: " .. (result.stderr or "") .. "\nRun make in " .. path .. " to retry.")
        end
      elseif name == "markdown-preview.nvim" then
        vim.cmd("packadd markdown-preview.nvim")
        vim.fn["mkdp#util#install_sync"]()
        local installed = false
        for _, binary in ipairs(vim.fn.glob(path .. "/app/bin/markdown-preview-*", false, true)) do
          if vim.fn.executable(binary) == 1 then installed = true end
        end
        assert(installed,
          "Markdown preview installation failed; retry :call mkdp#util#install_sync()")
      elseif name == "nvim-treesitter" and event.data.kind == "update" then
        vim.schedule(function() vim.cmd("TSUpdate") end)
      end
    end,
  })
  vim.pack.add(M.specs, { confirm = false, load = true })
end

return M
