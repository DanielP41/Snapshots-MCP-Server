package platform

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
	"golang.org/x/sys/windows"
)

// WindowsAdapterV2 es una versión mejorada con mejor matching
type WindowsAdapterV2 struct {
	matcher *WindowMatcher
}

func NewWindowsAdapterV2() *WindowsAdapterV2 {
	return &WindowsAdapterV2{
		matcher: DefaultMatcher(),
	}
}

func (w *WindowsAdapterV2) Name() string {
	return "windows-v2"
}

// GetWindows obtiene todas las ventanas visibles
func (w *WindowsAdapterV2) GetWindows(ctx context.Context) ([]core.Window, error) {
	var wins []core.Window

	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		// Filter invisible windows
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
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

		// Get Process ID
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		// Get App Name
		appName := w.getProcessName(pid)
		if appName == "" {
			appName = fmt.Sprintf("PID_%d", pid)
		}

		// Get Window Rect
		var r rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))

		win := core.Window{
			WindowTitle: title,
			AppName:     appName,
			AppPath:     "", // Se podría obtener el path completo del exe
			X:           int(r.Left),
			Y:           int(r.Top),
			Width:       int(r.Right - r.Left),
			Height:      int(r.Bottom - r.Top),
			State:       w.getWindowState(hwnd),
			LaunchArgs:  nil,
		}

		wins = append(wins, win)
		return 1
	})

	procEnumWindows.Call(cb, 0)
	return wins, nil
}

// RestoreWindow usa el matcher mejorado para encontrar y restaurar ventanas
func (w *WindowsAdapterV2) RestoreWindow(ctx context.Context, window core.Window) error {
	// Obtener todas las ventanas actuales
	currentWindows, err := w.GetWindows(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current windows: %w", err)
	}

	// Usar el matcher para encontrar la mejor coincidencia
	match := w.matcher.FindBestMatch(window, currentWindows)
	if match == nil {
		return fmt.Errorf("no suitable window found for: %s (app: %s)", window.WindowTitle, window.AppName)
	}

	log.Printf("[WindowRestore] Matched '%s' with '%s' (score: %d)",
		window.WindowTitle, match.Window.WindowTitle, match.Score)

	// Encontrar el HWND de la ventana matched
	foundHwnd := w.findWindowHandle(match.Window.WindowTitle)
	if foundHwnd == 0 {
		return fmt.Errorf("window handle not found for: %s", match.Window.WindowTitle)
	}

	// Restaurar posición y tamaño
	return w.setWindowPosition(foundHwnd, window)
}

// findWindowHandle busca el handle de una ventana por su título
func (w *WindowsAdapterV2) findWindowHandle(title string) syscall.Handle {
	var foundHwnd syscall.Handle

	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		ret, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
		if int(ret) == 0 {
			return 1
		}

		buf := make([]uint16, int(ret)+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(int(ret)+1))
		currentTitle := syscall.UTF16ToString(buf)

		if currentTitle == title {
			foundHwnd = hwnd
			return 0 // Stop enumeration
		}
		return 1
	})

	procEnumWindows.Call(cb, 0)
	return foundHwnd
}

// setWindowPosition mueve y redimensiona una ventana
func (w *WindowsAdapterV2) setWindowPosition(hwnd syscall.Handle, window core.Window) error {
	// SWP_NOZORDER = 0x0004, SWP_NOACTIVATE = 0x0010
	flags := uintptr(0x0004 | 0x0010)

	ret, _, err := procSetWindowPos.Call(
		uintptr(hwnd),
		0,
		uintptr(window.X),
		uintptr(window.Y),
		uintptr(window.Width),
		uintptr(window.Height),
		flags,
	)

	if ret == 0 {
		return fmt.Errorf("SetWindowPos failed: %v", err)
	}

	// Restaurar estado si es necesario
	if window.State == "maximized" {
		procShowWindow.Call(uintptr(hwnd), 3) // SW_MAXIMIZE
	} else if window.State == "minimized" {
		procShowWindow.Call(uintptr(hwnd), 6) // SW_MINIMIZE
	} else {
		procShowWindow.Call(uintptr(hwnd), 1) // SW_SHOWNORMAL
	}

	return nil
}

// getWindowState detecta el estado de una ventana
func (w *WindowsAdapterV2) getWindowState(hwnd syscall.Handle) string {
	// IsIconic = minimized
	ret, _, _ := user32.NewProc("IsIconic").Call(uintptr(hwnd))
	if ret != 0 {
		return "minimized"
	}

	// IsZoomed = maximized
	ret, _, _ = user32.NewProc("IsZoomed").Call(uintptr(hwnd))
	if ret != 0 {
		return "maximized"
	}

	return "normal"
}

// getProcessName obtiene el nombre del proceso dado su PID
func (w *WindowsAdapterV2) getProcessName(pid uint32) string {
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

// Implementación de métodos restantes (sin cambios significativos)
func (w *WindowsAdapterV2) CloseWindow(ctx context.Context, window core.Window) error {
	return nil // No implementado por seguridad
}

func (w *WindowsAdapterV2) GetTerminals(ctx context.Context) ([]core.Terminal, error) {
	windowsList, err := w.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	var terminals []core.Terminal
	for _, win := range windowsList {
		if isTerminal(win.AppName) {
			terminals = append(terminals, core.Terminal{
				TerminalApp:      win.AppName,
				ActiveCommand:    win.WindowTitle,
				WorkingDirectory: "",
				ShellType:        guessShell(win.AppName),
			})
		}
	}
	return terminals, nil
}

func (w *WindowsAdapterV2) RestoreTerminal(ctx context.Context, terminal core.Terminal) error {
	return nil // No implementado
}

func (w *WindowsAdapterV2) OpenURL(ctx context.Context, url string, browser string) error {
	return nil // No implementado
}

func (w *WindowsAdapterV2) GetBrowserTabs(ctx context.Context) ([]core.BrowserTab, error) {
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
				URL:         "",
				IsPinned:    false,
			})
		}
	}
	return tabs, nil
}

func (w *WindowsAdapterV2) GetIDEFiles(ctx context.Context) ([]core.IDEFile, error) {
	windowsList, err := w.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	var files []core.IDEFile
	for _, win := range windowsList {
		if isIDE(win.AppName) {
			files = append(files, core.IDEFile{
				IDEName:  win.AppName,
				FilePath: extractProjectFromTitle(win.WindowTitle),
				IsActive: true,
			})
		}
	}
	return files, nil
}

func (w *WindowsAdapterV2) GetProcesses(ctx context.Context) ([]core.Process, error) {
	return []core.Process{}, nil
}

func (w *WindowsAdapterV2) StartProcess(ctx context.Context, process core.Process) error {
	return nil
}
