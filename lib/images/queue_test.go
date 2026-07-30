package images

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnqueueWhileActiveQueuesBehindInFlightBuild covers the window between a
// build's terminal signal and its queue-slot release: a fresh request for the
// same image landing in that window must queue behind the in-flight build, not
// be dropped. Dropping it leaves recreated images pending forever.
func TestEnqueueWhileActiveQueuesBehindInFlightBuild(t *testing.T) {
	q := NewBuildQueue(1)

	started := make(chan struct{})
	release := make(chan struct{})

	pos := q.Enqueue("sha256:abc", CreateImageRequest{}, func() {
		close(started)
		<-release
	})
	require.Equal(t, 0, pos)
	<-started

	// First build is in its tail: still holding the active slot.
	var secondRuns atomic.Int32
	pos = q.Enqueue("sha256:abc", CreateImageRequest{}, func() {
		secondRuns.Add(1)
	})
	assert.Equal(t, 1, pos, "second request should queue behind the in-flight build")
	assert.Equal(t, 1, q.PendingCount())

	// A duplicate request returns the existing pending position.
	pos = q.Enqueue("sha256:abc", CreateImageRequest{}, func() {
		secondRuns.Add(1)
	})
	assert.Equal(t, 1, pos)
	assert.Equal(t, 1, q.PendingCount())

	// While active and pending at once, GetPosition reports the pending slot.
	queuePos := q.GetPosition("sha256:abc")
	require.NotNil(t, queuePos)
	assert.Equal(t, 1, *queuePos)

	close(release)
	require.Eventually(t, func() bool {
		return secondRuns.Load() == 1 && q.QueueLength() == 0
	}, 5*time.Second, time.Millisecond, "queued build must run after the in-flight build completes")
}

// TestEnqueueStartsImmediatelyAfterSlotFrees verifies the normal path is
// unchanged: once the in-flight build fully completes, a new request starts
// right away instead of queueing.
func TestEnqueueStartsImmediatelyAfterSlotFrees(t *testing.T) {
	q := NewBuildQueue(1)

	done := make(chan struct{})
	pos := q.Enqueue("sha256:abc", CreateImageRequest{}, func() {
		close(done)
	})
	require.Equal(t, 0, pos)
	<-done

	require.Eventually(t, func() bool {
		return q.QueueLength() == 0
	}, 5*time.Second, time.Millisecond)

	var ran atomic.Bool
	pos = q.Enqueue("sha256:abc", CreateImageRequest{}, func() {
		ran.Store(true)
	})
	assert.Equal(t, 0, pos)
	require.Eventually(t, func() bool {
		return ran.Load()
	}, 5*time.Second, time.Millisecond)
}
