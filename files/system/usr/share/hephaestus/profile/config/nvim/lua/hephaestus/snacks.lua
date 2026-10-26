local function map(key, action, description, mode)
  vim.keymap.set(mode or "n", key, action, { silent = true, desc = description })
end
local picker_keys = {
  ["<C-d>"] = { "preview_scroll_down", mode = "n" },
  ["<C-u>"] = { "preview_scroll_up", mode = "n" },
  ["-"] = { "edit_split", mode = "n" },
  ["|"] = { "edit_vsplit", mode = "n" },
}
require("snacks").setup({
  bigfile = { enabled = true }, scroll = { enabled = false }, animate = { enabled = true },
  quickfile = { enabled = true }, indent = { enabled = true }, words = { enabled = true },
  statuscolumn = { enabled = true },
  image = { enabled = true, border = "none", doc = { conceal = true, inline = true } },
  notifier = { enabled = true, style = "minimal", top_down = false },
  styles = { ziggity = { width = 0, height = 0 } },
  dashboard = {
    enabled = true,
    sections = {
      { section = "header" },
      { pane = 2, section = "terminal", cmd = "colorscript -e square", height = 5, padding = 1 },
      { section = "keys", gap = 1, padding = 1 },
      { pane = 2, desc = "Browse repo", key = "b", action = function() Snacks.gitbrowse() end },
      function()
        local in_git = Snacks.git.get_root() ~= nil
        local commands = {
          { title = "Notifications",
            cmd = [[gh api notifications -q '.[0:5][] | (.repository.full_name + ": " + .subject.title)']],
            action = function() vim.ui.open("https://github.com/notifications") end, key = "n", height = 5 },
          { title = "Open issues", cmd = "PAGER= gh issue list -L 3", key = "i", height = 7,
            action = function() vim.fn.jobstart({ "gh", "issue", "list", "--web" }, { detach = true }) end },
          { title = "Open PRs", cmd = "PAGER= gh pr list -L 3", key = "P", height = 7,
            action = function() vim.fn.jobstart({ "gh", "pr", "list", "--web" }, { detach = true }) end },
          { title = "Git status", cmd = "git --no-pager diff --stat -B -M -C", height = 10 },
        }
        return vim.tbl_map(function(command)
          return vim.tbl_extend("force", {
            pane = 2, section = "terminal", enabled = in_git, padding = 1, ttl = 300, indent = 3,
          }, command)
        end, commands)
      end,
    },
  },
  picker = {
    enabled = true, layout = { preset = "ivy" }, layouts = { vscode = { layout = { width = 0.7 } } },
    actions = {
      pick_win = function(picker)
        if not picker.layout.split then picker.layout:hide() end
        local win = Snacks.picker.util.pick_win({
          main = picker.main, float = false,
          filter = function(_, buf)
            local ft = vim.bo[buf].filetype
            return ft == "snacks_dashboard" or not ft:find("^snacks")
          end,
        })
        if win then picker.main = win end
        if not picker.layout.split then
          vim.defer_fn(function() if not picker.closed then picker.layout:unhide() end end, 100)
        end
        return not win
      end,
    },
    win = {
      input = { keys = picker_keys },
      list = { keys = vim.tbl_extend("force", picker_keys, { ["<Enter>"] = { { "pick_win", "jump" }, mode = { "n", "i" } } }) },
    },
    sources = { explorer = {
      layout = {
        preset = "sidebar", preview = false,
        layout = {
          backdrop = false, width = 40, min_width = 40, height = 0, position = "right",
          border = "none", box = "vertical", { win = "list", border = "none" },
          { win = "preview", title = "{preview}", height = 0.4, border = "top" },
        },
      },
      on_show = function(picker)
        local relative = picker.layout.root
        local position = picker.resolved_layout.layout.position
        local function update(win)
          local border = win:border_size().left + win:border_size().right
          win.opts.row = vim.api.nvim_win_get_position(relative.win)[1]
          win.opts.height = 0.8
          if position == "left" then
            win.opts.col = vim.api.nvim_win_get_width(relative.win) + 1
            win.opts.width = math.max(20, math.min(100, vim.o.columns - border - win.opts.col))
          else
            win.opts.col = -vim.api.nvim_win_get_width(relative.win) - 1
            win.opts.width = math.max(20, math.min(100, vim.o.columns - border + win.opts.col))
          end
          win:update()
        end
        local preview = Snacks.win.new({
          relative = "editor", external = false, focusable = false, border = "rounded",
          backdrop = false, show = false,
          bo = { filetype = "snacks_float_preview", buftype = "nofile", buflisted = false, swapfile = false, undofile = false },
          on_win = function(win) update(win); picker:show_preview() end,
        })
        relative:on("WinLeave", function()
          vim.schedule(function() if not picker:is_focused() then picker.preview.win:close() end end)
        end)
        relative:on("WinResized", function() update(preview) end)
        picker.preview.win = preview
        picker.main = preview.win
      end,
      on_close = function(picker) picker.preview.win:close() end,
      actions = { toggle_preview = function(picker) picker.preview.win:toggle() end },
    } },
  },
})
map("<leader>e", function() Snacks.explorer() end, "Explorer")
map("<leader>..", function() Snacks.scratch() end, "Toggle scratch")
map("<leader>.s", function() Snacks.scratch.select() end, "Select scratch")
map("<leader>.r", function() Snacks.rename.rename_file() end, "Rename file")
map("<leader>sn", function() Snacks.notifier.show_history() end, "Notification history")
map("<leader>un", function() Snacks.notifier.hide() end, "Dismiss notifications")
map("<leader>gg", function()
  Snacks.terminal({ "ziggity" }, { win = { style = "ziggity" } })
end, "Ziggity")
for key, name in pairs({
  sb = "lines", sB = "grep_buffers", sg = "grep", sw = "grep_word", ['s"'] = "registers",
  sa = "autocmds", sc = "command_history", sC = "commands", sd = "diagnostics",
  sD = "diagnostics_buffer", sH = "highlights", si = "icons", sj = "jumps",
  sl = "loclist", sm = "marks", sM = "man", sq = "qflist", sR = "resume",
  sS = "lsp_workspace_symbols", ss = "smart", st = "todo_comments", ["s:"] = "command_history",
  ["s,"] = "buffers", sh = "help", ff = "files", fF = "smart", ["f/"] = "grep",
  fr = "recent", fp = "projects", lD = "lsp_definitions", ls = "lsp_symbols",
}) do
  map("<leader>" .. key, function() Snacks.picker[name]() end, name)
end
map("<leader>sw", function() Snacks.picker.grep_word() end, "Grep selection", "x")
map("gD", function() Snacks.picker.lsp_declarations() end, "Declaration")
map("gI", function() Snacks.picker.lsp_implementations() end, "Implementation")
map("gy", function() Snacks.picker.lsp_type_definitions() end, "Type definition")
map("<leader>sP", function() Snacks.picker({ layout = "vscode" }) end, "Pickers")
map("<leader>sk", function() Snacks.picker.keymaps({ layout = "vscode" }) end, "Keymaps")
map("<leader>sT", function() Snacks.picker.todo_comments({ keywords = { "TODO", "FIX", "FIXME" } }) end, "Todo")
map("<leader>f?", function()
  Snacks.picker.grep({ args = { "--vimgrep", "--smart-case", "--fixed-strings" } })
end, "Literal grep")
map("<leader>su", function()
  Snacks.picker.undo({ win = { input = { keys = { y = { "yank_add", mode = "n" }, Y = { "yank_del", mode = "n" } } } } })
end, "Undo")
vim.api.nvim_create_autocmd("VimEnter", {
  once = true,
  callback = function()
    _G.dd = function(...) Snacks.debug.inspect(...) end
    _G.bt = function() Snacks.debug.backtrace() end
    vim._print = function(_, ...) _G.dd(...) end
    Snacks.toggle.diagnostics():map("<leader>ud")
    Snacks.toggle.line_number():map("<leader>ul")
    Snacks.toggle.inlay_hints():map("<leader>uh")
    Snacks.toggle.treesitter():map("<leader>uT")
    Snacks.toggle.option("spell", { name = "Spelling" }):map("<leader>us")
    Snacks.toggle.option("wrap", { name = "Wrap" }):map("<leader>uw")
    Snacks.toggle.option("relativenumber", { name = "Relative number" }):map("<leader>uL")
    Snacks.toggle.option("conceallevel", { off = 0, on = 2 }):map("<leader>uc")
    Snacks.toggle.option("background", { off = "light", on = "dark", name = "Dark background" }):map("<leader>ub")
  end,
})
