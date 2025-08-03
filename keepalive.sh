#!/bin/bash

PIPE=/tmp/virtmic

# Generate 0.1s of silence at 48kHz stereo
generate_silence() {
  sox -n -r 48000 -c 2 -b 16 -e signed-integer -t raw - synth 0.1 sine 0
}

while true; do
  # Check if pipe exists and is writable
  if [[ -p "$PIPE" ]]; then
    generate_silence > "$PIPE"
    sleep 0.1
  else
    echo "Pipe not found or closed."
    sleep 1
  fi
done
