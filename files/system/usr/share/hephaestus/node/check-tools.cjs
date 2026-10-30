const { accessSync, constants } = require("node:fs");
const path = require("node:path");

const executables = [
  "bash-language-server", "typescript-language-server", "tsserver", "tsc",
  "vscode-css-language-server", "vscode-eslint-language-server",
  "vscode-html-language-server", "vscode-json-language-server",
  "tailwindcss-language-server", "svelteserver", "yaml-language-server",
  "emmet-ls", "biome", "pyright", "pyright-langserver", "fixjson",
];

function checkTools(directory) {
  for (const executable of executables) {
    try {
      accessSync(path.join(directory, executable), constants.X_OK);
    } catch (error) {
      throw new Error(`Missing executable: ${executable}`, { cause: error });
    }
  }
}

if (require.main === module) {
  checkTools(path.join(__dirname, "node_modules/.bin"));
  if (process.argv.includes("--list")) console.log(executables.join("\n"));
}

module.exports = { checkTools, executables };
