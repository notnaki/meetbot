# MeetBot Go

A Go-based bot that can automatically join Google Meet sessions, enable microphone, and provide text-to-speech functionality through a web interface.

## Features

- **Automated Google Meet joining**: Join meetings via URL
- **Microphone control**: Enable/disable microphone programmatically
- **Text-to-Speech**: Generate and play audio through virtual microphone
- **Web interface**: Control the bot through a simple web UI
- **Screenshot capability**: Take screenshots of the current meeting
- **Docker support**: Containerized deployment with all dependencies

## Prerequisites

### Local Development
- Go 1.24.4 or later
- Playwright browsers
- PulseAudio (Linux/macOS)
- espeak-ng and sox for TTS
- X11 display server (for headless browser)

### Docker (Recommended)
- Docker
- Docker Compose (optional)

## Quick Start with Docker

1. **Clone and setup environment**:
   ```bash
   git clone <repository-url>
   cd meetbot-go-2
   cp .env.example .env  # Create from template if available
   ```

2. **Configure credentials**:
   Edit `.env` file with your Google account credentials:
   ```
   GOOGLE_EMAIL=your-email@gmail.com
   GOOGLE_PASSWORD=your-password
   ```

3. **Build and run**:
   ```bash
   ./build.sh
   ```

4. **Access the web interface**:
   Open http://localhost:8080 in your browser

## Local Development Setup

1. **Install Go dependencies**:
   ```bash
   go mod download
   ```

2. **Install system dependencies** (Ubuntu/Debian):
   ```bash
   sudo apt update
   sudo apt install -y \
     pulseaudio \
     espeak-ng \
     sox \
     xvfb \
     chromium-browser
   ```

3. **Install Playwright browsers**:
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright install
   ```

4. **Setup virtual microphone**:
   ```bash
   ./setup.sh
   ```

5. **Run the application**:
   ```bash
   go run main.go
   ```

## Usage

### Web Interface

1. **Initialize Bot**: Click "Initialize Bot" to start the browser session
2. **Join Meeting**: Enter a Google Meet URL and click "Join Meeting"
3. **Enable Microphone**: Click "Enable Microphone" to unmute
4. **Text-to-Speech**: Enter text and click "Generate TTS" to speak
5. **Take Screenshot**: Click "Screenshot" to capture current view
6. **Leave Meeting**: Click "Leave Meeting" to exit gracefully

### API Endpoints

- `GET /` - Web interface
- `POST /init-bot` - Initialize the bot
- `POST /join-meeting` - Join a meeting (requires `meetUrl` parameter)
- `POST /leave-meeting` - Leave current meeting
- `POST /enable-microphone` - Enable microphone
- `POST /generate` - Generate TTS (requires `text` parameter)
- `GET /screenshot` - Take screenshot
- `GET /bot-status` - Check bot initialization status
- `POST /clear-popups` - Clear browser popups

## Configuration

### Environment Variables

Create a `.env` file with:

```env
# Google Login Credentials
GOOGLE_EMAIL=your-email@gmail.com
GOOGLE_PASSWORD=your-app-password

# Optional: Browser settings
HEADLESS=false
DISPLAY=:99
```

### Audio Setup

The bot uses a virtual microphone to inject TTS audio into Google Meet:

1. **PulseAudio Configuration**: Automatically configured by `setup.sh`
2. **Virtual Microphone**: Creates `/tmp/virtmic` FIFO pipe
3. **TTS Pipeline**: espeak-ng → sox → virtual microphone

## Docker Details

### Build Process

The build uses a two-stage approach:
1. **Base image** (`Dockerfile.base`): Contains Playwright and system dependencies
2. **App image** (`Dockerfile`): Contains the Go application

### Container Features

- **Headless browser**: Runs Chromium in Xvfb
- **Audio support**: PulseAudio with virtual microphone
- **Port mapping**: Exposes port 8080
- **Shared memory**: 2GB for browser stability

### Development Commands

```bash
# Build and run
./build.sh

# Development build (rebuilds base image)
./build-dev.sh

# View logs
docker logs -f meetbot-go-container

# Debug container
docker exec -it meetbot-go-container /bin/bash

# Stop container
docker stop meetbot-go-container
```

## Troubleshooting

### Common Issues

1. **Login failures**: 
   - Use app-specific passwords for Google accounts with 2FA
   - Ensure credentials are correct in `.env`

2. **Audio not working**:
   - Check PulseAudio is running: `pactl info`
   - Verify virtual microphone: `pactl list sources short`

3. **Browser crashes**:
   - Increase shared memory: `--shm-size=2gb`
   - Check display server: `echo $DISPLAY`

4. **Permission errors**:
   - Clear browser popups: Use `/clear-popups` endpoint
   - Check microphone permissions in browser

### Logs and Debugging

- **Application logs**: Check console output or Docker logs
- **Browser debugging**: Screenshots available via `/screenshot`
- **Audio debugging**: Check PulseAudio with `pactl list`

## Security Notes

- **Credentials**: Never commit `.env` file to version control
- **Network**: Bot requires internet access for Google Meet
- **Permissions**: Requires microphone and camera permissions
- **Container**: Runs with necessary privileges for audio/video

## Development

### Project Structure

```
├── main.go              # HTTP server and main application
├── bot/                 # Bot implementation
│   └── bot.go          # Playwright automation logic
├── index.html          # Web interface
├── setup.sh            # Audio and display setup
├── keepalive.sh        # Process monitoring
├── Dockerfile          # Application container
├── Dockerfile.base     # Base image with dependencies
├── build.sh            # Build and run script
└── build-dev.sh        # Development build script
```

### Adding Features

1. **New endpoints**: Add handlers in `main.go`
2. **Bot actions**: Extend `bot/bot.go` with new methods
3. **UI updates**: Modify `index.html` for new controls
4. **Audio features**: Update TTS pipeline in `generateAndSendTTS()`

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]