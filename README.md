# neoray

neoray is a simple and lightweight gui client for neovim.
It's written in golang using glfw and opengl bindings.
Neoray is easy to use and binary size is small. Supports
most of the neovim features. Uses small amount of ram and
leaves no footprints on your computer.

You can install neoray with go install command.
```
go install github.com/hismailbulut/neoray/src@latest
```
Also you can download prebuild binaries from releases page.

Neoray doesn't need any additional configuration, but you
can customize it in your init.vim.

These are some options you can specify for now:

The cursor is moving smoothly in neoray and you can specify
how long it's move takes. Default is 0.06 (1.0 is one second)
You can disable it by setting to 0.
```vim
let neoray_cursor_animation_time=0.06
```

Transparency of the window background. Default is 1 means
no transparency, and 0 is fully transparent. Only background
colors will be transparent, and statusline, tabline and texts
are fully opaque.
```vim
let neoray_background_transparency=1
```

The target update time in one second. Like FPS but neoray
doesn't render screen in every frame. Default is 60.
```vim
let neoray_target_ticks_per_second=60
```

Neoray has a simple right click menu that gives you some abilities
like copying, cutting to system clipboard and pasting. It has a
open file functionality that opens system file dialog. Menu text
is same as the font and the colors are from your colorscheme. This
makes it look and feel like terminal. You can disable it by setting
this option to 0. Default is 1 which means enabled.
NOTE: The old name "neoray_popup_menu_enabled" has changed because of the
possible confuse with ext_popupmenu. If you used old name, you need to change
it in your init.vim.
```vim
let neoray_context_menu_enabled=1
```

You can specify how the neoray window will be shown. The possible
values are 'minimized', 'maximized', 'fullscreen', 'centered'.
Default is none.
```vim
let neoray_window_startup_state=''
```

Also you can specify the startup size of the window in cells.
The default is none. The syntax is same below. Frist value is
width (columns) and second value is height (rows).
```vim
let neoray_window_startup_size='99x33'
```

Neoray uses some key combinations for switching between fullscreen and
windowed mode, zoom in and out eg. You can set these keys and also
disable as you wish. All options here are strings contains vim style
keybindings and set to defaults.
```vim
let neoray_key_toggle_fullscreen='<F11>'
let neoray_key_increase_fontsize='<C-kPlus>'
let neoray_key_decrease_fontsize='<C-kMinus>'
```
___
### font
Neoray respects your guifont option, finds the font and loads it. If it can't
find your font, try with different names and also with file name. Giving full
shared name except the style and weight names will give best result. You can
change the font without needing restart neoray. Underscores are treated as
spaces. If you think you tried every possibilities but neoray still can't find
the font, please report to me.
```vim
set guifont=Consolas:h11
set guifont=Ubuntu\ Mono:h12
set guifont=:h13 " Use default font with 13 pt size
```
NOTE: For now neoray doesn't support ttc fonts.
___
### exaple init.vim
```vim
if exists('g:neoray')
    set guifont=Go_Mono:h11
    let neoray_cursor_animation_time=0.07
    let neoray_background_transparency=0.95
    let neoray_target_ticks_per_second=120
    let neoray_context_menu_enabled=1
    let neoray_window_startup_state='centered'
    let neoray_window_startup_size='120x40'
    let neoray_key_toggle_fullscreen='<M-C-CR>' " AltGr+Enter
    let neoray_key_increase_fontsize='<C-PageUp>'
    let neoray_key_decrease_fontsize='<C-PageDown>'
endif
```

You can disable all of these features.
```vim
if exists('g:neoray')
    let neoray_cursor_animation_time=0
    let neoray_context_menu_enabled=0
    let neoray_window_startup_state=''
    let neoray_key_toggle_fullscreen=''
    let neoray_key_increase_fontsize=''
    let neoray_key_decrease_fontsize=''
endif
```
___
### flags
Neoray accepts some flags for startup options and more.
Other flags are used for creating neovim. You can look all
of them starting neoray with -h option.

Some of them are very important (at least for me)

#### --single-instance, -si
When this option has given, neoray opens only one instance.
Others will send all flags to already open instance and
immediately quits. This is usefull for game engine like
programs that you can use neovim as an external editor.
For examle if you are using godot engine you can set external
editor exec path to neoray executable and exec flags to this:
```
-si --file {file} --line {line} --column {col}
```
Now, everytime you open a script in godot, this will open in the
same neoray, and cursor goes to {line} and {col}
___
### contributing
All types of contributing are apreciated. If you want to be a part of this
project you can open issue when you find something not working, or help
development by solving issues and implementing some features what you want.
___
### development
The source code is well documented enough. I try to make everything
understandable. Neoray has no external dependencies (except linux). You need to
clone this repository and perform a go get command. Everything will be
installed and you will ready to fly.
___
### copyright
Neoray is licensed under MIT license. You can use, change, distribute
it however you want.
