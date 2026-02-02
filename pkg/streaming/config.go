package streaming

import (
	"time"
)

// Config holds streaming optimization configuration.
type Config struct {
	// BufferSize is the buffer size in words for size-based flushing.
	BufferSize int `json:"buffer_size"`
	// FlushInterval is the maximum time between flushes.
	FlushInterval time.Duration `json:"flush_interval"`
	// MinChunkSize is the minimum chunk size for merging.
	MinChunkSize int `json:"min_chunk_size"`
	// Strategy is the flush strategy for the stream buffer.
	Strategy FlushStrategy `json:"strategy"`
}

// DefaultConfig returns a default streaming configuration.
func DefaultConfig() *Config {
	return &Config{
		BufferSize:    5,
		FlushInterval: 100 * time.Millisecond,
		MinChunkSize:  3,
		Strategy:      FlushOnWord,
	}
}
