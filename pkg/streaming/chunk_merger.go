package streaming

import (
	"strings"
)

// ChunkMerger combines small stream chunks into larger ones
// to reduce overhead from processing many small pieces.
type ChunkMerger struct {
	buffer       strings.Builder
	minChunkSize int
	tokenCounter *TokenCounter
}

// NewChunkMerger creates a new chunk merger.
func NewChunkMerger(minChunkSize int) *ChunkMerger {
	if minChunkSize <= 0 {
		minChunkSize = 3
	}
	return &ChunkMerger{
		minChunkSize: minChunkSize,
		tokenCounter: NewTokenCounter(),
	}
}

// Add adds a chunk and returns merged content if minimum size is reached.
// Returns empty string if still accumulating.
func (m *ChunkMerger) Add(chunk string) string {
	m.buffer.WriteString(chunk)

	content := m.buffer.String()
	wordCount := m.tokenCounter.CountWords(content)

	if wordCount >= m.minChunkSize {
		m.buffer.Reset()
		return content
	}

	return ""
}

// Flush returns any remaining buffered content.
func (m *ChunkMerger) Flush() string {
	content := m.buffer.String()
	m.buffer.Reset()
	return content
}

// Reset clears the merger state.
func (m *ChunkMerger) Reset() {
	m.buffer.Reset()
}
