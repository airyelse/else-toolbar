@echo off
setlocal

set "ROOT=%~dp0"
set "PROJECT_ROOT=%ROOT%..\.."
set "OUTDIR=%PROJECT_ROOT%\build\bin"
set "CSC=C:\Windows\Microsoft.NET\Framework64\v4.0.30319\csc.exe"
set "FRAMEWORK=C:\Program Files (x86)\Reference Assemblies\Microsoft\Framework\.NETFramework\v4.7.1"
set "WINRT=C:\WINDOWS\Microsoft.Net\assembly\GAC_MSIL\System.Runtime.WindowsRuntime\v4.0_4.0.0.0__b77a5c561934e089\System.Runtime.WindowsRuntime.dll"

if not exist "%OUTDIR%" mkdir "%OUTDIR%"

"%CSC%" /nologo /t:exe /out:"%OUTDIR%\windowshellolink.exe" /r:"%FRAMEWORK%\Facades\System.Runtime.dll" /r:"%WINRT%" /r:"C:\Windows\System32\WinMetadata\Windows.Foundation.winmd" /r:"C:\Windows\System32\WinMetadata\Windows.Security.winmd" "%ROOT%Program.cs" "%ROOT%AsyncBridge.cs" "%ROOT%NativeMethods.cs"

exit /b %errorlevel%
