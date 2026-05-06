@echo off
setlocal

set "PRODUCTION=true"
wails3 build %*

endlocal
