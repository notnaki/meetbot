#!/bin/bash
set -e


echo "Generating PulseAudio system mode config..."

cat > /tmp/system.pa <<EOF
load-module module-native-protocol-unix socket=/tmp/pulse-socket auth-anonymous=1
load-module module-alsa-sink
load-module module-alsa-source device=hw:0,0
EOF

echo "Starting PulseAudio system mode..."
pulseaudio --system --disallow-exit --disable-shm --exit-idle-time=-1 --file=/tmp/system.pa &

sleep 3

echo "Setting PULSE_SERVER environment variable"
export PULSE_SERVER=unix:/tmp/pulse-socket

if [ -p /tmp/virtmic ]; then
    echo "FIFO /tmp/virtmic already exists, removing it..."
    rm /tmp/virtmic
fi

echo "Loading virtual mic module..."
pactl load-module module-pipe-source source_name=virtmic file=/tmp/virtmic format=s16le rate=48000 channels=2

echo "Setting virtmic as the default source..."
pactl set-default-source virtmic

echo "Listing PulseAudio sources..."
pactl list sources short

echo "Checking default source..."
pactl info | grep "Default Source"

echo "Virtual mic module loaded and configured."

echo "Starting Xvfb for headless browser support..."
Xvfb :99 -screen 0 1024x768x24 &
export DISPLAY=:99

echo "Disabling D-Bus for Chrome in container..."
export DBUS_SESSION_BUS_ADDRESS=""
unset DBUS_SESSION_BUS_ADDRESS

echo "Setting up Playwright environment..."
export PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

# Verify Playwright browsers are available
echo "Checking Playwright browser installation..."
if [ -d "/ms-playwright" ]; then
    echo "Playwright browsers directory found:"
    ls -la /ms-playwright/
else
    echo "Playwright browsers directory not found, attempting to install..."
    go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium
fi

# Find chromium executable
CHROMIUM_PATH=$(find /ms-playwright -name "chrome" -type f 2>/dev/null | head -1)
if [ -n "$CHROMIUM_PATH" ]; then
    echo "Found Chromium at: $CHROMIUM_PATH"
else
    echo "Warning: Chromium executable not found"
    find /ms-playwright -name "*chrome*" -type f 2>/dev/null || echo "No chrome files found"
fi


echo "Running main application..."
exec "$@"
