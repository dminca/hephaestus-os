local servers = assert(vim.g.hephaestus_language_server_commands, "Hephaestus LSP configuration did not load")
local failures = {}
local count = 0
for name, command in pairs(servers) do
  count = count + 1
  if type(command) ~= "string" then
    failures[#failures + 1] = name .. ": no executable command configured"
  elseif vim.fn.executable(command) ~= 1 then
    failures[#failures + 1] = name .. ": missing " .. command
  end
end
table.sort(failures)
assert(#failures == 0, "Language server validation failed:\n" .. table.concat(failures, "\n"))
io.stdout:write("All " .. count .. " configured language server commands are installed\n")
