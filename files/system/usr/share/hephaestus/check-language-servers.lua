local function fail(message)
  io.stderr:write(message .. "\n")
  vim.cmd("cquit 1")
end
local servers = vim.g.hephaestus_language_server_commands
if not servers then
  fail("Hephaestus LSP configuration did not load")
end
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
if #failures > 0 then
  fail("Language server validation failed:\n" .. table.concat(failures, "\n"))
end
io.stdout:write("All " .. count .. " configured language server commands are installed\n")
