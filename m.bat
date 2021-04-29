@echo off
set SOURCEFILES=src\main.go src\window.go src\nvim_process.go src\event_handler.go
set SOURCEFILES=%SOURCEFILES% src\grid_table.go src\utils.go src\input.go src\cursor.go
set SOURCEFILES=%SOURCEFILES% src\canvas.go src\font.go
set LDFLAGS=
set GCFLAGS="-N -l"
set EXECUTABLENAME=bin\neoray.exe
set BUILDSUCCESS=0
echo BUILDING %EXECUTABLENAME%
go build -o %EXECUTABLENAME% -ldflags=%LDFLAGS% -gcflags=all=%GCFLAGS% %SOURCEFILES%
if %ERRORLEVEL% gtr 0 (
    echo BUILD FAILED Error Level: %ERRORLEVEL%
) else (
    if exist %EXECUTABLENAME% (
        echo FINISHED
        set BUILDSUCCESS=1
    ) else (
        echo BUILD FAILED Executable Not Found
    )
)
