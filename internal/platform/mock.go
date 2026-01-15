package platform

import (
	"context"
	"fmt"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
)

// MockAdapter implements PlatformAdapter for testing purposes
type MockAdapter struct {
	Windows   []core.Window
	Terminals []core.Terminal
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		Windows:   []core.Window{},
		Terminals: []core.Terminal{},
	}
}

func (m *MockAdapter) Name() string {
	return "mock"
}

func (m *MockAdapter) GetWindows(ctx context.Context) ([]core.Window, error) {
	// Return some dummy data if empty, or the set state
	if len(m.Windows) == 0 {
		return []core.Window{
			{
				AppName:     "Code",
				WindowTitle: "project - VS Code",
				X:           100,
				Y:           100,
				Width:       1200,
				Height:      800,
			},
		}, nil
	}
	return m.Windows, nil
}

func (m *MockAdapter) RestoreWindow(ctx context.Context, window core.Window) error {
	fmt.Printf("[Mock] Restoring window: %s at (%d, %d)\n", window.AppName, window.X, window.Y)
	return nil
}

func (m *MockAdapter) CloseWindow(ctx context.Context, window core.Window) error {
	fmt.Printf("[Mock] Closing window: %s\n", window.AppName)
	return nil
}

func (m *MockAdapter) GetTerminals(ctx context.Context) ([]core.Terminal, error) {
	return m.Terminals, nil
}

func (m *MockAdapter) RestoreTerminal(ctx context.Context, terminal core.Terminal) error {
	fmt.Printf("[Mock] Restoring terminal: %s in %s\n", terminal.TerminalApp, terminal.WorkingDirectory)
	return nil
}

func (m *MockAdapter) GetIDEFiles(ctx context.Context) ([]core.IDEFile, error) {
	return []core.IDEFile{}, nil
}

func (m *MockAdapter) GetBrowserTabs(ctx context.Context) ([]core.BrowserTab, error) {
	return []core.BrowserTab{}, nil
}

func (m *MockAdapter) OpenURL(ctx context.Context, url string, browser string) error {
	fmt.Printf("[Mock] Opening URL: %s in %s\n", url, browser)
	return nil
}

func (m *MockAdapter) GetProcesses(ctx context.Context) ([]core.Process, error) {
	return []core.Process{}, nil
}

func (m *MockAdapter) StartProcess(ctx context.Context, process core.Process) error {
	fmt.Printf("[Mock] Starting process: %s\n", process.Command)
	return nil
}
