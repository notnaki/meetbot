# Use pre-built base image with Playwright (build once, reuse many times)
FROM meetbot-go-base:latest

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o app .

# Expose PulseAudio config and runtime dir
ENV PULSE_SERVER=unix:/tmp/pulse-socket

# Chrome/Chromium environment variables for Docker
ENV DISPLAY=:99
ENV CHROME_BIN=/ms-playwright/chromium-1169/chrome-linux/chrome
ENV CHROMIUM_FLAGS="--no-sandbox --disable-dev-shm-usage"

# Disable D-Bus for Chrome in container
ENV DBUS_SESSION_BUS_ADDRESS=""

# PulseAudio runtime
RUN mkdir -p /home/appuser/.config/pulse /tmp/pulse
RUN echo "default-sample-format = s16le" > /home/appuser/.config/pulse/daemon.conf && \
    echo "default-sample-rate = 16000" >> /home/appuser/.config/pulse/daemon.conf

# Make setup script executable
RUN chmod +x setup.sh keepalive.sh

# Expose port for web interface
EXPOSE 8080

# Entry point
CMD ["./setup.sh", "./app"]
