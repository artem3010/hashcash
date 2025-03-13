package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     string
		unset        bool
		defaultValue string
		expected     string
	}{
		{
			name:         "Variable exists",
			key:          "EXISTING_KEY",
			setValue:     "found",
			unset:        false,
			defaultValue: "default",
			expected:     "found",
		},
		{
			name:         "Variable does not exist, return default",
			key:          "MISSING_KEY",
			setValue:     "",
			unset:        true,
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unset {
				_ = os.Unsetenv(tc.key)
			} else {
				_ = os.Setenv(tc.key, tc.setValue)
			}

			actual := GetEnv(tc.key, tc.defaultValue)
			require.Equal(t, tc.expected, actual, "unexpected environment variable value")

			_ = os.Unsetenv(tc.key)
		})
	}
}
func TestLoadEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "envtest")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	envFilePath := filepath.Join(tmpDir, ".env")
	content := []byte("TEST_ENV_VAR=loaded_value\n")
	err = os.WriteFile(envFilePath, content, 0644)
	require.NoError(t, err)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	LoadEnv()

	value := os.Getenv("TEST_ENV_VAR")
	require.Equal(t, "loaded_value", value, "expected environment variable to be loaded from .env")

	os.Unsetenv("TEST_ENV_VAR")
}
