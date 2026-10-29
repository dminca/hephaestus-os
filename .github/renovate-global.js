module.exports = {
  autodiscover: false,
  onboarding: false,
  repositories: ["dminca/hephaestus-os"],
  binarySource: "global",
  allowedCommands: [
    "^go run \\.\\/tools\\/lock-profile$",
    "^go run \\.\\/tools\\/lock-binaries$",
    "^bash tools\\/update-flow-desktop-pin\\.sh$",
  ],
};
