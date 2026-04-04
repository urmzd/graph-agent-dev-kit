// Package eval provides agent-specific evaluation scorers and utilities.
package eval

import (
	"sort"
	"strings"
	"time"

	"github.com/urmzd/saige/agent/types"
)

// StreamTiming holds latency metrics collected from a delta stream.
type StreamTiming struct {
	TTFTMs       int64   `json:"ttft_ms"`
	TTLTMs       int64   `json:"ttlt_ms"`
	MedianITL    float64 `json:"median_itl_ms"`
	ChunkCount   int     `json:"chunk_count"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
}

// CollectStreamTiming drains a delta channel, collecting timing data and
// concatenating text content. Returns the collected timing, the full text
// output, and all deltas (for further inspection by scorers).
func CollectStreamTiming(ch <-chan types.Delta) (StreamTiming, string, []types.Delta) {
	start := time.Now()
	var timing StreamTiming
	var firstTokenAt time.Time
	var lastTokenAt time.Time
	var itls []time.Duration
	var prevChunkAt time.Time
	var textBuf strings.Builder
	var allDeltas []types.Delta

	for delta := range ch {
		now := time.Now()
		allDeltas = append(allDeltas, delta)

		switch v := delta.(type) {
		case types.TextContentDelta:
			if firstTokenAt.IsZero() {
				firstTokenAt = now
			}
			if !prevChunkAt.IsZero() {
				itls = append(itls, now.Sub(prevChunkAt))
			}
			prevChunkAt = now
			lastTokenAt = now
			timing.ChunkCount++
			textBuf.WriteString(v.Content)

		case types.UsageDelta:
			timing.InputTokens += v.PromptTokens
			timing.OutputTokens += v.CompletionTokens
		}
	}

	if !firstTokenAt.IsZero() {
		timing.TTFTMs = firstTokenAt.Sub(start).Milliseconds()
	}
	if !lastTokenAt.IsZero() {
		timing.TTLTMs = lastTokenAt.Sub(start).Milliseconds()
	}
	if len(itls) > 0 {
		sort.Slice(itls, func(i, j int) bool { return itls[i] < itls[j] })
		mid := len(itls) / 2
		if len(itls)%2 == 0 {
			timing.MedianITL = float64(itls[mid-1]+itls[mid]) / 2.0 / float64(time.Millisecond)
		} else {
			timing.MedianITL = float64(itls[mid]) / float64(time.Millisecond)
		}
	}

	return timing, textBuf.String(), allDeltas
}
