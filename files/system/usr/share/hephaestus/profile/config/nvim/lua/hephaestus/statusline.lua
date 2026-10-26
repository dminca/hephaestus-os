local transparent = { a = { fg = "none" }, c = { bg = "none" } }
require("lualine").setup({
  options = {
    theme = {
      normal = transparent, insert = transparent, visual = transparent,
      replace = transparent, command = transparent, inactive = transparent,
    },
    always_divide_middle = true, globalstatus = true, icons_enabled = true,
    component_separators = { left = "", right = "" }, section_separators = { left = "", right = "" },
    disabled_filetypes = { "Outline", "neo-tree", "dashboard", "snacks_dashboard", "snacks_terminal" },
  },
  sections = {
    lualine_a = { "fileformat" }, lualine_b = { "encoding" },
    lualine_c = {
      "b:gitsigns_head",
      { "diff", source = function()
        local signs = vim.b.gitsigns_status_dict
        if signs then return { added = signs.added, modified = signs.changed, removed = signs.removed } end
      end },
      { "diagnostics", sources = { "nvim_diagnostic" } },
    },
    lualine_x = {
      { function() return require("noice").api.status.mode.get() end,
        cond = function() return package.loaded.noice and require("noice").api.status.mode.has() end,
        color = { fg = "#ff9e64" } },
      function()
        local names, seen = {}, {}
        for _, client in ipairs(vim.lsp.get_clients({ bufnr = 0 })) do
          if client.name ~= "copilot" and client.name ~= "null-ls" and client.name ~= "typos_lsp" then
            local name = client.name:gsub("%[%d+%]", "")
            if not seen[name] then table.insert(names, name); seen[name] = true end
          end
        end
        for _, formatter in ipairs(require("conform").list_formatters()) do
          if formatter.available and not seen[formatter.name]
            and not vim.tbl_contains({ "squeeze_blanks", "trim_whitespace", "trim_newlines" }, formatter.name) then
            table.insert(names, formatter.name)
            seen[formatter.name] = true
          end
        end
        return #names == 0 and "Ls Inactive" or "[" .. table.concat(names, ", ") .. "]"
      end,
      "filetype",
    },
    lualine_y = { "progress" }, lualine_z = { "location" },
  },
})
vim.api.nvim_set_hl(0, "StatusLine", { bg = "NONE" })
