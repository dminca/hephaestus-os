vim.g.mapleader = " "
vim.g.floating_window_options = { border = "rounded" }
local options = {
  clipboard = "unnamedplus", cursorline = true, cursorlineopt = "number",
  pumblend = 0, pumheight = 10, expandtab = true, shiftwidth = 2, smartindent = true,
  tabstop = 2, softtabstop = 2, ignorecase = true, smartcase = true, mouse = "a",
  cmdheight = 0, number = true, relativenumber = true, numberwidth = 2, ruler = false,
  signcolumn = "yes", splitbelow = true, splitright = true, splitkeep = "screen",
  termguicolors = true, conceallevel = 2, undofile = true, wrap = false,
  virtualedit = "block", winminwidth = 10, winwidth = 10, equalalways = false,
  fileencoding = "utf-8", list = true, smoothscroll = true, autoread = true,
  autowrite = true, swapfile = false, updatetime = 500, timeout = true, timeoutlen = 250,
  foldcolumn = "1", foldlevel = 99, foldlevelstart = 99, foldenable = true,
}
for name, value in pairs(options) do
  vim.opt[name] = value
end
vim.opt.whichwrap:append("<>[]hl")
vim.opt.listchars:append("space:·")
vim.opt.fillchars:append({ eob = " " })
if vim.env.SSH_TTY then
  local function paste()
    return { vim.split(vim.fn.getreg('"'), "\n"), "v" }
  end
  vim.g.clipboard = {
    name = "OSC 52",
    copy = {
      ["+"] = require("vim.ui.clipboard.osc52").copy("+"),
      ["*"] = require("vim.ui.clipboard.osc52").copy("*"),
    },
    paste = { ["+"] = paste, ["*"] = paste },
  }
end
vim.diagnostic.config({
  virtual_text = false, underline = true, severity_sort = true,
  float = { border = "rounded", source = "always", focusable = false },
  signs = { text = {
    [vim.diagnostic.severity.ERROR] = "E",
    [vim.diagnostic.severity.WARN] = "W",
    [vim.diagnostic.severity.INFO] = "I",
    [vim.diagnostic.severity.HINT] = "H",
  } },
})
local group = vim.api.nvim_create_augroup("Hephaestus", { clear = true })
vim.api.nvim_create_autocmd("TextYankPost", {
  group = group, callback = function() vim.highlight.on_yank() end,
})
vim.api.nvim_create_autocmd({ "FocusGained", "BufEnter", "CursorHold" }, {
  group = group,
  callback = function()
    if vim.fn.getcmdwintype() == "" then vim.cmd.checktime() end
  end,
})
vim.api.nvim_create_autocmd({ "CursorMovedI", "InsertLeave" }, {
  group = group,
  callback = function()
    if vim.fn.pumvisible() == 0 and not vim.wo.previewwindow and vim.fn.getcmdwintype() == "" then
      vim.cmd.pclose()
    end
  end,
})
local watchers = {}
vim.api.nvim_create_autocmd({ "BufReadPost", "BufWritePost" }, {
  group = group,
  callback = function(event)
    local buf = event.buf
    local path = vim.api.nvim_buf_get_name(buf)
    if path == "" or vim.bo[buf].buftype ~= "" then return end
    if watchers[buf] then watchers[buf]:stop(); watchers[buf]:close() end
    local watcher, error = vim.uv.new_fs_event()
    if not watcher then
      vim.notify("File watcher: " .. tostring(error), vim.log.levels.WARN)
      return
    end
    watchers[buf] = watcher
    local ok, message = watcher:start(path, {}, vim.schedule_wrap(function(err)
      if err then
        vim.notify("File watcher: " .. err, vim.log.levels.WARN)
      elseif vim.api.nvim_buf_is_valid(buf) and not vim.bo[buf].modified then
        vim.api.nvim_buf_call(buf, function() vim.cmd.checktime() end)
      end
    end))
    if not ok then
      watcher:close()
      watchers[buf] = nil
      vim.notify("Cannot watch " .. path .. ": " .. tostring(message), vim.log.levels.WARN)
    end
  end,
})
vim.api.nvim_create_autocmd({ "BufDelete", "VimLeavePre" }, {
  group = group,
  callback = function(event)
    for buf, watcher in pairs(watchers) do
      if event.event == "VimLeavePre" or buf == event.buf then
        watcher:stop()
        watcher:close()
        watchers[buf] = nil
      end
    end
  end,
})
