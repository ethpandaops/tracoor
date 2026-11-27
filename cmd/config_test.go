package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const awkwardValue = "a\tb #c\nd: e\"f'g\\h"

func unsetEnv(t *testing.T, name string) {
	t.Helper()

	t.Setenv(name, "")
	require.NoError(t, os.Unsetenv(name))
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestDecodeConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		set     map[string]string
		unset   []string
		want    map[string]any
		wantErr string
	}{
		{
			name:    "variable is replaced",
			content: "dsn: postgres://user:${DB_PASS}@db:5432/tracoor",
			set:     map[string]string{"DB_PASS": "s3cret"},
			want:    map[string]any{"dsn": "postgres://user:s3cret@db:5432/tracoor"},
		},
		{
			name:    "multiple variables in one value",
			content: "addr: ${HOST}:${PORT}",
			set:     map[string]string{"HOST": "example.com", "PORT": "8080"},
			want:    map[string]any{"addr": "example.com:8080"},
		},
		{
			name:    "default used when variable unset",
			content: "addr: ${HOST:-localhost}",
			unset:   []string{"HOST"},
			want:    map[string]any{"addr": "localhost"},
		},
		{
			name:    "default used when variable empty",
			content: "addr: ${HOST:-localhost}",
			set:     map[string]string{"HOST": ""},
			want:    map[string]any{"addr": "localhost"},
		},
		{
			name:    "default ignored when variable set",
			content: "addr: ${HOST:-localhost}",
			set:     map[string]string{"HOST": "example.com"},
			want:    map[string]any{"addr": "example.com"},
		},
		{
			name:    "empty default allows unset variable",
			content: "token: \"${TOKEN:-}\"",
			unset:   []string{"TOKEN"},
			want:    map[string]any{"token": ""},
		},
		{
			name:    "empty variable without default is an error",
			content: "token: ${TOKEN}",
			set:     map[string]string{"TOKEN": ""},
			wantErr: "config references unset or empty environment variables: TOKEN",
		},
		{
			name:    "unset variables without default are an error",
			content: "user: ${DB_USER}\nnested:\n  - pass: ${DB_PASS}\nagain: ${DB_USER}",
			unset:   []string{"DB_USER", "DB_PASS"},
			wantErr: "config references unset or empty environment variables: DB_USER, DB_PASS",
		},
		{
			name:    "comments are not expanded",
			content: "# dsn: ${DB_PASS}\naddr: ${HOST} # ${DB_PASS}",
			set:     map[string]string{"HOST": "example.com"},
			unset:   []string{"DB_PASS"},
			want:    map[string]any{"addr": "example.com"},
		},
		{
			name:    "keys are not expanded",
			content: "${HOST}: value",
			set:     map[string]string{"HOST": "example.com"},
			want:    map[string]any{"${HOST}": "value"},
		},
		{
			name:    "literal dollar signs are left unchanged",
			content: "password: pa$$word $HOST $5 ${HOST",
			set:     map[string]string{"HOST": "example.com"},
			want:    map[string]any{"password": "pa$$word $HOST $5 ${HOST"},
		},
		{
			name:    "value with YAML syntax arrives unchanged",
			content: "plain: ${P}\ndouble: \"${P}\"\nsingle: '${P}'\nlist:\n  - ${P}\nanchored: &a ${P}\naliased: *a",
			set:     map[string]string{"P": awkwardValue},
			want: map[string]any{
				"plain":    awkwardValue,
				"double":   awkwardValue,
				"single":   awkwardValue,
				"list":     []any{awkwardValue},
				"anchored": awkwardValue,
				"aliased":  awkwardValue,
			},
		},
		{
			name:    "unquoted value resolves its type, quoted value stays a string",
			content: "port: ${PORT}\nquoted: \"${PORT}\"\nenabled: ${ENABLED}",
			set:     map[string]string{"PORT": "8080", "ENABLED": "true"},
			want:    map[string]any{"port": 8080, "quoted": "8080", "enabled": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.set {
				t.Setenv(k, v)
			}

			for _, k := range tt.unset {
				unsetEnv(t, k)
			}

			var got map[string]any

			err := decodeConfig([]byte(tt.content), &got)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeConfigWithoutPlaceholdersIsUnchanged(t *testing.T) {
	for _, file := range []string{
		"../example_server_config.yaml",
		"../example_agent_config.yaml",
		"../example_single_config.yaml",
	} {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			require.NoError(t, err)

			var want, got any

			require.NoError(t, yaml.Unmarshal(data, &want))
			require.NoError(t, decodeConfig(data, &got))
			assert.Equal(t, want, got)
		})
	}
}

func TestLoadServerConfigExpandsEnvVars(t *testing.T) {
	t.Setenv("TRACOOR_TEST_DB_PASS", awkwardValue)
	t.Setenv("TRACOOR_TEST_PRESTOP", "7")

	path := writeConfig(t, `
addr: ":8081"
preStopSleepSeconds: ${TRACOOR_TEST_PRESTOP}
persistence:
  dsn: ${TRACOOR_TEST_DB_PASS}
  driver_name: postgres
`)

	config, err := loadServerConfigFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, awkwardValue, config.Persistence.DSN)
	assert.Equal(t, 7, config.PreStopSleepSeconds)
	assert.Equal(t, ":8081", config.Addr)
}

func TestLoadServerConfigFailsOnUnsetEnvVar(t *testing.T) {
	unsetEnv(t, "TRACOOR_TEST_DB_PASS")

	path := writeConfig(t, `
persistence:
  dsn: "postgres://user:${TRACOOR_TEST_DB_PASS}@db:5432/tracoor"
`)

	_, err := loadServerConfigFromFile(path)
	require.ErrorContains(t, err, "TRACOOR_TEST_DB_PASS")
}

func TestLoadAgentConfigExpandsEnvVars(t *testing.T) {
	t.Setenv("TRACOOR_TEST_BEACON_AUTH", awkwardValue)
	t.Setenv("TRACOOR_TEST_S3_SECRET", awkwardValue)

	path := writeConfig(t, `
name: test-agent
ethereum:
  beacon:
    nodeAddress: http://localhost:5052
    nodeHeaders:
      Authorization: "${TRACOOR_TEST_BEACON_AUTH}"
store:
  type: s3
  config:
    access_secret: ${TRACOOR_TEST_S3_SECRET}
`)

	config, err := loadagentConfigFromFile(path)
	require.NoError(t, err)

	require.NotNil(t, config.Ethereum.Beacon)
	assert.Equal(t, awkwardValue, config.Ethereum.Beacon.NodeHeaders["Authorization"])
	assert.Equal(t, "test-agent", config.Name)

	require.NotNil(t, config.Store)

	var s3 store.S3StoreConfig

	require.NoError(t, config.Store.Config.Unmarshal(&s3))
	assert.Equal(t, awkwardValue, s3.AccessSecret)
}
