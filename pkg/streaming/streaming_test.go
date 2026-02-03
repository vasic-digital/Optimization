package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamBuffer_FlushOnWord(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []string
		want    [][]string
		wantEnd string
	}{
		{
			name:   "single word with space",
			inputs: []string{"hello "},
			want:   [][]string{{"hello "}},
		},
		{
			name:    "word without space stays buffered",
			inputs:  []string{"hello"},
			want:    [][]string{nil},
			wantEnd: "hello",
		},
		{
			name:   "multiple words",
			inputs: []string{"hello world "},
			want:   [][]string{{"hello ", "world "}},
		},
		{
			name:    "incremental input",
			inputs:  []string{"hel", "lo ", "wor", "ld"},
			want:    [][]string{nil, {"hello "}, nil, nil},
			wantEnd: "world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewStreamBuffer(FlushOnWord, 0)

			for i, input := range tt.inputs {
				result := buf.Add(input)
				if i < len(tt.want) {
					if tt.want[i] == nil {
						assert.Nil(t, result)
					} else {
						assert.Equal(t, tt.want[i], result)
					}
				}
			}

			if tt.wantEnd != "" {
				assert.Equal(t, tt.wantEnd, buf.Flush())
			}
		})
	}
}

func TestStreamBuffer_FlushOnSentence(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []string
		want    [][]string
		wantEnd string
	}{
		{
			name:   "complete sentence",
			inputs: []string{"Hello world. "},
			want:   [][]string{{"Hello world."}},
		},
		{
			name:    "incomplete sentence",
			inputs:  []string{"Hello world"},
			want:    [][]string{nil},
			wantEnd: "Hello world",
		},
		{
			name:   "sentence with exclamation",
			inputs: []string{"Wow! Great."},
			want:   [][]string{{"Wow!", "Great."}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewStreamBuffer(FlushOnSentence, 0)

			for i, input := range tt.inputs {
				result := buf.Add(input)
				if i < len(tt.want) {
					if tt.want[i] == nil {
						assert.Nil(t, result)
					} else {
						assert.Equal(t, tt.want[i], result)
					}
				}
			}

			if tt.wantEnd != "" {
				assert.Equal(t, tt.wantEnd, buf.Flush())
			}
		})
	}
}

func TestStreamBuffer_FlushOnLine(t *testing.T) {
	buf := NewStreamBuffer(FlushOnLine, 0)

	result := buf.Add("first line\n")
	assert.Equal(t, []string{"first line\n"}, result)

	result = buf.Add("partial")
	assert.Nil(t, result)

	assert.Equal(t, "partial", buf.Flush())
}

func TestStreamBuffer_FlushOnSize(t *testing.T) {
	buf := NewStreamBuffer(FlushOnSize, 3)

	result := buf.Add("one two")
	assert.Nil(t, result)

	result = buf.Add(" three four")
	assert.Equal(t, []string{"one two three four"}, result)
}

func TestStreamBuffer_Reset(t *testing.T) {
	buf := NewStreamBuffer(FlushOnWord, 0)
	buf.Add("hello")
	buf.Reset()
	assert.Equal(t, "", buf.Flush())
}

func TestStreamBuffer_ImplementsBuffer(t *testing.T) {
	var _ Buffer = (*StreamBuffer)(nil)
}

func TestTokenCounter_Count(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		ratio float64
		want  int
	}{
		{name: "empty", text: "", ratio: 1.3, want: 0},
		{name: "single word", text: "hello", ratio: 1.0, want: 1},
		{name: "multiple words", text: "hello world foo", ratio: 1.0, want: 3},
		{
			name:  "with default ratio",
			text:  "hello world foo",
			ratio: 1.3,
			want:  3, // 3 * 1.3 = 3.9, truncated to 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewTokenCounterWithRatio(tt.ratio)
			assert.Equal(t, tt.want, counter.Count(tt.text))
		})
	}
}

func TestTokenCounter_CountWords(t *testing.T) {
	counter := NewTokenCounter()
	assert.Equal(t, 0, counter.CountWords(""))
	assert.Equal(t, 3, counter.CountWords("hello world foo"))
	assert.Equal(t, 2, counter.CountWords("  hello   world  "))
}

func TestTokenCounter_CountCharacters(t *testing.T) {
	counter := NewTokenCounter()
	assert.Equal(t, 0, counter.CountCharacters(""))
	assert.Equal(t, 5, counter.CountCharacters("hello"))
}

func TestTokenCounter_Fits(t *testing.T) {
	counter := NewTokenCounterWithRatio(1.0)
	assert.True(t, counter.Fits("hello world", 5))
	assert.True(t, counter.Fits("hello world", 2))
	assert.False(t, counter.Fits("hello world foo", 2))
}

func TestChunkMerger_Merge(t *testing.T) {
	tests := []struct {
		name    string
		min     int
		inputs  []string
		want    []string
		wantEnd string
	}{
		{
			name:   "single large chunk passes through",
			min:    2,
			inputs: []string{"hello world foo"},
			want:   []string{"hello world foo"},
		},
		{
			name:    "small chunks get merged",
			min:     3,
			inputs:  []string{"hi", " there", " friend"},
			want:    []string{"", "", "hi there friend"},
			wantEnd: "",
		},
		{
			name:    "partial accumulation",
			min:     5,
			inputs:  []string{"one ", "two "},
			want:    []string{"", ""},
			wantEnd: "one two ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merger := NewChunkMerger(tt.min)

			for i, input := range tt.inputs {
				result := merger.Add(input)
				if i < len(tt.want) {
					assert.Equal(t, tt.want[i], result)
				}
			}

			if tt.wantEnd != "" {
				assert.Equal(t, tt.wantEnd, merger.Flush())
			}
		})
	}
}

func TestChunkMerger_Reset(t *testing.T) {
	merger := NewChunkMerger(5)
	merger.Add("hello")
	merger.Reset()
	assert.Equal(t, "", merger.Flush())
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 5, cfg.BufferSize)
	assert.Equal(t, 3, cfg.MinChunkSize)
	assert.Equal(t, FlushOnWord, cfg.Strategy)
}

// Tests for NewStreamBuffer default threshold.

func TestNewStreamBuffer_DefaultThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		expected  int
	}{
		{"zero threshold defaults to 5", 0, 5},
		{"negative threshold defaults to 5", -10, 5},
		{"positive threshold is kept", 10, 10},
		{"threshold of 1 is kept", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewStreamBuffer(FlushOnSize, tt.threshold)
			assert.Equal(t, tt.expected, buf.threshold)
		})
	}
}

// Tests for StreamBuffer with unknown strategy (default case).

func TestStreamBuffer_UnknownStrategy(t *testing.T) {
	// Using an undefined strategy should fall back to word flushing.
	buf := NewStreamBuffer(FlushStrategy("unknown"), 5)

	result := buf.Add("hello world ")
	// Should behave like FlushOnWord.
	assert.Equal(t, []string{"hello ", "world "}, result)
}

// Tests for NewChunkMerger default minChunkSize.

func TestNewChunkMerger_DefaultMinChunkSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expected int
	}{
		{"zero defaults to 3", 0, 3},
		{"negative defaults to 3", -5, 3},
		{"positive is kept", 10, 10},
		{"one is kept", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merger := NewChunkMerger(tt.size)
			assert.Equal(t, tt.expected, merger.minChunkSize)
		})
	}
}

// Tests for NewTokenCounterWithRatio default ratio.

func TestNewTokenCounterWithRatio_DefaultRatio(t *testing.T) {
	tests := []struct {
		name     string
		ratio    float64
		expected float64
	}{
		{"zero defaults to 1.3", 0, 1.3},
		{"negative defaults to 1.3", -1.0, 1.3},
		{"positive is kept", 2.0, 2.0},
		{"small positive is kept", 0.5, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewTokenCounterWithRatio(tt.ratio)
			assert.Equal(t, tt.expected, counter.TokensPerWord)
		})
	}
}

// Additional tests for StreamBuffer edge cases.

func TestStreamBuffer_FlushOnSentence_MultipleSentenceEnds(t *testing.T) {
	buf := NewStreamBuffer(FlushOnSentence, 5)

	// Test with question mark.
	result := buf.Add("What? Really! Yes. ")
	assert.Equal(t, []string{"What?", "Really!", "Yes."}, result)
}

func TestStreamBuffer_FlushOnSize_ExactThreshold(t *testing.T) {
	buf := NewStreamBuffer(FlushOnSize, 3)

	// Add exactly 3 words.
	result := buf.Add("one two three")
	assert.Equal(t, []string{"one two three"}, result)

	// Buffer should be empty now.
	assert.Equal(t, "", buf.Flush())
}

func TestStreamBuffer_FlushOnLine_MultipleLines(t *testing.T) {
	buf := NewStreamBuffer(FlushOnLine, 5)

	result := buf.Add("line1\nline2\nline3\n")
	assert.Equal(t, []string{"line1\n", "line2\n", "line3\n"}, result)
}

// Tests for ChunkMerger edge cases.

func TestChunkMerger_FlushEmpty(t *testing.T) {
	merger := NewChunkMerger(5)

	// Flush without adding anything.
	result := merger.Flush()
	assert.Equal(t, "", result)
}

func TestChunkMerger_AddEmptyString(t *testing.T) {
	merger := NewChunkMerger(3)

	result := merger.Add("")
	assert.Equal(t, "", result)

	// Add some content.
	result = merger.Add("one two three")
	assert.Equal(t, "one two three", result)
}

// Tests for TokenCounter edge cases.

func TestTokenCounter_CountWithCustomRatio(t *testing.T) {
	counter := NewTokenCounterWithRatio(2.0)

	// 3 words * 2.0 = 6 tokens.
	assert.Equal(t, 6, counter.Count("hello world foo"))

	// Empty string should return 0.
	assert.Equal(t, 0, counter.Count(""))
}

func TestTokenCounter_CountCharacters_Unicode(t *testing.T) {
	counter := NewTokenCounter()

	// Unicode characters should be counted correctly.
	assert.Equal(t, 5, counter.CountCharacters("Hello"))

	// Korean characters (each is one rune).
	assert.Equal(t, 2, counter.CountCharacters("\ud55c\uae00"))

	// Emoji (each emoji is typically one or more runes).
	// Simple emoji.
	text := "\U0001F600" // Grinning face.
	assert.Equal(t, 1, counter.CountCharacters(text))
}
