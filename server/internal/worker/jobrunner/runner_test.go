package jobrunner

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// testJob runs its units with run.
type testJob struct {
	id    uint
	units int
	run   func(ctx context.Context, unit int)
}

func (j *testJob) ID() uint                                           { return j.id }
func (j *testJob) Title() string                                      { return "test" }
func (j *testJob) Plan(context.Context) (int, error)                  { return j.units, nil }
func (j *testJob) Progress() map[string]any                           { return map[string]any{} }
func (j *testJob) Run(ctx context.Context, unit int, _ *logrus.Entry) { j.run(ctx, unit) }

func testLogger() *logrus.Entry { return logrus.NewEntry(logrus.New()) }

// TestInstancesShareJobUnits runs one job with idle instances helping it:
// every unit runs exactly once, on more than one instance, and the job ends
// only after its helpers are done.
func TestInstancesShareJobUnits(t *testing.T) {
	r := New(nil, Options{Name: "test", Instances: 4})
	var mu sync.Mutex
	seen := make(map[int]int)
	var running, inFlight, maxInFlight atomic.Int64

	const total = 200
	job := &testJob{id: 1, units: total, run: func(_ context.Context, unit int) {
		now := inFlight.Add(1)
		for {
			peak := maxInFlight.Load()
			if now <= peak || maxInFlight.CompareAndSwap(peak, now) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		mu.Lock()
		seen[unit]++
		mu.Unlock()
		running.Add(1)
		inFlight.Add(-1)
	}}
	a := newActiveJob(context.Background(), job, total, testLogger())
	stop := make(chan struct{})
	var helpers sync.WaitGroup
	for range 3 {
		helpers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				if !r.helpActiveJob(testLogger()) {
					time.Sleep(100 * time.Microsecond)
				}
			}
		})
	}

	r.process(a)
	// process returns only after every helper left the job.
	if got := running.Load(); got != total {
		t.Fatalf("units run when the job ended = %d, want %d", got, total)
	}
	close(stop)
	helpers.Wait()

	for unit := range total {
		if seen[unit] != 1 {
			t.Fatalf("unit %d ran %d times", unit, seen[unit])
		}
	}
	if maxInFlight.Load() < 2 {
		t.Fatalf("at most %d units ran at once; idle instances did not help", maxInFlight.Load())
	}
	if len(r.active) != 0 {
		t.Fatalf("finished job is still active")
	}
	if r.helpActiveJob(testLogger()) {
		t.Fatal("an instance joined a finished job")
	}
}

// TestStoppedJobLeavesUnits stops a job midway; no more units run.
func TestStoppedJobLeavesUnits(t *testing.T) {
	r := New(nil, Options{Name: "test"})
	ctx, cancel := context.WithCancel(context.Background())
	var ran atomic.Int64
	job := &testJob{id: 2, units: 50, run: func(context.Context, int) {
		if ran.Add(1) == 5 {
			cancel()
		}
	}}
	r.process(newActiveJob(ctx, job, 50, testLogger()))
	if got := ran.Load(); got != 5 {
		t.Fatalf("units run = %d, want 5", got)
	}
}

// TestRequeueChecksOnlyWhileJobsRun checks that stale jobs are looked for at
// start, then only once a claimed job may have gone stale, and at the time the
// database reports for jobs still running.
func TestRequeueChecksOnlyWhileJobsRun(t *testing.T) {
	const after = 40 * time.Millisecond
	r := New(nil, Options{Name: "test", StaleAfter: after})
	calls := make(chan time.Time, 10)
	// running is when the jobs still running go stale; nil when none is running.
	var running atomic.Pointer[time.Time]
	r.requeue = func(context.Context) (int64, *time.Time, error) {
		calls <- time.Now()
		return 0, running.Load(), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.requeueStaleJobs(ctx)
	}()

	expectCall := func(what string) time.Time {
		t.Helper()
		select {
		case at := <-calls:
			return at
		case <-time.After(time.Second):
			t.Fatalf("no check %s", what)
			return time.Time{}
		}
	}
	expectNoCall := func(what string, wait time.Duration) {
		t.Helper()
		select {
		case <-calls:
			t.Fatalf("checked %s", what)
		case <-time.After(wait):
		}
	}

	expectCall("at start")
	expectNoCall("with no running job", 3*after)

	claimed := time.Now()
	r.notifyClaimed()
	r.notifyClaimed()
	if at := expectCall("after a claim"); at.Sub(claimed) < after {
		t.Fatalf("checked %s after the claim, before the job could go stale", at.Sub(claimed))
	}
	expectNoCall("again after the claimed job finished", 3*after)

	// A job still running is checked again when the database says it may go stale.
	stale := time.Now().Add(5 * after)
	running.Store(&stale)
	r.notifyClaimed()
	expectCall("after the second claim")
	running.Store(nil)
	if at := expectCall("when the running job may go stale"); at.Before(stale) {
		t.Fatalf("checked %s before the running job could go stale", stale.Sub(at))
	}
	expectNoCall("once no job is running", 3*after)

	cancel()
	<-done
}

// fakeQueue hands out its jobs once each.
type fakeQueue struct {
	mu     sync.Mutex
	jobs   []Job
	claims atomic.Int64
	done   chan uint
}

func (q *fakeQueue) Claim(context.Context, string) (Job, error) {
	q.claims.Add(1)
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return nil, nil
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, nil
}

func (q *fakeQueue) UpdateOwned(_ context.Context, id uint, _ string, columns map[string]any) (bool, bool, error) {
	if _, finished := columns["finished_at"]; finished {
		q.done <- id
	}
	return true, false, nil
}

func (q *fakeQueue) RequeueStale(context.Context) (int64, *time.Time, error) { return 0, nil, nil }
func (q *fakeQueue) Notify(context.Context) error                            { return nil }

func (q *fakeQueue) add(job Job) {
	q.mu.Lock()
	q.jobs = append(q.jobs, job)
	q.mu.Unlock()
}

// TestWakeClaimsWithoutPolling checks that idle instances look at the queue
// when woken, not on a short poll.
func TestWakeClaimsWithoutPolling(t *testing.T) {
	queue := &fakeQueue{done: make(chan uint, 1)}
	var wake func()
	listening := make(chan struct{})
	r := New(queue, Options{Name: "test", Instances: 2, PollInterval: time.Hour, Listen: func(ctx context.Context, w func()) {
		wake = w
		close(listening)
		<-ctx.Done()
	}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)
	<-listening

	// Both instances look once at start and then wait.
	deadline := time.Now().Add(time.Second)
	for queue.claims.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	if got := queue.claims.Load(); got != 2 {
		t.Fatalf("claims before a wake-up = %d, want 2", got)
	}

	queue.add(&testJob{id: 7, units: 3, run: func(context.Context, int) {}})
	wake()
	select {
	case id := <-queue.done:
		if id != 7 {
			t.Fatalf("finished job %d, want 7", id)
		}
	case <-time.After(time.Second):
		t.Fatal("the queued job did not run after a wake-up")
	}
}
