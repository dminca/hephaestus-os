package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyboardRepeatPreferences(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required for Plasma update script tests")
	}
	script := filepath.Join(repository(t), "files/system/usr/share/plasma/shells/org.kde.plasma.desktop/contents/updates/hephaestus-key-repeat.js")
	command := exec.Command(node, "-e", `
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const script = fs.readFileSync(process.argv[1], "utf8");
const values = {NumLock: 2};
const context = vm.createContext({
    ConfigFile: function(file, group) {
        assert.equal(file, "kcminputrc");
        assert.equal(group, "Keyboard");
        this.writeEntry = (key, value) => {values[key] = value; return true;};
    },
});
for (let attempt = 0; attempt < 2; attempt++) {
    vm.runInContext(script, context);
    assert.deepEqual(values, {NumLock: 2, KeyRepeat: "repeat", RepeatDelay: 233, RepeatRate: 17});
}
const failure = vm.createContext({
    ConfigFile: function() {this.writeEntry = () => false;},
});
assert.throws(() => vm.runInContext(script, failure), /could not persist/);
`, script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Keyboard repeat update failed: %v\n%s", err, output)
	}
}

func TestNextcloudAutostart(t *testing.T) {
	root := repository(t)
	desktop := readFile(t, filepath.Join(root, "files/system/etc/xdg/autostart/com.nextcloud.desktopclient.nextcloud.desktop"))
	for _, entry := range []string{
		"Type=Application",
		"Exec=/usr/bin/flatpak run com.nextcloud.desktopclient.nextcloud --background",
		"TryExec=/usr/bin/flatpak",
	} {
		if !strings.Contains(desktop, entry+"\n") {
			t.Fatalf("Nextcloud autostart is missing %s", entry)
		}
	}
}
