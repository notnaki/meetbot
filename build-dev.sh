#!/bin/bash
set -e

echo "Development build - using existing base image..."

# Build only the application layer (super fast)
docker build -t meetbot-go .

# Stop and remove existing container if it exists
echo "Stopping existing container if running..."
docker stop meetbot-go-container 2>/dev/null || true
docker rm meetbot-go-container 2>/dev/null || true

# Run the container
echo "Starting container..."
docker run -d \
  --name meetbot-go-container \
  -p 8080:8080 \
  --shm-size=2gb \
  -e DISPLAY=:99 \
  -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
  meetbot-go

echo "Development container started!"
echo "Application is available at http://localhost:8080"
echo "Build time was much faster because we reused the base image!"

echo "Starting logs"
docker logs -f meetbot-go-container