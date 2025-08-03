package bot

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Bot struct {
	// Playwright instances
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page

	// Configuration
	headless bool
	email    string
	password string

	// State
	running bool
}

func (b *Bot) findElementFast(selectors []string, timeout int) (string, error) {
	for _, selector := range selectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(float64(timeout)),
		})
		if err == nil {
			return selector, nil
		}
	}
	return "", fmt.Errorf("none of the selectors found an element within %dms", timeout)
}

func (b *Bot) ClearPopups() {
	log.Printf("[POPUP_CLEARING] Starting popup clearing process...")

	// Common popup selectors that can appear in Google Meet
	popupSelectors := []string{
		// Permission popups
		"button:has-text('Allow')",
		"button:has-text('Block')",
		"div[role='button']:has-text('Allow')",
		"div[role='button']:has-text('Block')",

		// Notification popups
		"button:has-text('Got it')",
		"button:has-text('Dismiss')",
		"button:has-text('OK')",
		"button:has-text('Close')",
		"button:has-text('Continue')",
		"button:has-text('Next')",
		"button:has-text('Skip')",
		"button:has-text('Not now')",
		"button:has-text('Maybe later')",

		// Generic close buttons
		"[aria-label='Close']",
		"[aria-label='Dismiss']",
		"button[data-dismiss]",
		".close-button",

		// Meet-specific popups
		"button:has-text('Use a phone for audio')",
		"button:has-text('Join and use a phone')",
		"button:has-text(\"Don't use a phone\")",
		"button:has-text('Use a phone')",

		// Browser notification related
		"button:has-text('Turn on')",
		"button:has-text('Turn off')",
		"div[role='button']:has-text('Turn on')",
		"div[role='button']:has-text('Turn off')",

		// Generic modal close buttons
		"button[aria-label*='close']",
		"button[aria-label*='dismiss']",
		"div[role='button'][aria-label*='close']",
		"div[role='button'][aria-label*='dismiss']",
	}

	// Try to dismiss each type of popup
	for _, selector := range popupSelectors {
		elements, err := b.page.Locator(selector).All()
		if err != nil {
			continue
		}

		for _, element := range elements {
			isVisible, err := element.IsVisible()
			if err != nil || !isVisible {
				continue
			}

			log.Printf("[POPUP_CLEARING] Dismissing popup with selector: %s", selector)
			err = element.Click(playwright.LocatorClickOptions{
				Timeout: playwright.Float(1000),
			})
			if err != nil {
				log.Printf("[POPUP_CLEARING] Failed to click popup: %v", err)
				continue
			}

			// Wait for popup to disappear
			time.Sleep(500 * time.Millisecond)
			log.Printf("[POPUP_CLEARING] Successfully dismissed popup")
		}
	}

	// Wait a moment for any remaining popups to settle
	time.Sleep(1000 * time.Millisecond)
	log.Printf("[POPUP_CLEARING] Popup clearing completed")
}

func (b *Bot) clickWithLogging(selector, action, context string) error {
	b.logButtonClick(action, selector, context)

	err := b.page.Locator(selector).Click()
	if err != nil {
		log.Printf("[BUTTON_CLICK_ERROR] Failed to click %s: %v", selector, err)
		return err
	}

	log.Printf("[BUTTON_CLICK_SUCCESS] Successfully clicked: %s", selector)
	return nil
}

func (b *Bot) logButtonClick(action, selector, context string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	log.Printf("[BUTTON_CLICK] %s | Action: %s | Selector: %s | Context: %s",
		timestamp, action, selector, context)
}

func (b *Bot) GoogleLogin() error {
	if !b.running {
		return fmt.Errorf("bot not initialized")
	}

	_, err := b.page.Goto("https://accounts.google.com/signin/v2/identifier?flowName=GlifWebSignIn&flowEntry=ServiceLogin", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})

	if err != nil {
		return fmt.Errorf("failed to navigate to google login page: %v", err)
	}

	// Wait for the email input field to be visible with multiple selectors
	emailSelectors := []string{
		"input#identifierId",
		"input[type='email']",
		"input[name='identifier']",
		"input[autocomplete='username']",
	}

	var emailInput string
	for _, selector := range emailSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(1000), // Reduced from 5000ms to 1000ms
		})
		if err == nil {
			emailInput = selector
			fmt.Printf("Found email input with selector: %s\n", selector)
			break
		}
	}

	if emailInput == "" {
		return fmt.Errorf("failed to find email input field")
	}

	err = b.page.Locator(emailInput).PressSequentially(b.email, playwright.LocatorPressSequentiallyOptions{
		Delay: playwright.Float(50),
	})

	if err != nil {
		return fmt.Errorf("failed to type email in: %v", err)
	}

	// Click the "Next" button after email input
	emailNextSelectors := []string{
		"div#identifierNext",
		"button#identifierNext",
		"input#identifierNext",
		"button:has-text('Next')",
		"div[role='button']:has-text('Next')",
	}

	var emailNextButton string
	for _, selector := range emailNextSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(1000), // Reduced from 3000ms to 1000ms
		})
		if err == nil {
			emailNextButton = selector
			fmt.Printf("Found email next button with selector: %s\n", selector)
			break
		}
	}

	if emailNextButton == "" {
		return fmt.Errorf("failed to find email next button")
	}

	err = b.clickWithLogging(emailNextButton, "CLICK_EMAIL_NEXT", "Google Login - Email Step")
	if err != nil {
		return fmt.Errorf("failed to click email next button: %v", err)
	}

	// Wait for page transition to password step
	fmt.Println("Waiting for password page to load...")

	// Add a small delay to ensure page transition
	err = b.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		fmt.Printf("Warning: Failed to wait for network idle: %v\n", err)
	}

	// Wait for password input field with most specific selectors first
	// Based on the error, we want: input[name="Passwd"] which is the visible password field
	passwordSelectors := []string{
		"input[name='Passwd']",                                             // Most specific - the actual password field
		"input[name='Passwd'][type='password']",                            // Even more specific
		"input[autocomplete='current-password']:not([aria-hidden='true'])", // Exclude hidden fields
		"input[jsname='YPqjbf']",                                           // Google's internal identifier
		"input[type='password'][tabindex='0']",                             // Only focusable password fields
		"input[type='password']:not([name='hiddenPassword'])",              // Explicitly exclude hidden password
	}

	var passwordInput string
	for i, selector := range passwordSelectors {
		fmt.Printf("Trying password selector %d: %s\n", i+1, selector)
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(800), // Reduced from 3000ms to 800ms
		})
		if err == nil {
			passwordInput = selector
			fmt.Printf("✓ Found password input with selector: %s\n", selector)

			// Quick check that this selector returns exactly one element
			count, countErr := b.page.Locator(selector).Count()
			if countErr == nil && count == 1 {
				fmt.Printf("✓ Selector matches exactly 1 element\n")
				break
			} else if count > 1 {
				fmt.Printf("⚠ Selector matches %d elements, trying next...\n", count)
				passwordInput = ""
				continue
			}
			break
		}
	}

	if passwordInput == "" {
		// Try the getByLabel approach as a last resort (mentioned in the error)
		fmt.Println("Trying getByLabel approach as fallback...")
		labelLocator := b.page.GetByLabel("Enter your password")
		err := labelLocator.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(1000), // Reduced from 3000ms to 1000ms
		})
		if err == nil {
			fmt.Println("✓ Found password field using getByLabel")
			err = labelLocator.PressSequentially(b.password, playwright.LocatorPressSequentiallyOptions{
				Delay: playwright.Float(50),
			})
			if err != nil {
				return fmt.Errorf("failed to type password using getByLabel: %v", err)
			}
		} else {
			return fmt.Errorf("failed to find password input field with any method")
		}
	} else {
		err = b.page.Locator(passwordInput).PressSequentially(b.password, playwright.LocatorPressSequentiallyOptions{
			Delay: playwright.Float(50),
		})
		if err != nil {
			return fmt.Errorf("failed to type password in: %v", err)
		}
	}

	// Try multiple selectors for the next button
	nextButtonSelectors := []string{
		"div#passwordNext",
		"button[type='submit']",
		"input[type='submit']",
		"button:has-text('Next')",
		"div[role='button']:has-text('Next')",
	}

	var nextButton string
	for _, selector := range nextButtonSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(2000),
		})
		if err == nil {
			nextButton = selector
			fmt.Printf("Found next button with selector: %s\n", selector)
			break
		}
	}

	if nextButton != "" {
		err = b.clickWithLogging(nextButton, "CLICK_PASSWORD_NEXT", "Google Login - Password Step")
		if err != nil {
			return fmt.Errorf("failed to click next button: %v", err)
		}
	} else {
		// Fallback: try pressing Enter
		fmt.Println("Next button not found, trying Enter key...")
		log.Printf("[KEYBOARD_ACTION] Pressing Enter key as fallback for password next")
		err = b.page.Keyboard().Press("Enter")
		if err != nil {
			return fmt.Errorf("failed to press Enter key: %v", err)
		}
	}

	fmt.Println("Completing login...")

	// Wait for navigation or success indicators instead of timeout
	// Try to wait for URL change or success elements
	loginSuccessful := false

	// First, try waiting for URL changes that indicate successful login
	err = b.page.WaitForURL("**/myaccount.google.com/**", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(3000), // Reduced from 10000ms to 3000ms
	})
	if err == nil {
		fmt.Println("Google login successful - redirected to myaccount")
		loginSuccessful = true
	}

	if !loginSuccessful {
		err = b.page.WaitForURL("**/accounts.google.com/signin/oauth/**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(2000), // Reduced from 5000ms to 2000ms
		})
		if err == nil {
			fmt.Println("Google login successful - OAuth redirect")
			loginSuccessful = true
		}
	}

	// If URL-based detection didn't work, try waiting for success indicators
	if !loginSuccessful {
		successSelectors := []string{
			"text=Welcome",
			"[data-email]",
		}

		for _, selector := range successSelectors {
			err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(5000),
			})
			if err == nil {
				fmt.Println("Google login successful - found success indicator")
				loginSuccessful = true
				break
			}
		}
	}

	if !loginSuccessful {
		// Check current URL for login success
		currentUrl := b.page.URL()
		fmt.Printf("Current URL after login attempt: %s\n", currentUrl)

		// Check if URL indicates successful login
		if currentUrl != "" &&
			(strings.Contains(currentUrl, "myaccount.google.com") ||
				strings.Contains(currentUrl, "accounts.google.com/signin/oauth") ||
				!strings.Contains(currentUrl, "signin/v2/identifier")) {
			fmt.Println("Google login appears successful based on URL")
		} else {
			// Check if there's an error message
			errorLocator := b.page.Locator("[jsname='B34EJ'] span")
			err := errorLocator.WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(2000),
			})
			if err == nil {
				errorText, _ := errorLocator.TextContent()
				fmt.Printf("Login error: %s\n", errorText)
				return fmt.Errorf("google login failed: %s", errorText)
			} else {
				fmt.Println("Login status unclear, continuing...")
			}
		}
	}

	return nil
}

func (b *Bot) Close() error {
	if b.browser != nil {
		if err := b.browser.Close(); err != nil {
			return err
		}
	}
	if b.pw != nil {
		if err := b.pw.Stop(); err != nil {
			return err
		}
	}
	b.running = false
	return nil
}

// JoinGoogleMeet joins a Google Meet meeting using the provided URL
func (b *Bot) JoinGoogleMeet(meetingURL string) error {
	if !b.running {
		return fmt.Errorf("bot not initialized")
	}

	fmt.Printf("Joining Google Meet: %s\n", meetingURL)

	// Navigate to the meeting URL
	_, err := b.page.Goto(meetingURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return fmt.Errorf("failed to navigate to meeting URL: %v", err)
	}

	// Wait for the page to load
	err = b.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		return fmt.Errorf("failed to wait for page load: %v", err)
	}

	// Check if we need to login first
	currentUrl := b.page.URL()
	if strings.Contains(currentUrl, "accounts.google.com") && strings.Contains(currentUrl, "signin") {
		fmt.Println("Need to login first...")
		err = b.GoogleLogin()
		if err != nil {
			return fmt.Errorf("failed to login before joining meeting: %v", err)
		}

		// Navigate back to meeting after login
		_, err = b.page.Goto(meetingURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		})
		if err != nil {
			return fmt.Errorf("failed to navigate back to meeting after login: %v", err)
		}
	}

	// Clear any popups that might interfere
	b.ClearPopups()

	// Handle microphone and camera permissions
	fmt.Println("Handling microphone and camera settings...")

	// Try to turn off camera
	cameraSelectors := []string{
		"div[data-tooltip*='camera']",
		"button[aria-label*='camera']",
		"div[aria-label*='Turn off camera']",
		"button[data-tooltip*='Turn off camera']",
	}

	cameraSelector, err := b.findElementFast(cameraSelectors, 1000)
	if err == nil {
		fmt.Printf("Found camera button with selector: %s\n", cameraSelector)
		b.clickWithLogging(cameraSelector, "TOGGLE_CAMERA_OFF", "Google Meet - Pre-join Setup")
	}

	// Try to turn off microphone initially (we'll control it via virtual mic)
	micSelectors := []string{
		"div[data-tooltip*='microphone']",
		"button[aria-label*='microphone']",
		"div[aria-label*='Turn off microphone']",
		"button[data-tooltip*='Turn off microphone']",
	}

	micSelector, err := b.findElementFast(micSelectors, 1000)
	if err == nil {
		fmt.Printf("Found microphone button with selector: %s\n", micSelector)
		b.clickWithLogging(micSelector, "TOGGLE_MIC_OFF", "Google Meet - Pre-join Setup")
	}

	// Clear popups again after handling camera/microphone
	b.ClearPopups()

	// Look for and click the "Join now" button with retry logic
	fmt.Println("Looking for join button...")
	joinSelectors := []string{
		"button:has-text('Join now')",
		"div[role='button']:has-text('Join now')",
		"button:has-text('Ask to join')",
		"div[role='button']:has-text('Ask to join')",
		"button[aria-label*='Join']",
		"div[data-tooltip*='Join']",
	}

	joinButtonFound := false
	maxRetries := 2

	for retry := 0; retry < maxRetries && !joinButtonFound; retry++ {
		if retry > 0 {
			fmt.Printf("Retry attempt %d for join button...\n", retry)
			b.ClearPopups() // Clear popups before retry
		}

		for _, selector := range joinSelectors {
			err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(1500),
			})
			if err == nil {
				fmt.Printf("Found join button with selector: %s\n", selector)
				err = b.clickWithLogging(selector, "JOIN_MEETING", "Google Meet - Join Meeting")
				if err != nil {
					log.Printf("[JOIN_BUTTON_ERROR] Failed to click join button on attempt %d: %v", retry+1, err)
					if retry < maxRetries-1 {
						continue // Try next selector or retry
					}
					return fmt.Errorf("failed to click join button after %d attempts: %v", maxRetries, err)
				}
				joinButtonFound = true
				break
			}
		}
	}

	if !joinButtonFound {
		return fmt.Errorf("could not find join button after %d attempts", maxRetries)
	}

	// Wait for meeting to load
	fmt.Println("Waiting for meeting to load...")

	// Wait for meeting interface elements
	meetingSelectors := []string{
		"div[data-allocation-index]", // Meeting participants area
		"div[jsname='HzV7m']",        // Meeting controls
		"button[aria-label*='Leave call']",
		"div[aria-label*='You joined']",
	}

	meetingJoined := false
	for _, selector := range meetingSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(3000), // Reduced from 15000ms to 3000ms
		})
		if err == nil {
			fmt.Println("Successfully joined the meeting!")
			meetingJoined = true
			break
		}
	}

	if !meetingJoined {
		fmt.Println("Meeting join status unclear, but continuing...")
	}

	return nil
}

func (b *Bot) EnableMicrophone() error {
	if !b.running {
		return fmt.Errorf("bot not initialized")
	}

	micSelectors := []string{
		"button[aria-label*='Turn on microphone']",
		"div[data-tooltip*='Turn on microphone']",
		"button[aria-label*='Unmute']",
		"div[aria-label*='Unmute']",
	}

	for _, selector := range micSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(2000),
		})
		if err == nil {
			fmt.Printf("Enabling microphone with selector: %s\n", selector)
			return b.clickWithLogging(selector, "ENABLE_MICROPHONE", "Google Meet - Meeting Controls")
		}
	}

	return fmt.Errorf("could not find microphone enable button")
}

func loadEnv() error {
	file, err := os.Open(".env")
	if err != nil {
		return fmt.Errorf("failed to open .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

func NewBot(headless bool) (*Bot, error) {
	// Load environment variables from .env file
	if err := loadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %v", err)
	}

	email := os.Getenv("GOOGLE_EMAIL")
	password := os.Getenv("GOOGLE_PASSWORD")

	if email == "" {
		return nil, fmt.Errorf("GOOGLE_EMAIL not found in .env file")
	}
	if password == "" {
		return nil, fmt.Errorf("GOOGLE_PASSWORD not found in .env file")
	}

	return &Bot{
		headless: headless,
		email:    email,
		password: password,
	}, nil
}

func (b *Bot) Initialize() error {
	// Setup virtual microphone first

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to run playwright: %v", err)
	}
	b.pw = pw

	// Try to launch browser with additional options for Docker/Linux environments and virtual microphone
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(b.headless),
		Args: []string{
			// Essential Docker/container flags
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--disable-infobars",
			"--disable-features=IsolateOrigins,site-per-process",

			// Audio and media permissions - CRITICAL for virtual mic
			"--use-fake-ui-for-media-stream", // Auto-grant microphone permissions
			"--autoplay-policy=no-user-gesture-required",
		},
	}

	log.Printf("[BROWSER_INIT] Attempting to launch Chromium with Docker-optimized settings...")

	// Add timeout to launch options
	launchOptions.Timeout = playwright.Float(30000) // 30 seconds timeout

	var browser playwright.Browser
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[BROWSER_INIT] Launch attempt %d/%d", attempt, maxRetries)

		browser, err = pw.Chromium.Launch(launchOptions)
		if err != nil {
			log.Printf("[BROWSER_INIT] Attempt %d failed: %v", attempt, err)
			if attempt < maxRetries {
				log.Printf("[BROWSER_INIT] Waiting 2 seconds before retry...")
				time.Sleep(2 * time.Second)
				continue
			}

			// Try with minimal flags as final fallback
			log.Printf("[BROWSER_INIT] All attempts failed, trying minimal fallback...")
			fallbackOptions := playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(b.headless),
				Timeout:  playwright.Float(30000),
				Args: []string{
					"--no-sandbox",
					"--disable-setuid-sandbox",
					"--disable-dev-shm-usage",
					"--disable-gpu",
					"--disable-extensions",
					"--disable-default-apps",
					"--use-fake-ui-for-media-stream",
					"--auto-accept-camera-and-microphone-capture",
					"--log-level=3",
				},
			}

			browser, err = pw.Chromium.Launch(fallbackOptions)
			if err != nil {
				return fmt.Errorf("failed to launch chromium browser after %d attempts and fallback: %v", maxRetries, err)
			}
			log.Printf("[BROWSER_INIT] Fallback launch successful")
		} else {
			log.Printf("[BROWSER_INIT] Launch attempt %d successful", attempt)
			break
		}
	}

	// Add browser disconnect handler for debugging
	browser.On("disconnected", func() {
		log.Printf("[BROWSER_ERROR] Browser disconnected unexpectedly!")
	})

	b.browser = browser

	// Test if browser is still connected
	log.Printf("[BROWSER_INIT] Testing browser connection...")
	if !browser.IsConnected() {
		return fmt.Errorf("browser disconnected immediately after launch")
	}
	log.Printf("[BROWSER_INIT] Browser connection verified")

	// Create a new context with media permissions granted
	contextOptions := playwright.BrowserNewContextOptions{
		Permissions: []string{"camera", "microphone"},
	}

	// In Docker environment, we might need additional configuration
	if os.Getenv("PULSE_SERVER") != "" {
		log.Printf("[BROWSER_INIT] Docker environment detected, configuring for PulseAudio")
	}

	log.Printf("[BROWSER_INIT] Creating browser context...")
	context, err := browser.NewContext(contextOptions)
	if err != nil {
		return fmt.Errorf("failed to create browser context: %v", err)
	}

	log.Printf("[BROWSER_INIT] Creating new page...")
	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %v", err)
	}

	// Test page creation with a simple navigation
	log.Printf("[BROWSER_INIT] Testing page with simple navigation...")
	_, err = page.Goto("data:text/html,<html><body><h1>Browser Test</h1></body></html>", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(10000),
	})
	if err != nil {
		return fmt.Errorf("failed to navigate to test page: %v", err)
	}

	b.page = page
	b.running = true

	log.Printf("[BROWSER_INIT] Browser initialized successfully with virtual microphone support")
	log.Printf("[BROWSER_INIT] Virtual microphone path: /tmp/virtmic")
	log.Printf("[BROWSER_INIT] PulseAudio server: %s", os.Getenv("PULSE_SERVER"))

	return nil
}

func (b *Bot) LeaveMeeting() error {
	if !b.running {
		return fmt.Errorf("bot not initialized")
	}

	fmt.Println("Attempting to leave the meeting...")

	// Try multiple selectors for the leave button
	leaveSelectors := []string{
		"button[aria-label*='Leave call']",
		"div[data-tooltip*='Leave call']",
		"button[aria-label*='End call']",
		"div[data-tooltip*='End call']",
		"button:has-text('Leave call')",
		"div[role='button']:has-text('Leave call')",
		"button[jsname='CQylAd']", // Google Meet specific leave button
		"div[jsname='CQylAd']",
	}

	leaveButtonFound := false
	for _, selector := range leaveSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(3000),
		})
		if err == nil {
			fmt.Printf("Found leave button with selector: %s\n", selector)
			err = b.page.Locator(selector).Click()
			if err != nil {
				fmt.Printf("Failed to click leave button: %v\n", err)
				continue
			}
			leaveButtonFound = true
			break
		}
	}

	if !leaveButtonFound {
		// Try keyboard shortcut as fallback
		fmt.Println("Leave button not found, trying Ctrl+D shortcut...")
		err := b.page.Keyboard().Press("Control+d")
		if err != nil {
			return fmt.Errorf("could not find leave button and keyboard shortcut failed: %v", err)
		}
	}

	// Wait for confirmation that we've left
	fmt.Println("Waiting for meeting exit confirmation...")

	// Look for indicators that we've left the meeting
	exitSelectors := []string{
		"text=You left the meeting",
		"text=Call ended",
		"text=Meeting ended",
		"div[aria-label*='left the meeting']",
		"button:has-text('Rejoin')",
		"div:has-text('Thanks for joining')",
	}

	exitConfirmed := false
	for _, selector := range exitSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(5000),
		})
		if err == nil {
			fmt.Println("Successfully left the meeting!")
			exitConfirmed = true
			break
		}
	}

	if !exitConfirmed {
		// Check if we're back to a Google page or meeting lobby
		currentUrl := b.page.URL()
		if !strings.Contains(currentUrl, "meet.google.com") ||
			strings.Contains(currentUrl, "thanks") ||
			strings.Contains(currentUrl, "feedback") {
			fmt.Println("Meeting left successfully based on URL change")
			exitConfirmed = true
		}
	}

	if !exitConfirmed {
		fmt.Println("Meeting exit status unclear, but leave command was executed")
	}

	return nil
}

func (b *Bot) IsLoggedIn() (bool, error) {
	if !b.running {
		return false, fmt.Errorf("bot not initialized")
	}

	// Navigate to a Google service to check login status
	_, err := b.page.Goto("https://accounts.google.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return false, fmt.Errorf("failed to navigate to Google accounts: %v", err)
	}

	// Check if we're redirected to login page or already logged in
	currentUrl := b.page.URL()

	// If URL contains signin, we're not logged in
	if strings.Contains(currentUrl, "signin") {
		return false, nil
	}

	// Try to find elements that indicate we're logged in
	loggedInSelectors := []string{
		"[data-email]",
		"[aria-label*='Google Account']",
		"img[alt*='profile']",
		"div[data-email]",
	}

	for _, selector := range loggedInSelectors {
		err := b.page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(3000),
		})
		if err == nil {
			fmt.Println("User is already logged in to Google")
			return true, nil
		}
	}

	return false, nil
}

func (b *Bot) TakeScreenshot() ([]byte, error) {
	if !b.running {
		return nil, fmt.Errorf("bot not initialized")
	}

	log.Printf("[SCREENSHOT] Taking screenshot...")

	screenshot, err := b.page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(false), // Only visible area
		Type:     playwright.ScreenshotTypeJpeg,
		Quality:  playwright.Int(80), // Compress for faster streaming
	})

	if err != nil {
		log.Printf("[SCREENSHOT_ERROR] Failed to take screenshot: %v", err)
		return nil, fmt.Errorf("failed to take screenshot: %v", err)
	}

	log.Printf("[SCREENSHOT_SUCCESS] Screenshot taken, size: %d bytes", len(screenshot))
	return screenshot, nil
}
