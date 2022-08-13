if exists('g:neoray')
    set guifont=:h13
    set guicursor+=a:blinkwait1000-blinkon500-blinkoff250-Cursor
    NeoraySet CursorAnimTime  0.08
    NeoraySet Transparency    0.975
    NeoraySet TargetTPS       90
    NeoraySet ContextMenu     TRUE
    " NeoraySet ContextMenuItem ------------ :
    " NeoraySet ContextMenuItem Say\ Hello   :echo\ "Hello\ World"
    " NeoraySet ContextMenuItem Toggle\ Nerd :NERDTreeToggle
    NeoraySet BoxDrawing      TRUE
    NeoraySet ImageViewer     TRUE
    NeoraySet WindowSize      100x40
    NeoraySet WindowState     centered
    " NeoraySet KeyFullscreen <M-C-CR>
    " NeoraySet KeyZoomIn     <C-ScrollWheelUp>
    " NeoraySet KeyZoomOut    <C-ScrollWheelDown>
endif
