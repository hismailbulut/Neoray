@echo off
call m.bat
if %BUILDSUCCESS% equ 1 (
    echo Debugging %EXECUTABLENAME%
    echo.
    dlv -race exec %EXECUTABLENAME%
)
