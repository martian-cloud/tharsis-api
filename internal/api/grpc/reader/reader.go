// Package reader implements a size-limited reader for uploading templates of multiple kinds.
package reader

import (
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// LimitReader is a size-limited reader for uploading templates of multiple kinds.
type LimitReader struct {
	// stream is the GRPC streaming connection.
	stream func() ([]byte, error)

	// For internal bookkeeping: any data that has been read from the stream but not yet returned to the caller.
	pendingChunkData []byte

	// limitRemaining is the number of bytes remaining that are allowed to be read
	limitRemaining int64
}

// NewLimitReader creates a limit reader instance.
func NewLimitReader(stream func() ([]byte, error), limit int64) *LimitReader {
	return &LimitReader{
		stream:         stream,
		limitRemaining: limit,
	}
}

func (r *LimitReader) Read(p []byte) (n int, err error) {
	// If we don't have any bytes at all, get some from the stream.
	if len(r.pendingChunkData) == 0 {
		newChunkData, err := r.stream()
		if err != nil {
			// Return the error whether it's EOF or something else.
			return 0, err
		}

		if rErr := r.recordNewData(newChunkData); rErr != nil {
			return 0, rErr
		}

		if (len(r.pendingChunkData) == 0) && (err == io.EOF) {
			// If we got no data and an EOF, pass the EOF back up.
			return 0, io.EOF
		}

	}

	// It would be possible for this function to try to fill more of the caller's buffer if the
	// first Recv did not get as much data as the caller wanted.  However, it's simpler to not do that.

	copied := copy(p, r.pendingChunkData)
	if copied > 0 {
		r.pendingChunkData = r.pendingChunkData[copied:]
	}

	return copied, nil
}

func (r *LimitReader) recordNewData(newBytes []byte) error {
	newLen := int64(len(newBytes))
	if newLen > r.limitRemaining {
		return errors.New("exceeded file size limit", errors.WithErrorCode(errors.ETooLarge))
	}

	// Within limit, so accept the new bytes.
	r.pendingChunkData = append(r.pendingChunkData, newBytes...)
	r.limitRemaining -= newLen

	return nil
}
