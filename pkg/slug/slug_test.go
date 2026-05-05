package slug

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates slug from directory", func(t *testing.T) {
		srcDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte(`resource "null" "test" {}`), 0o600))

		s, err := New(srcDir)

		require.NoError(t, err)
		defer os.Remove(s.SlugPath)
		assert.NotEmpty(t, s.SlugPath)
		assert.NotEmpty(t, s.SHASum)
		assert.Greater(t, s.Size, int64(0))

		info, err := os.Stat(s.SlugPath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("errors on nonexistent directory", func(t *testing.T) {
		_, err := New("/nonexistent/path")

		assert.Error(t, err)
	})

	t.Run("errors on file instead of directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "notadir.txt")
		require.NoError(t, os.WriteFile(file, []byte("hello"), 0o600))

		_, err := New(file)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("produces deterministic checksum", func(t *testing.T) {
		srcDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte(`resource "null" "test" {}`), 0o600))

		s1, err := New(srcDir)
		require.NoError(t, err)
		defer os.Remove(s1.SlugPath)

		s2, err := New(srcDir)
		require.NoError(t, err)
		defer os.Remove(s2.SlugPath)

		assert.Equal(t, s1.SHASum, s2.SHASum)
	})

	t.Run("digest is stable after changing file timestamps", func(t *testing.T) {
		srcDir := t.TempDir()
		filePath := filepath.Join(srcDir, "main.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(`resource "null" "test" {}`), 0o600))

		s1, err := New(srcDir)
		require.NoError(t, err)
		defer os.Remove(s1.SlugPath)

		// Change the file's modification time.
		future := time.Now().Add(48 * time.Hour)
		require.NoError(t, os.Chtimes(filePath, future, future))

		s2, err := New(srcDir)
		require.NoError(t, err)
		defer os.Remove(s2.SlugPath)

		assert.Equal(t, s1.SHASum, s2.SHASum)
	})
}

func TestSlugOpen(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte("# test"), 0o600))

	s, err := New(srcDir)
	require.NoError(t, err)
	defer os.Remove(s.SlugPath)

	reader, err := s.Open()
	require.NoError(t, err)
	defer reader.Close()

	buf := make([]byte, 16)
	n, err := reader.Read(buf)

	assert.NoError(t, err)
	assert.Greater(t, n, 0)
}
