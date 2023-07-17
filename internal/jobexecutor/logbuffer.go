/*

Portions of this file from
https://gitlab.com/gitlab-org/gitlab-runner/-/blob/6d6b3946bc11375fbd67e4a00a6a51e978cdaf4f/helpers/trace/buffer.go.

The MIT License (MIT)

Copyright (c) 2015-2019 GitLab B.V.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

Tharsis Modifications:

	- Portions of the code have been altered to meet Tharsis' use-case.
	Such examples include but are not limited to renaming struct(s),
	function(s), removing certain function(s), etc.

	- Function(s), type(s), struct(s) like:
		- func WithURLParamMasking(enabled bool) Option
		- type Option func(*options) error
		- type options struct
		- ...
	have been renamed and / or removed as needed for use within Tharsis.
*/

package jobexecutor

import (
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const defaultBytesLimit = 4 * 1024 * 1024 // 4MiB

var errLogLimitExceeded = errors.New("log limit exceeded")

// LogBuffer stores logs in a file and limits the amount of logs written
type LogBuffer struct {
	checksum hash.Hash32
	lw       *limitWriter
	logFile  *os.File
	lock     sync.RWMutex
}

// SetLimit sets the limit for log data in bytes
func (b *LogBuffer) SetLimit(size int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.lw.limit = int64(size)
}

// Size returns the number of bytes written
func (b *LogBuffer) Size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.lw == nil {
		return 0
	}
	return int(b.lw.written)
}

// Bytes returns a chunk of bytes from the log file
func (b *LogBuffer) Bytes(offset, n int) ([]byte, error) {
	return io.ReadAll(io.NewSectionReader(b.logFile, int64(offset), int64(n)))
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	src := p
	var n int
	for len(src) > 0 {
		written, err := b.lw.Write(src)
		// if we get a log limit exceeded error, we've written the log limit
		// notice out to the log and will now silently not write any additional
		// data: we return len(p), nil so the caller continues as normal.
		if err == errLogLimitExceeded {
			return len(p), nil
		}
		if err != nil {
			return n, err
		}

		// the text/transformer implementation can return n < len(p) without an
		// error. For this reason, we continue writing whatever data is left
		// unless nothing was written (therefore zero progress) on our call to
		// Write().
		if written == 0 {
			return n, io.ErrShortWrite
		}

		src = src[written:]
		n += written
	}

	return n, nil
}

// Close removes the underlying log file
func (b *LogBuffer) Close() {
	_ = b.logFile.Close()
	_ = os.Remove(b.logFile.Name())
}

// Checksum returns the checksum for the log data
func (b *LogBuffer) Checksum() string {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return fmt.Sprintf("crc32:%08x", b.checksum.Sum32())
}

type limitWriter struct {
	w       io.Writer
	written int64
	limit   int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	capacity := w.limit - w.written

	if capacity <= 0 {
		return 0, errLogLimitExceeded
	}

	if int64(len(p)) >= capacity {
		p = p[:capacity]
		n, err := w.w.Write(p)
		if err == nil {
			err = errLogLimitExceeded
		}
		if n < 0 {
			n = 0
		}
		w.written += int64(n)
		w.writeLimitExceededMessage()

		return n, err
	}

	n, err := w.w.Write(p)
	if n < 0 {
		n = 0
	}
	w.written += int64(n)
	return n, err
}

func (w *limitWriter) writeLimitExceededMessage() {
	n, _ := fmt.Fprintf(
		w.w,
		"\nJob's log exceeded limit of %v bytes.\n"+
			"Job execution will continue but no more output will be collected.\n",
		w.limit,
	)
	w.written += int64(n)
}

// NewLogBuffer returns a new LogBuffer
func NewLogBuffer() (*LogBuffer, error) {
	logFile, err := os.CreateTemp("", "log_buffer")
	if err != nil {
		return nil, err
	}

	buffer := &LogBuffer{
		logFile:  logFile,
		checksum: crc32.NewIEEE(),
	}

	buffer.lw = &limitWriter{
		w:       io.MultiWriter(buffer.logFile, buffer.checksum),
		written: 0,
		limit:   defaultBytesLimit,
	}

	return buffer, nil
}
