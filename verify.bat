@echo off
REM Quick verification script for Build Accelerator installation

echo Verifying Build Accelerator Installation
echo ==========================================
echo.

REM Check environment variables
echo [1/5] Checking environment variables...
if not defined BUILDCACHE_HOME (
    echo   ERROR: BUILDCACHE_HOME not set
    goto :error
) else (
    echo   OK: BUILDCACHE_HOME=%BUILDCACHE_HOME%
)

if not defined BUILDCACHE_PORT (
    echo   ERROR: BUILDCACHE_PORT not set
    goto :error
) else (
    echo   OK: BUILDCACHE_PORT=%BUILDCACHE_PORT%
)

REM Check Node.js installation
echo.
echo [2/5] Checking Node.js installation...
node --version >nul 2>&1
if errorlevel 1 (
    echo   ERROR: Node.js not found in PATH
    goto :error
) else (
    echo   OK: Node.js installed
)

REM Check dashboard dependencies
echo.
echo [3/5] Checking dashboard dependencies...
if not exist "dashboard\node_modules\" (
    echo   ERROR: Dashboard dependencies not installed
    echo   Run: cd dashboard ^&^& npm install
    goto :error
) else (
    echo   OK: Dashboard dependencies present
)

REM Check configuration file
echo.
echo [4/5] Checking configuration...
if not exist "config\buildcache.ini" (
    echo   WARNING: Configuration file not found
    echo   This may indicate setup was not completed
) else (
    echo   OK: Configuration file present
)

REM Check dashboard availability
echo.
echo [5/5] Checking dashboard server...
curl -s http://localhost:%BUILDCACHE_PORT%/api/metrics >nul 2>&1
if errorlevel 1 (
    echo   WARNING: Dashboard not responding
    echo   Start it with: cd dashboard ^&^& node server.js
) else (
    echo   OK: Dashboard running
)

echo.
echo ==========================================
echo Installation verification PASSED
echo Dashboard: http://localhost:%BUILDCACHE_PORT%/metrics
echo.
exit /b 0

:error
echo.
echo ==========================================
echo Installation verification FAILED
echo See README.md for setup instructions
echo.
exit /b 1
