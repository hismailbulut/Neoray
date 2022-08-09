# This is template for Neoray runtime files
function s:NeorayOptionSet(...)
	if a:0 < 2
		echoerr 'NeoraySet needs at least 2 arguments'
		return
	endif
	call call(function("rpcnotify"), [$(CHANID), "NeorayOptionSet"] + a:000)
endfunction

# TODO: The completion must respect user input
function s:NeorayCompletion(A, L, P)
	return 
	\	[
	\	'CursorAnimTime',
	\	'Transparency',
	\	'TargetTPS',
	\	'ContextMenu',
	\	'ContextButton',
	\	'BoxDrawing',
	\	'ImageViewer',
	\	'WindowState',
	\	'WindowSize',
	\	'KeyFullscreen',
	\	'KeyZoomIn',
	\	'KeyZoomOut' 
	\	]
endfunction

command -nargs=+ -complete=customlist,s:NeorayCompletion NeoraySet call s:NeorayOptionSet(<f-args>)

# Delete buffer but keep window layout
function s:NeorayDeleteBuffer()
    let l:currentBufNum = bufnr("%")
    let l:alternateBufNum = bufnr("#")
    if buflisted(l:alternateBufNum)
        buffer #
    else
        bnext
    endif
    if bufnr("%") == l:currentBufNum
      new
    endif
    if buflisted(l:currentBufNum)
      execute("bwipeout! ".l:currentBufNum)
    endif
endfunction

augroup Neoray
	autocmd VimEnter * call rpcnotify($(CHANID), 'NeorayVimEnter')
	autocmd VimLeave * call rpcnotify($(CHANID), 'NeorayVimLeave')
	autocmd BufReadPre *.png,*.jpg,*.jpeg,*.gif,*.webp,*.bmp let s:imageViewed = rpcrequest($(CHANID), "NeorayViewImage", expand("%:p"))
	autocmd BufReadPost *.png,*.jpg,*.jpeg,*.gif,*.webp,*.bmp if s:imageViewed == 1 | call s:NeorayDeleteBuffer() | endif
augroup end
