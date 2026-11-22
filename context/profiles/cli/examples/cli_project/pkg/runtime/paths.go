package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example.com/cliproj/pkg/config"
)

// Paths groups the four working-directory categories of one invocation,
// distinguished by lifetime and visibility:
//
//   - Ephemeral: temp-scoped, removed on exit (alt-screen captures)
//   - Retained: run-scoped, kept after exit for inspection (logs, side files)
//   - UserArtifacts: user-visible outputs the caller is expected to consume
//   - CrossSession: stable cache shared across invocations
//
// All four root under the application name; the first three also under the
// session ID. Only CrossSession exists after ResolvePaths: the session-scoped
// directories are unique per invocation, and creating them eagerly would
// leak one empty directory per run, so their consumers create them on first
// use (EnsureEphemeral, output.WithRetainedDir).
type Paths struct {
	Ephemeral     string
	Retained      string
	UserArtifacts string
	CrossSession  string
}

// EnsureEphemeral creates the Ephemeral directory on demand and returns it.
// It is idempotent; an empty Ephemeral is a no-op returning "".
func (p Paths) EnsureEphemeral() (string, error) {
	if p.Ephemeral == "" {
		return "", nil
	}
	if err := os.MkdirAll(p.Ephemeral, 0o755); err != nil {
		return "", fmt.Errorf("create ephemeral dir %s: %w", p.Ephemeral, err)
	}
	return p.Ephemeral, nil
}

// ResolvePaths computes the four categories from platform defaults,
// overridden by explicit settings: TempDir overrides the Ephemeral root,
// CacheDir the Retained and CrossSession root, OutputDir the UserArtifacts
// root.
func ResolvePaths(g config.Global) (Paths, error) {
	tempRoot := g.TempDir
	if tempRoot == "" {
		tempRoot = os.TempDir()
	}
	cacheRoot := g.CacheDir
	if cacheRoot == "" {
		if d, err := os.UserCacheDir(); err == nil {
			cacheRoot = d
		} else {
			cacheRoot = tempRoot
		}
	}
	outputRoot := g.OutputDir
	if outputRoot == "" {
		if d, err := os.UserConfigDir(); err == nil {
			outputRoot = d
		} else {
			outputRoot = cacheRoot
		}
	}
	p := Paths{
		Ephemeral:     filepath.Join(tempRoot, g.AppName, g.SessionID),
		Retained:      filepath.Join(cacheRoot, g.AppName, g.SessionID),
		UserArtifacts: filepath.Join(outputRoot, g.AppName, g.SessionID),
		CrossSession:  filepath.Join(cacheRoot, g.AppName),
	}
	if err := os.MkdirAll(p.CrossSession, 0o755); err != nil {
		return Paths{}, fmt.Errorf("create cache dir %s: %w", p.CrossSession, err)
	}
	return p, nil
}

// GenerateSessionID returns an 8-character hex identifier from crypto/rand:
// uniform and not sequentially guessable.
func GenerateSessionID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", uint32(time.Now().UnixNano())^uint32(os.Getpid()))
	}
	return hex.EncodeToString(b)
}
