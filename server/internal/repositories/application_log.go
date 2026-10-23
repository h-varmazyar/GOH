package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/h-varmazyar/gatec/pkg/paginator"
)

const logReadBlockSize = 64 * 1024

type ApplicationLogRepository struct {
	path string
}

func NewApplicationLogRepository(path string) *ApplicationLogRepository {
	return &ApplicationLogRepository{path: path}
}

func (r *ApplicationLogRepository) List(
	ctx context.Context,
	component string,
	pg paginator.Params,
) ([]map[string]any, int64, error) {
	file, err := os.Open(r.path)
	if err != nil {
		return nil, 0, fmt.Errorf("open application log: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("stat application log: %w", err)
	}

	items := make([]map[string]any, 0, pg.Limit())
	var total int64
	err = readLinesReverse(ctx, file, info.Size(), func(line []byte) error {
		entry := make(map[string]any)
		if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &entry) != nil || !logEntryMatches(entry, component) {
			return nil
		}

		index := total
		total++
		if index >= int64(pg.Offset()) && len(items) < pg.Limit() {
			items = append(items, entry)
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("read application log: %w", err)
	}
	return items, total, nil
}

func readLinesReverse(
	ctx context.Context,
	file *os.File,
	size int64,
	visit func([]byte) error,
) error {
	position := size
	var carry []byte

	for position > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		blockSize := int64(logReadBlockSize)
		if position < blockSize {
			blockSize = position
		}
		position -= blockSize

		block := make([]byte, int(blockSize))
		read, err := file.ReadAt(block, position)
		if err != nil && err != io.EOF {
			return err
		}
		data := append(block[:read], carry...)
		carry = nil

		if position > 0 {
			separator := bytes.IndexByte(data, '\n')
			if separator < 0 {
				carry = data
				continue
			}
			carry = append(carry, data[:separator]...)
			data = data[separator+1:]
		}

		lines := bytes.Split(data, []byte{'\n'})
		for i := len(lines) - 1; i >= 0; i-- {
			if err := visit(lines[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func logEntryMatches(entry map[string]any, component string) bool {
	for _, field := range [...]string{"component", "worker", "adapter"} {
		if value, ok := entry[field].(string); ok && value == component {
			return true
		}
	}
	return false
}
