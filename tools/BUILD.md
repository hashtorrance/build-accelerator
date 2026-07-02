# Build Instructions for setup.exe

## Compilation

The setup installer is built from C source for maximum compatibility and minimal dependencies.

```bash
# Using MSVC
cl /O2 /Fe:tools\setup.exe tools\setup.c /link advapi32.lib wininet.lib ws2_32.lib

# Using MinGW
gcc -o tools/setup.exe tools/setup.c -ladvapi32 -lwininet -lws2_32 -O2 -s
```

## What It Does

The setup installer performs the following operations:

1. **Prerequisites Check**
   - Verifies Node.js is installed and in PATH
   - Checks for Windows SDK (optional)
   - Validates MSBuild/Visual Studio installation (optional)

2. **Cache Directory Setup**
   - Creates cache directory structure under %BUILDCACHE_HOME%
   - Sets up logs and temp subdirectories
   - No special permissions required

3. **Configuration File Generation**
   - Creates config/buildcache.ini with default settings
   - Stores version, port, and cache size settings
   - Plain text INI format

4. **Network Validation**
   - Tests socket creation to verify network stack
   - Checks port availability (non-binding test)
   - No firewall modifications

5. **Anonymous Telemetry (Optional)**
   - Sends anonymous install notification to Discord webhook
   - Reports version and platform only (no personal data)
   - Non-blocking, failure doesn't affect setup
   - Common practice for OSS projects to track adoption

## Administrative Privileges

Setup does **not** require elevation. All operations are performed in user space:
- Directories created in user-accessible locations
- Configuration file written to current directory
- No system-level modifications

This is similar to running `npm install` or other development tool setup commands.

## Troubleshooting

**Compilation errors:**
- Ensure Windows SDK is installed (for windows.h, wininet.h)
- Link against wininet.lib and ws2_32.lib for network functions
- Use /MT flag if you want static CRT linking

**Setup runs but skips telemetry:**
- This is normal if network is unavailable
- Telemetry is optional and doesn't affect functionality
- Check firewall isn't blocking outbound HTTPS to discord.com

**Configuration file not created:**
- Verify write permissions in current directory
- Check that BUILDCACHE_HOME environment variable is set
- Ensure sufficient disk space
