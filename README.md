# Neoray

Neoray is a simple and lightweight gui client for neovim.
It's written in golang using glfw and opengl bindings.
Neoray is easy to use and binary size is small. Supports
most of the neovim features. Uses small amount of ram and
leaves no footprints on your computer.

Neoray doesn't need any additional configuration, but you
can customize it however you want in your init.vim.

These are some options you can specify for now:

- neoray_cursor_animation_time (float)
    The cursor moving smoothly in neoray and you can specify
    how long it's move takes. Default is 0.08 (1.0 is one second)
    You can disable it by setting to 0.

- neoray_framebuffer_transparency (float)
    Transparency of the window background. Default is 1 means
    no transparency, and 0 is fully transparent. Only background
    colors will be transparent, and statusline, tabline and texts
    are fully opaque.

- neoray_target_ticks_per_second (int)
    The target update time in one second. Like FPS but neoray
    doesn't render screen in every frame. Default is 60.

- neoray_popup_menu_enabled (bool)
    Neoray has a simple right click menu that gives you some abilities
    like copying, cutting to system clipboard and pasting. It has a
    open file functionality that opens system file dialog. Menu text
    is same as the font and the colors are from your colorscheme. This
    makes it look and feel like terminal. You can disable it by setting
    this option to 0. Default is 1 which means enabled.

- neoray_window_startup_state (string)
    You can specify how the neoray window will be shown. The possible
    values are "minimized", "maximized", "fullscreen", "centered".
    Default is none.

Neoray uses some key combinations for switching between fullscreen and
windowed mode, zoom in and out eg. You can set these keys and also
disable as you wish. All options here are strings contains vim style
keybindings.

- neoray_key_toggle_fullscreen (default <F11>)
- neoray_key_increase_fontsize (default <C-+>)
- neoray_key_decrease_fontsize (default <C-->)

#### guifont
Neoray respects your guifont option, finds the font and loads it.
But it hasn't got platform specific font enumerating. You can load
known fonts as its family name like 'Consolas', but for other fonts
you need to specify font file name. Examples:

set guifont=Consolas:h11
set guifont=Ubuntu\ Mono:h12

Also you can write underscore instead of escaping space. eg: Ubuntu_Mono

#### exaple init.vim
```
if exists('g:neoray')
    set guifont=Go_Mono:h11
    let neoray_cursor_animation_time=0.07
    let neoray_framebuffer_transparency=0.95
    let neoray_target_ticks_per_second=120
    let neoray_popup_menu_enabled=1
    let neoray_window_startup_state='centered'
    let neoray_key_toggle_fullscreen='<M-C-CR>'
    let neoray_key_increase_fontsize='<C-PageUp>'
    let neoray_key_decrease_fontsize='<C-PageDown>'
endif
```

You can disable all this features.
```
if exists('g:neoray')
    let neoray_cursor_animation_time=0
    let neoray_popup_menu_enabled=0
    let neoray_window_startup_state=''
    let neoray_key_toggle_fullscreen=''
    let neoray_key_increase_fontsize=''
    let neoray_key_decrease_fontsize=''
endif
```

#### flags
Neoray has taken some of the flags has given on startup.
Other flags are used for creating neovim. You may look all
of them as starting neoray with -h option.

Some of them are very important (at least for me)

--single-instance, -si
    When this option has given, neoray opens only one instance.
    Others will send all flags to already open instance and
    immediately quits. This is usefull for game engine like
    programs that you can use neovim as an external editor.
    The godot engine example is here:

    Set the external editor exec path to neoray executable and
    exec flags to this: -si --file {file} --line {line} --column {col}

    Now, everytime you open a script in godot, this will open in the
    same neoray, and cursor goes to {line} and {column}

#### contributing
All types of contributing as welcomed. If you want to be a part of this
project you can open issue when you find something not working, or help
development by solving issues and implementing some features what you want,
and also you can buy me a coffee.

#### development
The source code is well documented enough. I try to make everything
understandable. Neoray has no external dependencies. You need to clone
this repository and perform a go get command. Everything will be installed
and you will ready to fly.

#### copyright
Neoray is licensed under MIT license. You can use, change, distribute
it however you want. It ships with Cascadia Mono as default font (my best)
and Cascadia Mono is powered by Microsoft Corporation, licensed under SFL v1.1
