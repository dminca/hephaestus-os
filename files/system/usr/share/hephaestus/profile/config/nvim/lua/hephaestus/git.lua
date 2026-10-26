local gs = require("gitsigns")
local function map(mode, key, action, desc)
  vim.keymap.set(mode, key, action, { desc = desc, silent = true })
end
for key, direction in pairs({ ["]h"] = "next", ["[h"] = "prev", ["]H"] = "last", ["[H"] = "first" }) do
  map("n", key, function()
    if vim.wo.diff and (direction == "next" or direction == "prev") then
      vim.cmd.normal({ direction == "next" and "]c" or "[c", bang = true })
    else
      gs.nav_hunk(direction)
    end
  end, direction .. " hunk")
end
for key, action in pairs({
  gs = "stage_buffer", gr = "reset_buffer", gu = "undo_stage_hunk",
  gp = "preview_hunk_inline", gP = "preview_hunk", gK = "blame",
  gb = "toggle_current_line_blame", gw = "toggle_word_diff",
}) do
  map("n", "<leader>" .. key, function() gs[action]() end, action)
end
map("n", "<leader>gk", function() gs.blame_line({ full = true }) end, "Full blame")
for key, action in pairs({ gs = "stage_hunk", gr = "reset_hunk", gp = "preview_hunk_inline" }) do
  map("v", "<leader>" .. key, function() gs[action]({ vim.fn.line("."), vim.fn.line("v") }) end, action)
end
map("n", "<leader>gdd", "<cmd>DiffviewOpen<cr>", "Diffview")
map("n", "<leader>gdh", "<cmd>DiffviewFileHistory<cr>", "File history")
map("n", "<leader>gC", "<cmd>GitConflictListQf<cr>", "Conflicts to quickfix")
map("n", "<leader>ghH", function() gs.setqflist() end, "Buffer hunks")
map("n", "<leader>ghh", function() gs.setqflist("all") end, "All hunks")
map({ "o", "x" }, "ih", ":<C-U>Gitsigns select_hunk<cr>", "Select hunk")
