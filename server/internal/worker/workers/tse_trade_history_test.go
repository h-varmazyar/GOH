package workers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/sirupsen/logrus"
)

func testMarkets(count int) []*models.TSEMarket {
	markets := make([]*models.TSEMarket, count)
	for i := range markets {
		markets[i] = &models.TSEMarket{ID: uint(i + 1), Label: "M"}
	}
	return markets
}

// TestInstancesShareJobMarkets runs one job with idle instances helping it:
// every market is fetched exactly once, by more than one instance, and the
// job ends only after its helpers are done.
func TestInstancesShareJobMarkets(t *testing.T) {
	w := NewTSETradeHistory(nil, &configs.TSETradeHistory{Instances: 4})
	var mu sync.Mutex
	seen := make(map[uint]int)
	var running, inFlight atomic.Int64
	var maxInFlight atomic.Int64
	w.collect = func(_ context.Context, _ *models.TSETradeHistoryJob, market *models.TSEMarket, _ *jobProgress, _ *logrus.Entry) {
		now := inFlight.Add(1)
		for {
			peak := maxInFlight.Load()
			if now <= peak || maxInFlight.CompareAndSwap(peak, now) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		mu.Lock()
		seen[market.ID]++
		mu.Unlock()
		running.Add(1)
		inFlight.Add(-1)
	}

	const total = 200
	a := newActiveJob(context.Background(), &models.TSETradeHistoryJob{ID: 1}, testMarkets(total), logrus.NewEntry(logrus.New()))
	stop := make(chan struct{})
	var helpers sync.WaitGroup
	for range 3 {
		helpers.Add(1)
		go func() {
			defer helpers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if !w.helpActiveJob(logrus.NewEntry(logrus.New())) {
					time.Sleep(100 * time.Microsecond)
				}
			}
		}()
	}

	w.process(a)
	// process returns only after every helper left the job.
	if got := running.Load(); got != total {
		t.Fatalf("markets fetched when the job ended = %d, want %d", got, total)
	}
	close(stop)
	helpers.Wait()

	for id := uint(1); id <= total; id++ {
		if seen[id] != 1 {
			t.Fatalf("market %d fetched %d times", id, seen[id])
		}
	}
	if maxInFlight.Load() < 2 {
		t.Fatalf("at most %d markets were fetched at once; idle instances did not help", maxInFlight.Load())
	}
	if len(w.active) != 0 {
		t.Fatalf("finished job is still active")
	}
	if w.helpActiveJob(logrus.NewEntry(logrus.New())) {
		t.Fatal("an instance joined a finished job")
	}
}

// TestStoppedJobLeavesMarkets stops a job midway; no more markets are fetched.
func TestStoppedJobLeavesMarkets(t *testing.T) {
	w := NewTSETradeHistory(nil, &configs.TSETradeHistory{})
	ctx, cancel := context.WithCancel(context.Background())
	var fetched atomic.Int64
	w.collect = func(context.Context, *models.TSETradeHistoryJob, *models.TSEMarket, *jobProgress, *logrus.Entry) {
		if fetched.Add(1) == 5 {
			cancel()
		}
	}
	w.process(newActiveJob(ctx, &models.TSETradeHistoryJob{ID: 2}, testMarkets(50), logrus.NewEntry(logrus.New())))
	if got := fetched.Load(); got != 5 {
		t.Fatalf("markets fetched = %d, want 5", got)
	}
}

// TestRequeueChecksOnlyWhileJobsRun checks that stale jobs are looked for at
// start, then only once a claimed job may have gone stale, and at the time the
// database reports for jobs still running.
func TestRequeueChecksOnlyWhileJobsRun(t *testing.T) {
	const after = 40 * time.Millisecond
	w := NewTSETradeHistory(nil, &configs.TSETradeHistory{StaleJobAfter: after})
	calls := make(chan time.Time, 10)
	// running is when the jobs still running go stale; nil when none is running.
	var running atomic.Pointer[time.Time]
	w.requeue = func(context.Context) (int64, *time.Time, error) {
		calls <- time.Now()
		return 0, running.Load(), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.requeueStaleJobs(ctx)
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
	w.notifyClaimed()
	w.notifyClaimed()
	if at := expectCall("after a claim"); at.Sub(claimed) < after {
		t.Fatalf("checked %s after the claim, before the job could go stale", at.Sub(claimed))
	}
	expectNoCall("again after the claimed job finished", 3*after)

	// A job still running is checked again when the database says it may go stale.
	stale := time.Now().Add(5 * after)
	running.Store(&stale)
	w.notifyClaimed()
	expectCall("after the second claim")
	running.Store(nil)
	if at := expectCall("when the running job may go stale"); at.Before(stale) {
		t.Fatalf("checked %s before the running job could go stale", stale.Sub(at))
	}
	expectNoCall("once no job is running", 3*after)

	cancel()
	<-done
}

func TestJobProgressListsCurrentMarkets(t *testing.T) {
	progress := newJobProgress()
	progress.start(&models.TSEMarket{ID: 1, Label: "فولاد"})
	progress.start(&models.TSEMarket{ID: 2, Label: "خودرو"})
	progress.finish(&models.TSEMarket{ID: 1})
	if got := progress.columns()["current_market"]; got != "خودرو" {
		t.Fatalf("current market = %q", got)
	}
}
