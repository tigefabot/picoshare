package file

import (
	"database/sql"
	"io"

	"github.com/mtlynch/picoshare/v2/types"
)

type (
	sqlTx interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
	}

	writer struct {
		tx      sqlTx
		entryID types.EntryID
		buf     []byte
		written int
	}
)

func NewWriter(tx sqlTx, id types.EntryID, chunkSize int) io.WriteCloser {
	return &writer{
		tx:      tx,
		entryID: id,
		buf:     make([]byte, chunkSize),
	}
}

func (w *writer) Write(p []byte) (int, error) {
	n := 0

	for {
		if n == len(p) {
			break
		}
		dstStart := w.written % len(w.buf)
		copySize := min(len(w.buf)-dstStart, len(p)-n)
		dstEnd := dstStart + copySize
		copy(w.buf[dstStart:dstEnd], p[n:n+copySize])
		if dstEnd == len(w.buf) {
			w.flush(len(w.buf))
		}
		w.written += copySize
		n += copySize
	}

	return n, nil
}

func (w *writer) Close() error {
	unflushed := w.written % len(w.buf)
	if unflushed != 0 {
		return w.flush(unflushed)
	}
	return nil
}

func (w *writer) flush(n int) error {
	idx := w.written / len(w.buf)
	_, err := w.tx.Exec(`
	INSERT INTO
		entries_data
	(
		id,
		chunk_index,
		chunk
	)
	VALUES(?,?,?)`, w.entryID, idx, w.buf[0:n])
	return err
}
