local function map(mode, key, action, description, extra)
  vim.keymap.set(mode, key, action, vim.tbl_extend("force", {
    silent = true, desc = description,
  }, extra or {}))
end
map("i", "jk", "<Esc>", "Normal mode")
map({ "n", "i", "v" }, "<C-s>", "<Esc><cmd>w ++p<cr>", "Save file")
map("v", "<C-c>", "<Esc>", "Escape")
map("v", "<", "<gv", "Indent out")
map("v", ">", ">gv", "Indent in")
map("v", "<Space>", "<Nop>", "No action")
map("n", "<A-j>", "<cmd>m .+1<cr>==", "Move line down")
map("n", "<A-k>", "<cmd>m .-2<cr>==", "Move line up")
map("i", "<A-j>", "<Esc>:m .+1<cr>==gi", "Move line down")
map("i", "<A-k>", "<Esc>:m .-2<cr>==gi", "Move line up")
map("v", "<A-j>", ":m '>+1<cr>gv-gv", "Move selection down")
map("v", "<A-k>", ":m '<-2<cr>gv-gv", "Move selection up")
map("n", "<A-+>", "<C-a>", "Increase number")
map("n", "<A-->", "<C-x>", "Decrease number")
for _, key in ipairs({ "j", "k" }) do
  map({ "n", "x" }, key, 'v:count || mode(1)[0:1] == "no" ? "' .. key .. '" : "g' .. key .. '"',
    "Move " .. key, { expr = true })
end
for _, key in ipairs({ "x", "X", "c", "C" }) do
  map("n", key, '"_' .. key, "Keep register")
  map("v", key, '"_' .. ((key == "x" or key == "X") and "d" or key), "Keep register")
end
map("v", "p", '"_dP', "Paste without replacing clipboard")
map("c", "<C-j>", "<C-n>", "Next command")
map("c", "<C-k>", "<C-p>", "Previous command")
map("n", "n", "nzzzv", "Centered next match")
map("n", "N", "Nzzzv", "Centered previous match")
map("n", "<leader>qq", "<cmd>quitall!<cr>", "Quit")
map("n", "<leader>qw", function()
  if #vim.api.nvim_tabpage_list_wins(0) > 1 then
    vim.cmd.close()
  else
    vim.notify("Cannot close the last window", vim.log.levels.WARN)
  end
end, "Close window")
map("n", "<leader><leader>", "<cmd>nohlsearch<cr>", "Clear search")
map("n", "<Esc>", "<Esc><cmd>nohlsearch<cr>", "Escape")
map("n", "<leader>A", "ggVG", "Select all")
map("n", "<leader>|", "<cmd>vsplit<cr>", "Vertical split")
map("n", "<leader>-", "<cmd>split<cr>", "Horizontal split")
map("n", "<leader>cn", "<cmd>cnext<cr>", "Next quickfix")
map("n", "<leader>cp", "<cmd>cprev<cr>", "Previous quickfix")
map("n", "<leader>cq", "<cmd>cclose<cr>", "Close quickfix")
map("n", "<leader>dd", function()
  local diff = false
  for _, win in ipairs(vim.api.nvim_tabpage_list_wins(0)) do
    diff = diff or vim.wo[win].diff
  end
  vim.cmd(diff and "windo diffoff" or "windo diffthis")
end, "Toggle diff")
map("n", "<leader>id", function()
  vim.api.nvim_put({ "# " .. os.date("%d-%m-%y") }, "c", false, true)
end, "Insert date")
map("n", "<leader>ft", function()
  vim.ui.input({ prompt = "Enter FileType: " }, function(value)
    if value and value ~= "" then vim.bo.filetype = value end
  end)
end, "Set filetype")
for key, direction in pairs({ h = "left", j = "down", k = "up", l = "right" }) do
  map("n", "<C-" .. key .. ">", function() require("smart-splits")["move_cursor_" .. direction]() end, "Move " .. direction)
  map("n", "<C-A-" .. key .. ">", function() require("smart-splits")["resize_" .. direction]() end, "Resize " .. direction)
end
map("n", "<C-\\>", function() require("smart-splits").move_cursor_previous() end, "Previous pane")
map("n", "?", function() require("flash").jump({ forward = true, wrap = true, multi_window = true }) end, "Flash search")
map("n", "<leader>vt", function() require("flash").treesitter() end, "Select Treesitter node")
map("n", "<leader>uC", function() require("stay-centered").toggle() end, "Toggle centering")
map("n", "<leader>ut", "<cmd>TrimToggle<cr>", "Toggle trim")
map("v", "<leader>us", "<cmd>CodeSnap<cr>", "Code snapshot")
map("n", "<leader>b.", function() require("harpoon"):list():add() end, "Add Harpoon file")
map("n", "<leader>bb", function()
  local harpoon = require("harpoon")
  harpoon.ui:toggle_quick_menu(harpoon:list())
end, "Harpoon")
for key, command in pairs({
  bp = "BufferLinePick", bP = "BufferLineTogglePin", bd = "BufferLineSortByDirectory",
  be = "BufferLineSortByExtension", bt = "BufferLineSortByTabs", bL = "BufferLineCloseRight",
  bH = "BufferLineCloseLeft", bc = "BufferLineCloseOther", qc = "bp | bd #",
  qt = "tabclose", ["<Tab>q"] = "tabclose", ["<Tab>n"] = "tab split",
  ["<Tab>j"] = "tabnext", ["<Tab>l"] = "tabnext", ["<Tab>k"] = "tabprevious",
  ["<Tab>h"] = "tabprevious", ["q."] = "AutoSession restore", ql = "AutoSession search",
  qd = "AutoSession deletePicker", qs = "AutoSession save", qD = "AutoSession purgeOrphaned",
}) do
  map("n", "<leader>" .. key, "<cmd>" .. command .. "<cr>", command)
end
map("n", "<S-h>", "<cmd>BufferLineCyclePrev<cr>", "Previous buffer")
map("n", "<S-l>", "<cmd>BufferLineCycleNext<cr>", "Next buffer")
map("n", "<A-S-h>", "<cmd>BufferLineMovePrev<cr>", "Move buffer left")
map("n", "<A-S-l>", "<cmd>BufferLineMoveNext<cr>", "Move buffer right")
for key, command in pairs({
  la = "Lspsaga code_action", lo = "Lspsaga outline", lw = "Lspsaga show_workspace_diagnostics",
  lr = "Lspsaga rename ++project", ["l."] = "Lspsaga show_line_diagnostics",
  li = "LspInfo", lq = "LspStop", lR = "LspRestart",
}) do
  map("n", "<leader>" .. key, "<cmd>" .. command .. "<cr>", command)
end
for key, command in pairs({
  gd = "Trouble lsp_definitions", gr = "Trouble lsp_references",
  gt = "Lspsaga goto_type_definition", gpd = "Lspsaga peek_definition",
  gpt = "Lspsaga peek_type_definition", ["[e"] = "Lspsaga diagnostic_jump_prev",
  ["]e"] = "Lspsaga diagnostic_jump_next",
}) do
  map("n", key, "<cmd>" .. command .. "<cr>", command)
end
map("n", "[d", function() vim.diagnostic.jump({ count = -1 }) end, "Previous diagnostic")
map("n", "]d", function() vim.diagnostic.jump({ count = 1 }) end, "Next diagnostic")
map("n", "<C-S-k>", vim.lsp.buf.signature_help, "Signature help")
map({ "n", "x" }, "<leader>lf", function() require("conform").format() end, "Format")
map("n", "<leader>lL", function() vim.diagnostic.enable(not vim.diagnostic.is_enabled()) end, "Toggle diagnostics")
map("n", "<leader>ll", function()
  vim.diagnostic.config({ virtual_text = not vim.diagnostic.config().virtual_text and { source = "always" } or false })
end, "Toggle virtual text")
map("n", "zR", function() require("ufo").openAllFolds() end, "Open folds")
map("n", "zM", function() require("ufo").closeAllFolds() end, "Close folds")
map("n", "zK", function()
  if not require("ufo").peekFoldedLinesUnderCursor() then vim.lsp.buf.hover() end
end, "Peek fold")
map("n", "K", function()
  if not require("ufo").peekFoldedLinesUnderCursor() then vim.cmd("Lspsaga hover_doc") end
end, "Hover")
