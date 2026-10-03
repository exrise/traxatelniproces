@echo off
rem Сборка SvinoVoyna.exe на Windows (нужен установленный Go 1.24+).
if not exist dist mkdir dist
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-H=windowsgui -s -w" -o dist\SvinoVoyna.exe .\cmd\svinovoyna
if errorlevel 1 exit /b 1
copy /Y README.md dist\README.md >nul
echo Готово: dist\SvinoVoyna.exe
