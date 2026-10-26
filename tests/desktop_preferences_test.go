package tests

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDesktopPreferences(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required for Plasma update script tests")
	}
	script := filepath.Join(repository(t), "files/system/usr/share/plasma/shells/org.kde.plasma.desktop/contents/updates/hephaestus-desktop-preferences.js")
	command := exec.Command(node, "-e", `
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const script = fs.readFileSync(process.argv[1], "utf8");
for (const initial of [undefined, "", "compose:ralt,caps:ctrl_modifier,grp:alt_shift_toggle", "ctrl:nocaps", "ctrl:swapcaps"]) {
    const values = {Options: initial, LayoutList: "us,ro"};
    const clock = {
        type: "org.kde.plasma.digitalclock",
        currentConfigGroup: ["Existing"],
        writeConfig(key, value) {
            assert.equal(this.currentConfigGroup.join("/"), "Appearance");
            assert.equal(key, "showSeconds");
            assert.equal(value, 2);
            this.seconds = value;
        },
    };
    const other = {type: "org.kde.plasma.taskmanager", writeConfig() {assert.fail("unrelated widget changed");}};
    const context = vm.createContext({
        panels: () => [{widgets: () => [clock, other]}],
        desktops: () => [],
        ConfigFile: function(file, group) {
            assert.equal(file, "kxkbrc");
            assert.equal(group, "Layout");
            this.readEntry = key => values[key];
            this.writeEntry = (key, value) => {values[key] = value; return true;};
        },
    });
    for (let attempt = 0; attempt < 2; attempt++) {
        vm.runInContext(script, context);
        assert.equal(clock.seconds, 2);
        assert.equal(clock.currentConfigGroup.join("/"), "Existing");
        assert.equal(values.Options, initial && initial.includes("compose:")
            ? "compose:ralt,grp:alt_shift_toggle,caps:escape" : "caps:escape");
        assert.equal(values.ResetOldOptions, true);
        assert.equal(values.LayoutList, "us,ro");
    }
}
const failure = vm.createContext({
    panels: () => [], desktops: () => [],
    ConfigFile: function() {this.readEntry = () => ""; this.writeEntry = () => false;},
});
assert.throws(() => vm.runInContext(script, failure), /could not persist/);
`, script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Plasma preference update failed: %v\n%s", err, output)
	}
}
