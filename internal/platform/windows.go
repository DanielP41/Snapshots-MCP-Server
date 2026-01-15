package platform

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
	"golang.org/x/sys/windows"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procShowWindow               = user32.NewProc("ShowWindow")
)

type WindowsAdapter struct{}

func NewWindowsAdapter() *WindowsAdapter {
	return &WindowsAdapter{}
}

func (w *WindowsAdapter) Name() string {
	return "windows"
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (w *WindowsAdapter) GetWindows(ctx context.Context) ([]core.Window, error) {
	var wins []core.Window

	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		// Filter invisible windows
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1 // Continue enumeration
		}

		// Get Title
		ret, _, _ = procGetWindowTextLengthW.Call(uintptr(hwnd))
		len := int(ret)
		if len == 0 {
			return 1
		}

		buf := make([]uint16, len+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len+1))
		title := syscall.UTF16ToString(buf)

		// Get Process ID (PID)
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		// Get Real App Name
		appName := w.getProcessName(pid)
		if appName == "" {
			appName = fmt.Sprintf("PID_%d", pid)
		}

		// Get Rect
		var r rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))

		win := core.Window{
			WindowTitle: title,
			AppName:     appName,
			X:           int(r.Left),
			Y:           int(r.Top),
			Width:       int(r.Right - r.Left),
			Height:      int(r.Bottom - r.Top),
			State:       "normal",
			LaunchArgs:  nil,
		}

		wins = append(wins, win)
		return 1
	})

	procEnumWindows.Call(cb, 0)

	return wins, nil
}

func (w *WindowsAdapter) RestoreWindow(ctx context.Context, window core.Window) error {
	// 1. Get current windows to use as candidates
	// We need both the metadata (for matching) and the HWND (for action)
	// Since GetWindows returns core.Window (without HWND), we need a way to map back.
	// For this implementation, we will re-enumerate and build a map.

	type windowWithHandle struct {
		Window core.Window
		Hwnd   syscall.Handle
	}

	var candidates []windowWithHandle

	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		// Visible check
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
		}

		// Title
		ret, _, _ = procGetWindowTextLengthW.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
		}

		buf := make([]uint16, int(ret)+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(int(ret)+1))
		title := syscall.UTF16ToString(buf)

		// PID & AppName
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
		appName := w.getProcessName(pid)
		if appName == "" {
			appName = fmt.Sprintf("PID_%d", pid)
		}

		// Rect
		var r rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))

		candidate := core.Window{
			WindowTitle: title,
			AppName:     appName,
			Width:       int(r.Right - r.Left),
			Height:      int(r.Bottom - r.Top),
		}

		candidates = append(candidates, windowWithHandle{
			Window: candidate,
			Hwnd:   hwnd,
		})
		return 1
	})
	procEnumWindows.Call(cb, 0)

	// 2. Use Matcher
	matcher := DefaultMatcher()

	// Prepare simple list for matcher
	var candidateList []core.Window
	for _, c := range candidates {
		candidateList = append(candidateList, c.Window)
	}

	match := matcher.FindBestMatch(window, candidateList)
	if match == nil {
		return fmt.Errorf("no suitable match found for window: %s", window.WindowTitle)
	}

	// 3. Find the HWND for the best match
	var targetHwnd syscall.Handle
	for _, c := range candidates {
		// Identify by exact object properties since we just built it
		// Or simpler: match on Title + AppName + Size
		if c.Window.WindowTitle == match.Window.WindowTitle &&
			c.Window.AppName == match.Window.AppName &&
			c.Window.Width == match.Window.Width {
			targetHwnd = c.Hwnd
			break
		}
	}

	if targetHwnd != 0 {
		// Restore Position
		procSetWindowPos.Call(
			uintptr(targetHwnd),
			0,
			uintptr(window.X),
			uintptr(window.Y),
			uintptr(window.Width),
			uintptr(window.Height),
			0x0004, // SWP_NOZORDER
		)
		procShowWindow.Call(uintptr(targetHwnd), 5) // SW_SHOW
		return nil
	}

	return fmt.Errorf("failed to resolve handle for matched window")
}

func (w *WindowsAdapter) CloseWindow(ctx context.Context, window core.Window) error {
	return nil // Not implemented
}

// Helper struct for Process
type processInfo struct {
	PID  uint32
	Name string
}

func (w *WindowsAdapter) getProcessName(pid uint32) string {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snapshot)

	var pe32 windows.ProcessEntry32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	if err := windows.Process32First(snapshot, &pe32); err != nil {
		return ""
	}

	for {
		if pe32.ProcessID == pid {
			return windows.UTF16ToString(pe32.ExeFile[:])
		}
		if err := windows.Process32Next(snapshot, &pe32); err != nil {
			break
		}
	}
	return ""
}

func (w *WindowsAdapter) GetTerminals(ctx context.Context) ([]core.Terminal, error) {
	windowsList, err := w.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	var terminals []core.Terminal
	for _, win := range windowsList {
		if isTerminal(win.AppName) {
			terminals = append(terminals, core.Terminal{
				TerminalApp:      win.AppName,     // e.g. "WindowsTerminal.exe"
				ActiveCommand:    win.WindowTitle, // best guess
				WorkingDirectory: "",              // hard to get without injection
				ShellType:        guessShell(win.AppName),
			})
		}
	}
	return terminals, nil
}

func (w *WindowsAdapter) RestoreTerminal(ctx context.Context, terminal core.Terminal) error {
	return nil
}

func (w *WindowsAdapter) OpenURL(ctx context.Context, url string, browser string) error {
	return nil
}

func (w *WindowsAdapter) GetBrowserTabs(ctx context.Context) ([]core.BrowserTab, error) {
	windowsList, err := w.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	var tabs []core.BrowserTab
	for _, win := range windowsList {
		if isBrowser(win.AppName) {
			tabs = append(tabs, core.BrowserTab{
				BrowserName: win.AppName,
				Title:       win.WindowTitle,
				URL:         "", // impossible without extensions
				IsPinned:    false,
			})
		}
	}
	return tabs, nil
}

func (w *WindowsAdapter) GetIDEFiles(ctx context.Context) ([]core.IDEFile, error) {
	windowsList, err := w.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	var files []core.IDEFile
	for _, win := range windowsList {
		if isIDE(win.AppName) {
			files = append(files, core.IDEFile{
				IDEName:  win.AppName,
				FilePath: extractProjectFromTitle(win.WindowTitle), // e.g. "main.go - Project - VS Code"
				IsActive: true,
			})
		}
	}
	return files, nil
}

func (w *WindowsAdapter) GetProcesses(ctx context.Context) ([]core.Process, error) {
	return []core.Process{}, nil
}

func (w *WindowsAdapter) StartProcess(ctx context.Context, process core.Process) error {
	return nil
}

// Classification Helpers
func isTerminal(app string) bool {
	switch app {
	case "WindowsTerminal.exe", "cmd.exe", "powershell.exe", "pwsh.exe", "mintty.exe":
		return true
	}
	return false
}

func isBrowser(app string) bool {
	switch app {
	case "chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "opera.exe":
		return true
	}
	return false
}

func isIDE(app string) bool {
	// "Code.exe" is VS Code
	switch app {
	case "Code.exe", "idea64.exe", "goland64.exe":
		return true
	}
	return false
}

func guessShell(app string) string {
	if app == "cmd.exe" {
		return "cmd"
	}
	if app == "powershell.exe" || app == "pwsh.exe" {
		return "powershell"
	}
	return "unknown"
}

func extractProjectFromTitle(title string) string {
	// VS Code: "filename.go - ProjectName - Visual Studio Code"
	// Heuristic: Extract the project name from the title string.
	return title // Currently returns the full title.
}
