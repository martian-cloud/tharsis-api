package logstream

//go:generate go tool mockery --name Store --inpackage --case underscore

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// Store interface encapsulates the object-store IO for a single log chunk object.
// It is intentionally free of any database access; chunk metadata and transaction
// boundaries are owned by the Manager.
type Store interface {
	// WriteChunk writes buffer into the object identified by key at byteOffset, truncating the
	// object to byteOffset+len(buffer). The object's existing content up to byteOffset is preserved.
	// This is bounded by the stream's max chunk size, so the download-modify-upload cost stays small.
	WriteChunk(ctx context.Context, key string, byteOffset int, buffer []byte) error
	// ReadRange returns a reader over up to length bytes starting at offset within the object
	// identified by key. The data is streamed directly from object storage: it is never buffered
	// in full nor written to local disk. The caller must Close the returned reader.
	ReadRange(ctx context.Context, key string, offset int, length int) (io.ReadCloser, error)
	// WriteObject writes the full contents of r to the object identified by key, replacing it. Used
	// by compaction to stream all of a stream's chunks into a single consolidated object.
	WriteObject(ctx context.Context, key string, r io.Reader) error
}

type store struct {
	objectStore objectstore.ObjectStore
}

// NewLogStore creates an instance of the Store interface
func NewLogStore(objectStore objectstore.ObjectStore) Store {
	return &store{objectStore: objectStore}
}

// WriteChunk writes buffer into the chunk object at byteOffset.
//
// byteOffset is always the chunk's committed size (the Manager appends at the chunk's current end),
// so it is server-derived, not client-supplied. Object storage is written before the metadata commit,
// so a committed chunk size is always backed by at least that many object bytes. If the existing
// object is therefore missing or shorter than byteOffset, the object store has lost data relative to
// the database — an internal inconsistency (EInternal), not a client gap.
func (ls *store) WriteChunk(ctx context.Context, key string, byteOffset int, buffer []byte) error {
	if byteOffset < 0 {
		return errors.New("offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	// Object stores have no in-place append, so writing a chunk means re-uploading the whole object as
	// the preserved prefix [0, byteOffset) followed by buffer. We read that prefix into memory (bounded
	// by the chunk fill size) and fully close the read stream before uploading, instead of
	// staging it on local disk.
	var prefix []byte
	if byteOffset > 0 {
		contentRange := fmt.Sprintf("bytes=0-%d", byteOffset-1)
		out, err := ls.objectStore.GetObjectStream(ctx, key, &objectstore.DownloadOptions{ContentRange: &contentRange})
		if err != nil {
			if errors.ErrorCode(err) == errors.ENotFound {
				// The object is missing or shorter than its committed size: object store data loss.
				return errors.New("log chunk object %s is missing bytes before offset %d (object store data loss)", key, byteOffset, errors.WithErrorCode(errors.EInternal))
			}
			return errors.Wrap(err, "Failed to read existing log chunk from object storage")
		}

		prefix, err = io.ReadAll(out.Body)
		closeErr := out.Body.Close()
		if err != nil {
			return errors.Wrap(err, "Failed to read existing log chunk from object storage")
		}
		if closeErr != nil {
			return errors.Wrap(closeErr, "Failed to close existing log chunk reader")
		}

		// The object held fewer bytes than its committed size: object store data loss.
		if len(prefix) != byteOffset {
			return errors.New("log chunk object %s is shorter (%d bytes) than its committed size %d (object store data loss)", key, len(prefix), byteOffset, errors.WithErrorCode(errors.EInternal))
		}
	}

	body := io.MultiReader(bytes.NewReader(prefix), bytes.NewReader(buffer))
	if err := ls.objectStore.UploadObject(ctx, key, body); err != nil {
		return errors.Wrap(err, "Failed to upload log chunk to object storage")
	}

	return nil
}

// WriteObject writes the full contents of r into the object identified by key, replacing any
// existing object. Used by compaction to stream a stream's chunks into a single consolidated object.
// The reader is passed straight through to the object store: compaction runs single-flight per
// instance, so the multipart uploader's bounded part buffering is the only memory cost.
func (ls *store) WriteObject(ctx context.Context, key string, r io.Reader) error {
	if err := ls.objectStore.UploadObject(ctx, key, r); err != nil {
		return errors.Wrap(err, "Failed to upload log object to object storage")
	}
	return nil
}

// ReadRange returns a reader that streams up to length bytes from the chunk object starting at
// offset, without buffering the data in full or using disk. The caller must Close the reader.
func (ls *store) ReadRange(ctx context.Context, key string, offset int, length int) (io.ReadCloser, error) {
	if offset < 0 || length < 0 {
		return nil, errors.New("offset and length cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	if length == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}

	// Object-store byte ranges are inclusive on both ends.
	contentRange := fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)

	out, err := ls.objectStore.GetObjectStream(ctx, key, &objectstore.DownloadOptions{ContentRange: &contentRange})
	if err != nil {
		return nil, err
	}

	return out.Body, nil
}
