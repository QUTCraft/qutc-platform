package serveradapter

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	ModeMock = "mock"
	ModeRCON = "rcon"
)

var ErrCommandNotAllowed = errors.New("server command is not allowed")

var minecraftGameIDPattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

type Status struct {
	Enabled       bool       `json:"enabled"`
	Adapter       string     `json:"adapter"`
	Mode          string     `json:"mode"`
	Label         string     `json:"label"`
	State         string     `json:"state"`
	Version       *string    `json:"version,omitempty"`
	OnlinePlayers int        `json:"online_players"`
	MaxPlayers    int        `json:"max_players"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastCommandAt *time.Time `json:"last_command_at,omitempty"`
}

type Result struct {
	Accepted   bool      `json:"accepted"`
	Executed   bool      `json:"executed"`
	Mode       string    `json:"mode"`
	Message    string    `json:"message"`
	ExecutedAt time.Time `json:"executed_at"`
}

type Adapter interface {
	Name() string
	Mode() string
	Status(context.Context) (Status, error)
	Execute(context.Context, string) (Result, error)
	AddWhitelist(context.Context, string) (Result, error)
}

type timeoutAdapter struct {
	adapter Adapter
	timeout time.Duration
}

func WithTimeout(adapter Adapter, timeout time.Duration) Adapter {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &timeoutAdapter{adapter: adapter, timeout: timeout}
}

func (a *timeoutAdapter) Name() string { return a.adapter.Name() }
func (a *timeoutAdapter) Mode() string { return a.adapter.Mode() }

func (a *timeoutAdapter) Status(ctx context.Context) (Status, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	return a.adapter.Status(ctx)
}

func (a *timeoutAdapter) Execute(ctx context.Context, command string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	return a.adapter.Execute(ctx, command)
}

func (a *timeoutAdapter) AddWhitelist(ctx context.Context, gameID string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	return a.adapter.AddWhitelist(ctx, gameID)
}

type MockAdapter struct {
	mu            sync.RWMutex
	lastCommandAt *time.Time
}

func NewMock() *MockAdapter {
	return &MockAdapter{}
}

func (a *MockAdapter) Name() string {
	return "minecraft-mock"
}

func (a *MockAdapter) Mode() string {
	return ModeMock
}

func (a *MockAdapter) Status(context.Context) (Status, error) {
	a.mu.RLock()
	lastCommandAt := a.lastCommandAt
	a.mu.RUnlock()
	return Status{
		Enabled:       true,
		Adapter:       a.Name(),
		Mode:          a.Mode(),
		Label:         "QUTCraft Minecraft Mock",
		State:         "maintenance",
		OnlinePlayers: 0,
		MaxPlayers:    60,
		UpdatedAt:     time.Now().UTC(),
		LastCommandAt: lastCommandAt,
	}, nil
}

func (a *MockAdapter) Execute(_ context.Context, command string) (Result, error) {
	command = strings.TrimSpace(command)
	if !AllowedCommand(command) {
		return Result{}, ErrCommandNotAllowed
	}
	now := time.Now().UTC()
	a.mu.Lock()
	a.lastCommandAt = &now
	a.mu.Unlock()
	return Result{
		Accepted:   true,
		Executed:   false,
		Mode:       a.Mode(),
		Message:    "Mock 适配器已模拟受理命令，未连接真实 RCON。",
		ExecutedAt: now,
	}, nil
}

func (a *MockAdapter) AddWhitelist(_ context.Context, gameID string) (Result, error) {
	gameID = strings.TrimSpace(gameID)
	if !minecraftGameIDPattern.MatchString(gameID) {
		return Result{}, errors.New("invalid whitelist game id")
	}
	now := time.Now().UTC()
	return Result{
		Accepted:   true,
		Executed:   false,
		Mode:       a.Mode(),
		Message:    "Mock 适配器已模拟白名单同步，未连接真实 RCON。",
		ExecutedAt: now,
	}, nil
}

func AllowedCommand(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" || len([]rune(command)) > 256 || strings.ContainsAny(command, "\r\n") {
		return false
	}
	for _, allowed := range []string{"list", "save-all", "time set day", "weather clear"} {
		if command == allowed {
			return true
		}
	}
	return strings.HasPrefix(command, "say ") && len(strings.TrimSpace(strings.TrimPrefix(command, "say "))) > 0
}
