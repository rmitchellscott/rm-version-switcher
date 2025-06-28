Output rm-version-switcher.mp4

Set Framerate 30

Set Theme "Builtin Dark"
Set FontSize 17
Set Width 970
Set Height 720
Set Padding 40
Set Margin 20
Set WindowBar Colorful
Set MarginFill "#7A6F64"
Set BorderRadius 10

Hide
Type@0 'rm-version-switcher() ( sleep 0.1 && exec go run main.go --dry-run "$@"); clear'
Enter
Sleep 100ms
Show
Sleep 50ms

Type "rm-version-switcher"
Sleep 1000ms
Enter
Sleep 1000ms

# Change boot Selector
Sleep 1000ms
Left
Sleep 1000ms
Enter
Sleep 2000ms

# Partition selector
Up
Sleep 1000ms
Enter
Sleep 2000ms

#Reboot selector
Left
Sleep 1000ms
Enter
Sleep 1000ms
