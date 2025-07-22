1. Enabling Auto-Scrolling in the Fyne Terminal

Cause: Fyne widgets don't automatically scroll to the bottom when content is added. You need to explicitly tell the scroll container to do so.

Solution:

Ensure the widget displaying the logs (e.g., widget.List, widget.TextGrid, or a container.NewVBox of widget.Labels) is placed inside a container.Scroll.
When you add a new log line and update/refresh the widget, get a reference to the scroll container and call its ScrollToBottom() method.
Conceptual Fyne Code:

go
package main

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
	"time" // Import time

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var logBuffer []string
var logLock sync.Mutex
var logList *widget.List // Using List for easy ScrollToBottom
var scrollContainer *container.Scroll

// Function to add a line and trigger UI update + scroll
func addLogLine(line string, isError bool) {
	logLock.Lock()
	prefix := ""
	if isError {
		// Prefix errors coming from stderr (if Go code doesn't add its own prefix)
		prefix = "Blockchain Error: "
	}
	logBuffer = append(logBuffer, prefix+line)

	// Optional: Limit buffer size
	const maxLogLines = 1000
	if len(logBuffer) > maxLogLines {
		logBuffer = logBuffer[len(logBuffer)-maxLogLines:]
	}
	logLock.Unlock()

	// Update the UI on the main thread and scroll
	// Use a small delay with time.AfterFunc to ensure layout is updated before scrolling
	time.AfterFunc(10*time.Millisecond, func() {
		if logList != nil {
			logList.Refresh()
		}
		if scrollContainer != nil {
			scrollContainer.ScrollToBottom()
		}
	})

	// Trigger a general UI refresh check (might help responsiveness)
	fyne.CurrentApp().SendNotification(&fyne.Notification{})
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("KNIRVCHAIN Console")

	// --- Log Display Widget (List is convenient) ---
	logList = widget.NewList(
		func() int {
			logLock.Lock()
			defer logLock.Unlock()
			return len(logBuffer)
		},
		func() fyne.CanvasObject {
			// Use a Label that can wrap text
			l := widget.NewLabel("template")
			l.Wrapping = fyne.TextWrapWord
			return l
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			logLock.Lock()
			defer logLock.Unlock()
			if i < len(logBuffer) {
				o.(*widget.Label).SetText(logBuffer[i])
			}
		},
	)

	// --- Scroll Container ---
	scrollContainer = container.NewScroll(logList) // Put the List inside the Scroll
	scrollContainer.SetMinSize(fyne.NewSize(600, 400))

	// --- Command Execution ---
	go func() {
		// --- IMPORTANT: Update this path ---
		// Option 1: Run compiled binary
		// cmdPath := "/path/to/your/knirvchain_binary"
		// cmd := exec.Command(cmdPath)

		// Option 2: Use 'go run' (slower startup)
		cmdPath := "/home/gperry/Documents/GitHub/KNIRVCHAIN_GO_Verifyer/main.go" // Adjust if main is elsewhere
		cmd := exec.Command("go", "run", cmdPath)
		cmd.Dir = "/home/gperry/Documents/GitHub/KNIRVCHAIN_GO_Verifyer" // Set working directory if needed

		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		err := cmd.Start()
		if err != nil {
			addLogLine("FATAL: Failed to start blockchain process: "+err.Error(), true)
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine to read stdout (Info logs)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				addLogLine(scanner.Text(), false) // Not an error
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				addLogLine("Error reading stdout: "+err.Error(), true)
			}
		}()

		// Goroutine to read stderr (Error logs)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				addLogLine(scanner.Text(), true) // IS an error
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				addLogLine("Error reading stderr: "+err.Error(), true)
			}
		}()

		wg.Wait() // Wait for both pipes to close

		err = cmd.Wait()
		if err != nil {
			addLogLine("Blockchain process exited with error: "+err.Error(), true)
		} else {
			addLogLine("Blockchain process exited successfully.", false)
		}
	}()

	myWindow.SetContent(scrollContainer) // Use the scroll container
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.SetMaster() // Make sure window closes app on exit
	myWindow.ShowAndRun()
}
Summary of Steps:

(Recommended) Modify your Go blockchain code (blockchain_struct.go, db.go, etc.) to use standard log for info (stdout) and write actual errors/critical messages to os.Stderr (or use log.Fatal).
Update your Fyne application's code that runs the blockchain process (exec.Command).
Capture stdout and stderr separately from the command.
Wrap your log display widget (widget.List, widget.TextGrid, etc.) in a container.Scroll.
When adding lines to the display:
Prefix only the lines that came from stderr (if needed).
Refresh the display widget.
Call scrollContainer.ScrollToBottom() (potentially with a small delay like time.AfterFunc to ensure the layout is updated first).