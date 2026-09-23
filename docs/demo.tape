# Terminal demo for the README.
#
#   vhs docs/demo.tape          # writes docs/demo.gif
#
# Needs vhs, ttyd and ffmpeg (brew install vhs ttyd ffmpeg) and a c3x on
# PATH. Run it from the repository root: the demo estimates the checked
# in examples/quickstart project, so it is reproducible and the numbers
# on screen are real prices from pricing.c3x.dev.
#
# The setup block is hidden so only the command appears. It warms the
# price cache first, so the speed on screen is a genuine warm run rather
# than a first network fetch. Height is sized to the output: change the
# example and the frame will need resizing with it.

Output docs/demo.gif

Require c3x

Set Shell "bash"
Set FontSize 18
Set Width 960
Set Height 510
Set Padding 22
Set TypingSpeed 55ms
Set Theme "Catppuccin Mocha"

Hide
Type "export PS1='$ '" Enter
Type "cd examples/quickstart" Enter
Type "c3x estimate >/dev/null 2>&1" Enter
Sleep 10s
Type "clear" Enter
Show

Sleep 800ms
Type "c3x estimate"
Sleep 400ms
Enter
Sleep 3500ms
