module.exports = {
  autodiscover: false,
  onboarding: false,
  repositories: ["dminca/hephaestus-os"],
  binarySource: "global",
  // Renovate filters subprocess environments; the lock tools use gh api.
  customEnvVariables: {
    GH_TOKEN: process.env.GH_TOKEN,
  },
  allowedCommands: [
    "^go run \\.\\/tools\\/lock-profile$",
    "^go run \\.\\/tools\\/lock-binaries$",
    "^bash tools\\/update-flow-desktop-pin\\.sh$",
  ],
};
