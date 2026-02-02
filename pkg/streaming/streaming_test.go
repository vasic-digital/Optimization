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
