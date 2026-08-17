package houses

import (
	"errors"
	"testing"
	"time"
)

func TestMemoryRepository(t *testing.T) {
	repository := NewMemoryRepository()
	if initial := repository.Snapshot(); initial.Houses == nil || initial.Count != 0 {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}
	attempt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	success := attempt.Add(time.Second)
	next := attempt.Add(30 * time.Minute)
	repository.RecordAttempt(attempt)
	repository.RecordFailure(errors.New("temporary"))
	repository.Replace([]House{{Key: "749"}}, success)
	repository.SetNextRun(&next)

	snapshot := repository.Snapshot()
	if snapshot.Count != 1 || snapshot.Houses[0].Key != "749" || snapshot.LastError != "" ||
		snapshot.LastAttempt == nil || !snapshot.LastAttempt.Equal(attempt) ||
		snapshot.LastSuccess == nil || !snapshot.LastSuccess.Equal(success) ||
		snapshot.NextRun == nil || !snapshot.NextRun.Equal(next) {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	snapshot.Houses[0].Key = "changed"
	if repository.Snapshot().Houses[0].Key != "749" {
		t.Fatal("Snapshot returned mutable repository storage")
	}
	repository.SetNextRun(nil)
	if repository.Snapshot().NextRun != nil {
		t.Fatal("SetNextRun(nil) did not clear next run")
	}
}
