package runtime

import (
	"context"
	"io"
	"os"
	"time"
)

// WarmFile reads path sequentially so its pages are in the OS page cache when
// llama-server maps the same file. After a reboot the first load otherwise pays for the
// whole model from disk while the health deadline runs. It stops at ctx cancellation,
// skips files that would not fit in available memory, and returns the bytes read.
func WarmFile(ctx context.Context, path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if avail, err := memAvailable(); err == nil && st.Size() > avail*8/10 {
		return 0, nil // would evict more than it saves
	}
	buf := make([]byte, 4<<20)
	var total int64
	for {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		n, err := f.Read(buf)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		// Yield briefly so warming never starves the process that needs the data now.
		time.Sleep(time.Millisecond)
	}
}
