package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"meetbot-go-2/bot"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

var (
	globalBot *bot.Bot
	botMutex  sync.Mutex
)

func openPipeNonBlocking(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK, 0644)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func generateAndSendTTS(text string) error {
	wavFile := "tts.wav"
	pipePath := "/tmp/virtmic"

	// Generate WAV file using espeak-ng
	convertedFile := "tts_48k_stereo.wav"
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		`espeak-ng -s 65 "%s" --stdout | sox -t wav - -r 48000 -c 2 -b 16 %s`, text, convertedFile,
	))
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate and convert wav: %v", err)
	}

	fmt.Println("Generated Wav file:", wavFile)

	wav, err := os.Open(convertedFile)
	if err != nil {
		return fmt.Errorf("failed to open wav file: %v", err)
	}
	defer wav.Close()

	// Skip WAV header (44 bytes)
	_, err = wav.Seek(44, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek wav file: %v", err)
	}

	if _, err := os.Stat(pipePath); os.IsNotExist(err) {
		return fmt.Errorf("pipe does not exist: %s", pipePath)
	}

	fmt.Println("About to open pipe for writing...")

	pipe, err := openPipeNonBlocking(pipePath)
	if err != nil {
		if err == unix.ENXIO {
			// No reader on the other end of the pipe
			return fmt.Errorf("no reader available on the pipe")
		}
		return fmt.Errorf("failed to open pipe: %v", err)
	}

	defer pipe.Close()

	fmt.Println("Pipe opened successfully!")
	fmt.Println("Sending audio data in real-time chunks...")

	// Send audio in small chunks with timing to simulate real-time playback
	buf := make([]byte, 8192) // Small chunks
	for {
		n, err := wav.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read wav file: %v", err)
		}

		_, err = pipe.Write(buf[:n])
		if err != nil {
			return fmt.Errorf("failed to write to pipe: %v", err)
		}
	}

	fmt.Println("Audio playback complete")

	// Original loop functionality
	i := 0
	for i < 10 {
		fmt.Printf("Loop iteration: %d\n", i)
		i++
	}

	fmt.Println("Wav file sent to pipe:", pipePath)

	// Keep pipe open a bit longer
	time.Sleep(5 * time.Second)

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, nil)
}

func joinMeetingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetUrl := r.FormValue("meetUrl")
	if meetUrl == "" {
		http.Error(w, "meetUrl parameter is required", http.StatusBadRequest)
		return
	}

	fmt.Printf("Processing join meeting request for URL: %s\n", meetUrl)

	botMutex.Lock()
	defer botMutex.Unlock()

	// Initialize bot if not already done
	if globalBot == nil {
		var err error
		globalBot, err = bot.NewBot(false) // false = not headless, show browser
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create bot: %v", err), http.StatusInternalServerError)
			return
		}

		err = globalBot.Initialize()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to initialize bot: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Check if already logged in
	loggedIn, err := globalBot.IsLoggedIn()
	if err != nil {
		fmt.Printf("Error checking login status: %v\n", err)
	}

	if !loggedIn {
		fmt.Println("Not logged in, performing login...")
		err = globalBot.GoogleLogin()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to login: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		fmt.Println("Already logged in, skipping login...")
	}

	// Join the meeting
	err = globalBot.JoinGoogleMeet(meetUrl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to join meeting: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully joined the meeting"))
}

func leaveMeetingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botMutex.Lock()
	defer botMutex.Unlock()

	if globalBot == nil {
		http.Error(w, "No active bot session", http.StatusBadRequest)
		return
	}

	// Leave the meeting gracefully
	err := globalBot.LeaveMeeting()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to leave meeting: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully left the meeting"))
}

func enableMicrophoneHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botMutex.Lock()
	defer botMutex.Unlock()

	if globalBot == nil {
		http.Error(w, "No active bot session", http.StatusBadRequest)
		return
	}

	fmt.Println("Processing enable microphone request...")

	err := globalBot.EnableMicrophone()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enable microphone: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Microphone enabled successfully"))
}

func initBotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	out, err := exec.Command("pgrep", "-f", "keepalive.sh").Output()
	if err == nil && len(out) > 0 {
		fmt.Println("keepalive.sh already running")
	} else {
		// Start keepalive.sh if not running
		cmd := exec.Command("/bin/bash", "./keepalive.sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start keepalive script: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Started keepalive.sh with PID %d\n", cmd.Process.Pid)
	}

	fmt.Println("Processing bot initialization request...")

	botMutex.Lock()
	defer botMutex.Unlock()

	// Initialize bot if not already done
	if globalBot == nil {
		var err error
		globalBot, err = bot.NewBot(false) // false = not headless, show browser
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create bot: %v", err), http.StatusInternalServerError)
			return
		}

		err = globalBot.Initialize()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to initialize bot: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Bot initialized successfully"))
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	text := r.FormValue("text")
	err := generateAndSendTTS(text)
	if err != nil {
		fmt.Println("Error:", err)
		http.Error(w, "Failed to generate and send TTS", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "TTS generated and sent successfully!")
}

func screenshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botMutex.Lock()
	defer botMutex.Unlock()

	if globalBot == nil {
		http.Error(w, "No active bot session", http.StatusBadRequest)
		return
	}

	fmt.Println("Processing screenshot request...")

	screenshot, err := globalBot.TakeScreenshot()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to take screenshot: %v", err), http.StatusInternalServerError)
		return
	}

	// Set appropriate headers for image response
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(screenshot)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.WriteHeader(http.StatusOK)
	w.Write(screenshot)
}

func botStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botMutex.Lock()
	defer botMutex.Unlock()

	isInitialized := globalBot != nil

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"initialized": %t}`, isInitialized)))
}

func clearPopupsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botMutex.Lock()
	defer botMutex.Unlock()

	if globalBot == nil {
		http.Error(w, "No active bot session", http.StatusBadRequest)
		return
	}

	fmt.Println("Processing clear popups request...")

	globalBot.ClearPopups()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Popups cleared successfully"))
}

func main() {

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/join-meeting", joinMeetingHandler)
	http.HandleFunc("/leave-meeting", leaveMeetingHandler)
	http.HandleFunc("/enable-microphone", enableMicrophoneHandler)
	http.HandleFunc("/init-bot", initBotHandler)
	http.HandleFunc("/bot-status", botStatusHandler)
	http.HandleFunc("/screenshot", screenshotHandler)
	http.HandleFunc("/clear-popups", clearPopupsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
