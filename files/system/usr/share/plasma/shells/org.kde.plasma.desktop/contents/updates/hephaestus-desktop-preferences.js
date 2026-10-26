var containments = panels().concat(desktops());
for (var i = 0; i < containments.length; i++) {
    var widgets = containments[i].widgets();
    for (var j = 0; j < widgets.length; j++) {
        var widget = widgets[j];
        if (widget.type === "org.kde.plasma.digitalclock") {
            var previousGroup = widget.currentConfigGroup;
            widget.currentConfigGroup = ["Appearance"];
            widget.writeConfig("showSeconds", 2);
            widget.currentConfigGroup = previousGroup;
        }
    }
}

var keyboard = new ConfigFile("kxkbrc", "Layout");
var options = (keyboard.readEntry("Options") || "").split(",").filter(function (option) {
    return option !== "" && option.indexOf("caps:") !== 0
        && option !== "ctrl:nocaps" && option !== "ctrl:swapcaps";
});
options.push("caps:escape");
if (!keyboard.writeEntry("Options", options.join(","))
    || !keyboard.writeEntry("ResetOldOptions", true)) {
    throw new Error("Hephaestus: could not persist Caps Lock to Escape mapping");
}
