#!/bin/bash
set -e

# Function to check if base image exists
check_base_image() {
    docker images meetbot-go-base:latest --format "table {{.Repository}}:{{.Tag}}" | grep -q "meetbot-go-base:latest"
}

# Build base image if it doesn't exist
if ! check_base_image; then
    echo "Base image not found. Building base image with Playwright (this may take a while)..."
    docker build -f Dockerfile.base -t meetbot-go-base:latest .
    echo "Base image built successfully!"
else
    echo "Base image found. Skipping Playwright installation."
fi

# Build the application image (this should be fast now)
echo "Building application image..."
docker build -t meetbot-go .

# Stop and remove existing container if it exists
echo "Stopping existing container if running..."
docker stop meetbot-go-container 2>/dev/null || true
docker rm meetbot-go-container 2>/dev/null || true

# Run the container with port mapping and additional options for GUI apps
echo "Starting container..."
docker run -d \
  --name meetbot-go-container \
  -p 8080:8080 \
  --shm-size=2gb \
  -e DISPLAY=:99 \
  -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
  meetbot-go

echo "Container started successfully!"
echo "Application is available at http://localhost:8080"
echo ""
echo "To view logs: docker logs -f meetbot-go-container"
echo "To stop: docker stop meetbot-go-container"
echo "To debug: docker exec -it meetbot-go-container /bin/bash"
echo ""
echo "To rebuild base image (if needed): docker build -f Dockerfile.base -t meetbot-go-base:latest ."
echo "To clean up base image: docker rmi meetbot-go-base:latest"