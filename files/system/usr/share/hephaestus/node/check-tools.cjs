const { accessSync, constants } = require("node:fs");
const path = require("node:path");

const executables = [
  "mmdc",
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
