// Package streaming provides streaming optimizations for LLM responses
// including configurable buffers, token counting, and chunk merging.
package streaming

import (
	"strings"
	"unicode"
)

// Buffer defines the interface for streaming buffers.
type Buffer interface {
	// Add adds text to the buffer and returns flushed content.
	Add(text string) []string
	// Flush returns any remaining content in the buffer.
	Flush() string
	// Reset clears the buffer.
	Reset()
}

// FlushStrategy defines how the buffer decides when to flush.
type FlushStrategy string

const (
	// FlushOnWord flushes on word boundaries.
	FlushOnWord FlushStrategy = "word"
	// FlushOnSentence flushes on sentence boundaries.
	FlushOnSentence FlushStrategy = "sentence"
	// FlushOnLine flushes on newline boundaries.
	FlushOnLine FlushStrategy = "line"
	// FlushOnSize flushes when buffer reaches a size threshold.
	FlushOnSize FlushStrategy = "size"
)

// StreamBuffer implements a configurable streaming buffer.
type StreamBuffer struct {
	buffer    strings.Builder
	strategy  FlushStrategy
	threshold int
}

// NewStreamBuffer creates a new stream buffer with the given strategy.
func NewStreamBuffer(strategy FlushStrategy, threshold int) *StreamBuffer {
	if threshold <= 0 {
		threshold = 5
	}
	return &StreamBuffer{
		strategy:  strategy,
		threshold: threshold,
	}
}

// Add adds text to the buffer and returns content according to the strategy.
func (b *StreamBuffer) Add(text string) []string {
	b.buffer.WriteString(text)

	switch b.strategy {
	case FlushOnWord:
		return b.flushWords()
	case FlushOnSentence:
		return b.flushSentences()
	case FlushOnLine:
		return b.flushLines()
	case FlushOnSize:
		return b.flushOnSize()
	default:
		return b.flushWords()
	}
}

// Flush returns any remaining content in the buffer.
func (b *StreamBuffer) Flush() string {
	remaining := b.buffer.String()
	b.buffer.Reset()
	return remaining
}

// Reset clears the buffer.
func (b *StreamBuffer) Reset() {
	b.buffer.Reset()
}

func (b *StreamBuffer) flushWords() []string {
	content := b.buffer.String()
	var result []string

	for {
		idx := strings.Index(content, " ")
		if idx < 0 {
			break
		}
		word := content[:idx+1]
		result = append(result, word)
		content = content[idx+1:]
	}

	b.buffer.Reset()
	b.buffer.WriteString(content)
	return result
}

func (b *StreamBuffer) flushSentences() []string {
	content := b.buffer.String()
	var result []string

	for {
		idx := findSentenceEnd(content)
		if idx < 0 {
			break
		}
		sentence := content[:idx+1]
		result = append(result, sentence)
		content = strings.TrimLeftFunc(content[idx+1:], unicode.IsSpace)
	}

	b.buffer.Reset()
	b.buffer.WriteString(content)
	return result
}

func findSentenceEnd(s string) int {
	endings := map[rune]bool{'.': true, '!': true, '?': true}
	for i, r := range s {
		if endings[r] {
			if i == len(s)-1 || unicode.IsSpace(rune(s[i+1])) {
				return i
			}
		}
	}
	return -1
}

func (b *StreamBuffer) flushLines() []string {
	content := b.buffer.String()
	var result []string

	for {
		idx := strings.Index(content, "\n")
		if idx < 0 {
			break
		}
		line := content[:idx+1]
		result = append(result, line)
		content = content[idx+1:]
	}

	b.buffer.Reset()
	b.buffer.WriteString(content)
	return result
}

func (b *StreamBuffer) flushOnSize() []string {
	content := b.buffer.String()
	words := strings.Fields(content)

	if len(words) < b.threshold {
		return nil
	}

	result := []string{content}
	b.buffer.Reset()
	return result
}
