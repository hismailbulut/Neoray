@echo off
call m.bat
if %BUILDSUCCESS% equ 1 (
    echo Running %EXECUTABLENAME%
    echo.
    .\%EXECUTABLENAME%
)
