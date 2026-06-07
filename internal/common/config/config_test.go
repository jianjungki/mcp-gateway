package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveEnv(t *testing.T) {
	t.Setenv("X_A", "va")
	in := []byte("a: ${X_A:da}\nb: ${X_B:db}")
	out := resolveEnv(in)
	assert.Contains(t, string(out), "a: va")
	assert.Contains(t, string(out), "b: db")
}

func TestLoadConfig_MCPGateway(t *testing.T) {
	tmp := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(tmp)

	// include env expansion and short reload interval to trigger defaulting
	yaml := `
port: 1234
reload_port: 0
reload_interval: 1s
reload_switch: true
forward:
  enabled: true
  mcp_arg:
    key_for_header: XH
  header:
    allow_headers: "A,B"
    ignore_headers: "C"
    case_insensitive: true
    override_existing: false
pid: ${X_PID:/tmp/gw.pid}
`
	file := filepath.Join(tmp, "mcp-gateway.yaml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o644))

	cfg, path, err := LoadConfig[MCPGatewayConfig]("mcp-gateway.yaml")
	assert.NoError(t, err)
	realFile, _ := filepath.EvalSymlinks(file)
	realPath, _ := filepath.EvalSymlinks(path)
	assert.Equal(t, realFile, realPath)
	assert.Equal(t, 1234, cfg.Port)
	// reload interval should be bumped to default (>= 600s)
	assert.GreaterOrEqual(t, int64(cfg.ReloadInterval), int64(600*time.Second))
	assert.True(t, cfg.Forward.Enabled)
	assert.Equal(t, "XH", cfg.Forward.McpArg.KeyForHeader)
	assert.Equal(t, "A,B", cfg.Forward.Header.AllowHeaders)
	assert.Equal(t, "C", cfg.Forward.Header.IgnoreHeaders)
	assert.True(t, cfg.Forward.Header.CaseInsensitive)
}

func TestLoadConfig_MCPGateway_InternalAllowlistString(t *testing.T) {
	tmp := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(tmp)

	yaml := `
tool_access:
  internal_network:
    allowlist: "127.0.0.1/32, localhost,,::1/128"
`
	file := filepath.Join(tmp, "mcp-gateway.yaml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o644))

	cfg, _, err := LoadConfig[MCPGatewayConfig]("mcp-gateway.yaml")
	assert.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1/32", "localhost", "::1/128"}, []string(cfg.ToolAccess.InternalNetwork.Allowlist))
}

func TestLoadConfig_MCPGateway_RedisSentinelCredentials(t *testing.T) {
	tmp := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(tmp)

	t.Setenv("NOTIFIER_REDIS_SENTINEL_USERNAME", "sentinel-notifier")
	t.Setenv("NOTIFIER_REDIS_SENTINEL_PASSWORD", "notifier-pass")
	t.Setenv("SESSION_REDIS_SENTINEL_USERNAME", "sentinel-session")
	t.Setenv("SESSION_REDIS_SENTINEL_PASSWORD", "session-pass")
	t.Setenv("OAUTH2_REDIS_SENTINEL_USERNAME", "sentinel-oauth")
	t.Setenv("OAUTH2_REDIS_SENTINEL_PASSWORD", "oauth-pass")

	yaml := `
notifier:
  redis:
    sentinel_username: "${NOTIFIER_REDIS_SENTINEL_USERNAME:}"
    sentinel_password: "${NOTIFIER_REDIS_SENTINEL_PASSWORD:}"
session:
  redis:
    sentinel_username: "${SESSION_REDIS_SENTINEL_USERNAME:}"
    sentinel_password: "${SESSION_REDIS_SENTINEL_PASSWORD:}"
auth:
  oauth2:
    storage:
      redis:
        sentinel_username: "${OAUTH2_REDIS_SENTINEL_USERNAME:}"
        sentinel_password: "${OAUTH2_REDIS_SENTINEL_PASSWORD:}"
`
	file := filepath.Join(tmp, "mcp-gateway.yaml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o644))

	cfg, _, err := LoadConfig[MCPGatewayConfig]("mcp-gateway.yaml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, "sentinel-notifier", cfg.Notifier.Redis.SentinelUsername)
	assert.Equal(t, "notifier-pass", cfg.Notifier.Redis.SentinelPassword)
	assert.Equal(t, "sentinel-session", cfg.Session.Redis.SentinelUsername)
	assert.Equal(t, "session-pass", cfg.Session.Redis.SentinelPassword)
	assert.Equal(t, "sentinel-oauth", cfg.Auth.OAuth2.Storage.Redis.SentinelUsername)
	assert.Equal(t, "oauth-pass", cfg.Auth.OAuth2.Storage.Redis.SentinelPassword)
}
