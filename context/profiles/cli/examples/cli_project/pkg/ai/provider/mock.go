// Package provider contains test doubles for provider.LanguageModel and
// related provider types defined by goai.
//
// The package name collides with github.com/zendev-sh/goai/provider; files
// in this package import the goai package under the goaiprovider alias to
// keep call sites unambiguous.
package provider

import (
	"context"
	"sync"
	"sync/atomic"

	goaiprovider "github.com/zendev-sh/goai/provider"
)

// MockModel implements goaiprovider.LanguageModel for testing. It records
// call counts via atomic counters so tests may assert invocation patterns
// under concurrent access.
type MockModel struct {
	// TextResponse is the text returned from DoGenerate and streamed from
	// DoStream as a single ChunkText followed by ChunkFinish.
	TextResponse string

	// Err, when non-nil, is returned from DoGenerate and DoStream instead
	// of a successful response.
	Err error

	// ModelIDValue is returned from ModelID.
	ModelIDValue string

	// GenerateCount counts calls to DoGenerate.
	GenerateCount atomic.Int64

	// StreamCount counts calls to DoStream.
	StreamCount atomic.Int64

	// mu protects LastParams from concurrent writes.
	mu sync.Mutex

	// LastParams records the most recent GenerateParams passed to
	// DoGenerate. Safe to read only after all concurrent calls have
	// completed (e.g. after WaitGroup.Wait).
	LastParams goaiprovider.GenerateParams
}

// ModelID returns the configured model identifier.
func (m *MockModel) ModelID() string { return m.ModelIDValue }

// DoGenerate records the call, captures params for later inspection, then
// returns the configured text or error.
func (m *MockModel) DoGenerate(ctx context.Context, params goaiprovider.GenerateParams) (*goaiprovider.GenerateResult, error) {
	m.GenerateCount.Add(1)
	m.mu.Lock()
	m.LastParams = params
	m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	return &goaiprovider.GenerateResult{Text: m.TextResponse}, nil
}

// DoStream records the call, then emits a single text chunk followed by a
// finish chunk. If Err is set, it is returned before any channel is opened.
func (m *MockModel) DoStream(ctx context.Context, params goaiprovider.GenerateParams) (*goaiprovider.StreamResult, error) {
	m.StreamCount.Add(1)
	if m.Err != nil {
		return nil, m.Err
	}
	ch := make(chan goaiprovider.StreamChunk, 2)
	ch <- goaiprovider.StreamChunk{Type: goaiprovider.ChunkText, Text: m.TextResponse}
	ch <- goaiprovider.StreamChunk{Type: goaiprovider.ChunkFinish}
	close(ch)
	return &goaiprovider.StreamResult{Stream: ch}, nil
}
