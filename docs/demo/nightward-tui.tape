Output site/public/demo/nightward-tui.gif

Require go

Set Shell "zsh"
Set FontSize 18
Set Width 1280
Set Height 760
Set Padding 18
Set BorderRadius 10
Set WindowBar Colorful
Set TypingSpeed 18ms
Set PlaybackSpeed 1.0
Set Framerate 20
Set Theme "TokyoNight"

Hide
Type "rm -rf /tmp/nightward-fixture-home && mkdir -p /tmp/nightward-fixture-home && cp -R testdata/homes/policy/. /tmp/nightward-fixture-home && NIGHTWARD_HOME=/tmp/nightward-fixture-home go run ./cmd/nw"
Enter
Sleep 2200ms
Show

Type "3"
Sleep 900ms
Type "/"
Type "secret"
Enter
Sleep 900ms
Down
Sleep 700ms
Type "5"
Sleep 900ms
Down
Sleep 700ms
Type "p"
Sleep 900ms
Down
Down
Enter
Sleep 900ms
Type "?"
Sleep 1100ms
Type "q"
