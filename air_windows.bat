@echo off
echo Building all services...

REM Build gateway
go build -o .\tmp\gateway.exe .\services\gateway
if %errorlevel% neq 0 (
    echo Failed to build gateway
    exit /b %errorlevel%
)

REM Build analyzer
go build -o .\tmp\analyzer.exe .\services\analyzer
if %errorlevel% neq 0 (
    echo Failed to build analyzer
    exit /b %errorlevel%
)

REM Build link-checker
go build -o .\tmp\link-checker.exe .\services\link-checker
if %errorlevel% neq 0 (
    echo Failed to build link-checker
    exit /b %errorlevel%
)

echo Starting gateway service...
.\tmp\gateway.exe