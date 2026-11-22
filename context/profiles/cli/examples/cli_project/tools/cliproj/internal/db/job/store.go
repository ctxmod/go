// Package jobdb persists cliproj jobs as keyed records — one JSON file per
// job, keyed `job/<id>`, last write wins. The key IS the job's identity;
// listing is the `job/` prefix, not a query language. Writes are atomic
// (temp file, then rename), so a reader never sees a half-written record.
package jobdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/cliproj/pkg/runtime"
)

// ErrNotFound reports a missing job; match it with errors.Is.
var ErrNotFound = errors.New("jobdb: job not found")

// Job is one persisted job record; the JSON shape is the record payload.
type Job struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists jobs under one root directory. Re-opening the same root is
// safe: records are atomic files and the newest write wins.
type Store struct {
	root string
}

// Open opens (creating if needed) the job store under the runtime's
// Retained directory.
func Open(rt *runtime.Runtime) (*Store, error) {
	root := filepath.Join(rt.Paths().Retained, "jobs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("jobdb: open store at %s: %w", root, err)
	}
	return &Store{root: root}, nil
}

// Put writes one job. The id is the key, so re-putting an existing id
// supersedes the previous record in place.
func (s *Store) Put(ctx context.Context, job Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validID(job.ID); err != nil {
		return fmt.Errorf("jobdb: put: %w", err)
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = time.Now().UTC()
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("jobdb: put job %s: %w", job.ID, err)
	}
	tmp, err := os.CreateTemp(s.root, ".put-*")
	if err != nil {
		return fmt.Errorf("jobdb: put job %s: %w", job.ID, err)
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return fmt.Errorf("jobdb: put job %s: %w", job.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("jobdb: put job %s: %w", job.ID, err)
	}
	if err := os.Rename(tmp.Name(), s.path(job.ID)); err != nil {
		return fmt.Errorf("jobdb: put job %s: %w", job.ID, err)
	}
	return nil
}

// Get reads one job by id. A missing id reads as ErrNotFound.
func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	if err := validID(id); err != nil {
		return Job{}, fmt.Errorf("jobdb: get: %w", err)
	}
	payload, err := os.ReadFile(s.path(id))
	if errors.Is(err, fs.ErrNotExist) {
		return Job{}, fmt.Errorf("jobdb: get job %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Job{}, fmt.Errorf("jobdb: get job %s: %w", id, err)
	}
	var job Job
	if err := json.Unmarshal(payload, &job); err != nil {
		return Job{}, fmt.Errorf("jobdb: decode job %s: %w", id, err)
	}
	return job, nil
}

// List returns every stored job, ordered by key.
func (s *Store) List(ctx context.Context) ([]Job, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("jobdb: list jobs: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if name := e.Name(); strings.HasSuffix(name, ".json") && !e.IsDir() {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(ids)
	jobs := make([]Job, 0, len(ids))
	for _, id := range ids {
		job, err := s.Get(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue // deleted between ReadDir and Get
		}
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Delete removes one job by id; deleting a missing id is not an error.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validID(id); err != nil {
		return fmt.Errorf("jobdb: delete: %w", err)
	}
	if err := os.Remove(s.path(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("jobdb: delete job %s: %w", id, err)
	}
	return nil
}

// Close releases the store; the file-per-record layout holds no handles.
func (s *Store) Close() error { return nil }

func (s *Store) path(id string) string { return filepath.Join(s.root, id+".json") }

// validID keeps the key inside the store root.
func validID(id string) error {
	if id == "" {
		return errors.New("job id is empty (the id is the record key)")
	}
	if strings.ContainsAny(id, `/\`) || id == "." || id == ".." || strings.HasPrefix(id, ".") {
		return fmt.Errorf("job id %q is not a plain name", id)
	}
	return nil
}
