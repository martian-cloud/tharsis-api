package joblogger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogBufferBytes(t *testing.T) {
	buf, err := NewLogBuffer()
	require.NoError(t, err)
	defer buf.Close()

	_, err = buf.Write([]byte("0123456789"))
	require.NoError(t, err)

	t.Run("reads a sub-range at an offset", func(t *testing.T) {
		got, err := buf.Bytes(2, 3)
		require.NoError(t, err)
		assert.Equal(t, "234", string(got))
	})

	t.Run("a read past the end is clamped to what exists", func(t *testing.T) {
		got, err := buf.Bytes(8, 10)
		require.NoError(t, err)
		assert.Equal(t, "89", string(got))
	})

	t.Run("an offset at or past the end returns empty", func(t *testing.T) {
		got, err := buf.Bytes(10, 5)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("a zero-length read returns empty", func(t *testing.T) {
		got, err := buf.Bytes(0, 0)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestLogBufferSize(t *testing.T) {
	buf, err := NewLogBuffer()
	require.NoError(t, err)
	defer buf.Close()

	assert.Equal(t, 0, buf.Size())

	_, err = buf.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, buf.Size())

	_, err = buf.Write([]byte(" world"))
	require.NoError(t, err)
	assert.Equal(t, 11, buf.Size())
}

func TestLogBufferLimit(t *testing.T) {
	buf, err := NewLogBuffer()
	require.NoError(t, err)
	defer buf.Close()

	buf.SetLimit(20)

	// Writing past the limit succeeds from the caller's perspective (returns len(p), nil) so the job
	// keeps running, but the buffer stops accumulating new output and appends a one-time notice.
	n, err := buf.Write([]byte(strings.Repeat("a", 100)))
	require.NoError(t, err)
	assert.Equal(t, 100, n, "Write reports the full input as written even past the limit")

	// Size is bounded near the limit (the limit's worth of data plus the appended notice).
	assert.LessOrEqual(t, 20, buf.Size())

	contents, err := buf.Bytes(0, buf.Size())
	require.NoError(t, err)
	assert.Contains(t, string(contents), "exceeded limit")

	// Further writes are silently dropped (still no error to the caller).
	sizeAfterLimit := buf.Size()
	n, err = buf.Write([]byte("more output"))
	require.NoError(t, err)
	assert.Equal(t, len("more output"), n)
	assert.Equal(t, sizeAfterLimit, buf.Size(), "no additional bytes accumulate once the limit is hit")
}
