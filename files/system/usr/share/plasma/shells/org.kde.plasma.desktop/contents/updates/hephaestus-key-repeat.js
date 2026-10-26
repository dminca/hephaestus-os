var repeat = new ConfigFile("kcminputrc", "Keyboard");
if (!repeat.writeEntry("KeyRepeat", "repeat")
    || !repeat.writeEntry("RepeatDelay", 233)
    || !repeat.writeEntry("RepeatRate", 17)) {
    throw new Error("Hephaestus: could not persist keyboard repeat settings");
}
