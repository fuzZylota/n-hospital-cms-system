package optionscache

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"models/data"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type readerFunc func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error)

func (f readerFunc) ReadSiteOptions(ctx context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
	return f(ctx, selection)
}

type typedNilReader struct{}

func (*typedNilReader) ReadSiteOptions(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
	panic("typed nil reader must not be called")
}

type readResult struct {
	options data.SiteOptions
	found   bool
	err     error
}

type observedContext struct {
	context.Context
	doneObserved chan struct{}
	once         sync.Once
}

func (c *observedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.doneObserved) })
	return c.Context.Done()
}

func TestActiveMissLoadHit(t *testing.T) {
	want := activeOptions("active", "/active-logo.webp")
	unexpectedSelection := errors.New("unexpected selection")
	var calls int
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		if selection != data.ActiveOptionSet {
			return data.SiteOptions{}, false, unexpectedSelection
		}
		return want, true, nil
	}))

	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	if calls != 1 {
		t.Fatal("active cache hit called the reader")
	}
}

func TestTestingMissLoadHit(t *testing.T) {
	want := testingOptions("testing", "/testing-logo.webp")
	unexpectedSelection := errors.New("unexpected selection")
	var calls int
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		if selection != data.TestingOptionSet {
			return data.SiteOptions{}, false, unexpectedSelection
		}
		return want, true, nil
	}))

	assertSuccessfulRead(t, cache, data.TestingOptionSet, want)
	assertSuccessfulRead(t, cache, data.TestingOptionSet, want)
	if calls != 1 {
		t.Fatal("testing cache hit called the reader")
	}
}

func TestActiveAndTestingEntriesAreIsolated(t *testing.T) {
	active := activeOptions("active site", "/active-logo.webp")
	testingOptions := testingOptions("testing site", "/testing-logo.webp")
	unexpectedSelection := errors.New("unexpected selection")
	var activeCalls, testingCalls int
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch selection {
		case data.ActiveOptionSet:
			activeCalls++
			return active, true, nil
		case data.TestingOptionSet:
			testingCalls++
			return testingOptions, true, nil
		default:
			return data.SiteOptions{}, false, unexpectedSelection
		}
	}))

	assertSuccessfulRead(t, cache, data.ActiveOptionSet, active)
	assertSuccessfulRead(t, cache, data.TestingOptionSet, testingOptions)
	assertSuccessfulRead(t, cache, data.TestingOptionSet, testingOptions)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, active)
	if activeCalls != 1 || testingCalls != 1 {
		t.Fatal("active and testing entries did not retain independent hits")
	}
}

func TestMissingTestingDoesNotFallbackToActive(t *testing.T) {
	var activeCalls, testingCalls int
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch selection {
		case data.ActiveOptionSet:
			activeCalls++
			return activeOptions("active", "/active.webp"), true, nil
		case data.TestingOptionSet:
			testingCalls++
			return data.SiteOptions{}, false, nil
		default:
			return data.SiteOptions{}, false, errInvalidSelection
		}
	}))

	options, found, err := cache.ReadSiteOptions(context.Background(), data.TestingOptionSet)
	if err != nil || found || options != (data.SiteOptions{}) {
		t.Fatal("missing testing lookup did not return the safe missing result")
	}
	if activeCalls != 0 || testingCalls != 1 {
		t.Fatal("missing testing lookup fell back or used the wrong selection")
	}
}

func TestMissingResultIsNotCached(t *testing.T) {
	var calls int
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		return data.SiteOptions{}, false, nil
	}))

	for range 2 {
		options, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
		if err != nil || found || options != (data.SiteOptions{}) {
			t.Fatal("missing lookup returned an unsafe result")
		}
	}
	if calls != 2 {
		t.Fatal("missing result was negative-cached")
	}
}

func TestReaderErrorIsNotCached(t *testing.T) {
	readerFailure := errors.New("safe reader failure")
	want := activeOptions("recovered", "/recovered.webp")
	var calls int
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		if calls == 1 {
			return data.SiteOptions{}, false, readerFailure
		}
		return want, true, nil
	}))

	options, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	if !errors.Is(err, readerFailure) || found || options != (data.SiteOptions{}) {
		t.Fatal("reader failure identity or safe zero result was not preserved")
	}
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	if calls != 2 {
		t.Fatal("reader failure was cached or successful retry was not cached")
	}
}

func TestInvalidSelectionsAreRejectedBeforeReader(t *testing.T) {
	var calls atomic.Int32
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		return data.SiteOptions{}, false, nil
	}))

	for _, selection := range []data.OptionSetSelection{0, 3, 255} {
		options, found, err := cache.ReadSiteOptions(context.Background(), selection)
		if !errors.Is(err, errInvalidSelection) || found || options != (data.SiteOptions{}) {
			t.Fatal("invalid selection did not return the fixed safe rejection")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid selection reached the reader")
	}
}

func TestNilReaderAndNilContextAreSafe(t *testing.T) {
	t.Run("nil reader", func(t *testing.T) {
		assertRejectedRead(t, New(nil), context.Background(), data.ActiveOptionSet, errNilReader)
	})
	t.Run("typed nil reader", func(t *testing.T) {
		var reader *typedNilReader
		assertRejectedRead(t, New(reader), context.Background(), data.ActiveOptionSet, errNilReader)
	})
	t.Run("nil context", func(t *testing.T) {
		var calls atomic.Int32
		cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
			calls.Add(1)
			return data.SiteOptions{}, false, nil
		}))
		assertRejectedRead(t, cache, nil, data.ActiveOptionSet, errNilContext)
		if calls.Load() != 0 {
			t.Fatal("nil context reached the reader")
		}
	})
}

func TestPreCanceledAndExpiredContextsDoNotReachReader(t *testing.T) {
	var calls atomic.Int32
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		return data.SiteOptions{}, false, nil
	}))

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	assertRejectedRead(t, cache, canceled, data.ActiveOptionSet, context.Canceled)

	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Unix(0, 0))
	t.Cleanup(cancelDeadline)
	assertRejectedRead(t, cache, expired, data.TestingOptionSet, context.DeadlineExceeded)

	if calls.Load() != 0 {
		t.Fatal("pre-failed context reached the reader")
	}
}

func TestReaderContextErrorIdentityIsPreserved(t *testing.T) {
	for _, readerError := range []error{context.Canceled, context.DeadlineExceeded} {
		cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
			return data.SiteOptions{}, false, readerError
		}))
		options, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
		if !errors.Is(err, readerError) || found || options != (data.SiteOptions{}) {
			t.Fatal("reader context error identity was not preserved")
		}
	}
}

func TestConcurrentMissesForOneSelectionAreCoalesced(t *testing.T) {
	const callerCount = 12
	want := activeOptions("coalesced", "/coalesced.webp")
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		if calls.Add(1) == 1 {
			close(loadStarted)
		}
		<-releaseLoad
		return want, true, nil
	}))

	results := make([]<-chan readResult, callerCount)
	observed := make([]chan struct{}, callerCount)
	for index := range callerCount {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		observed[index] = make(chan struct{})
		results[index] = readAsync(cache, &observedContext{Context: ctx, doneObserved: observed[index]}, data.ActiveOptionSet)
	}
	waitSignal(t, loadStarted, "coalesced reader did not start")
	for index := range callerCount {
		waitSignal(t, observed[index], "caller did not enter the load wait")
	}
	close(releaseLoad)

	for index := range callerCount {
		assertSuccessfulResult(t, waitResult(t, results[index], "coalesced caller did not finish"), want)
	}
	if calls.Load() != 1 {
		t.Fatal("same-generation misses were not coalesced")
	}
}

func TestWaitingCallerCancellationDoesNotCancelSharedLoad(t *testing.T) {
	want := activeOptions("shared", "/shared.webp")
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	readerContext := make(chan context.Context, 1)
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		readerContext <- ctx
		close(loadStarted)
		<-releaseLoad
		return want, true, nil
	}))

	leader := readAsync(cache, context.Background(), data.ActiveOptionSet)
	waitSignal(t, loadStarted, "shared reader did not start")

	canceledBase, cancel := context.WithCancel(context.Background())
	canceledObserved := make(chan struct{})
	canceledCaller := readAsync(cache, &observedContext{Context: canceledBase, doneObserved: canceledObserved}, data.ActiveOptionSet)

	survivorBase, survivorCancel := context.WithCancel(context.Background())
	t.Cleanup(survivorCancel)
	survivorObserved := make(chan struct{})
	survivor := readAsync(cache, &observedContext{Context: survivorBase, doneObserved: survivorObserved}, data.ActiveOptionSet)

	waitSignal(t, canceledObserved, "canceling caller did not enter the shared wait")
	waitSignal(t, survivorObserved, "surviving caller did not enter the shared wait")
	cancel()
	canceledResult := waitResult(t, canceledCaller, "canceled caller did not stop waiting")
	if !errors.Is(canceledResult.err, context.Canceled) || canceledResult.found || canceledResult.options != (data.SiteOptions{}) {
		t.Fatal("canceled waiter did not return its own safe context result")
	}
	sharedContext := <-readerContext
	if sharedContext.Err() != nil {
		t.Fatal("one canceled waiter canceled a load that still had active waiters")
	}

	close(releaseLoad)
	assertSuccessfulResult(t, waitResult(t, leader, "leader did not finish"), want)
	assertSuccessfulResult(t, waitResult(t, survivor, "surviving caller did not finish"), want)
	if calls.Load() != 1 {
		t.Fatal("waiter cancellation restarted or canceled the shared load")
	}
}

func TestLastWaiterCancellationCancelsOwnedLoad(t *testing.T) {
	loadStarted := make(chan struct{})
	readerCanceled := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		close(loadStarted)
		<-ctx.Done()
		close(readerCanceled)
		return data.SiteOptions{}, false, ctx.Err()
	}))

	callerContext, cancelCaller := context.WithCancel(context.Background())
	result := readAsync(cache, callerContext, data.ActiveOptionSet)
	waitSignal(t, loadStarted, "owned load reader did not start")
	current := registeredLoad(t, cache, data.ActiveOptionSet)
	cancelCaller()

	canceledResult := waitResult(t, result, "last waiter did not stop waiting")
	if !errors.Is(canceledResult.err, context.Canceled) || canceledResult.found || canceledResult.options != (data.SiteOptions{}) {
		t.Fatal("last waiter did not return its own safe context result")
	}
	waitSignal(t, readerCanceled, "last waiter departure did not cancel the owned load context")
	waitSignal(t, current.done, "canceled owned load did not complete")
	assertTerminalLoadState(t, cache, data.ActiveOptionSet, current, true)
	if calls.Load() != 1 {
		t.Fatal("last waiter cancellation caused an unexpected reader call")
	}
}

func TestAllWaitersMustLeaveBeforeOwnedLoadIsCanceled(t *testing.T) {
	loadStarted := make(chan struct{})
	readerCanceled := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		close(loadStarted)
		<-ctx.Done()
		close(readerCanceled)
		return data.SiteOptions{}, false, ctx.Err()
	}))

	leaderContext, cancelLeader := context.WithCancel(context.Background())
	leader := readAsync(cache, leaderContext, data.ActiveOptionSet)
	waitSignal(t, loadStarted, "leader load did not start")
	current := registeredLoad(t, cache, data.ActiveOptionSet)

	followerContext, cancelFollower := context.WithCancel(context.Background())
	followerObserved := make(chan struct{})
	follower := readAsync(cache, &observedContext{Context: followerContext, doneObserved: followerObserved}, data.ActiveOptionSet)
	waitSignal(t, followerObserved, "follower did not join the shared load")

	cancelLeader()
	leaderResult := waitResult(t, leader, "leader did not stop waiting")
	if !errors.Is(leaderResult.err, context.Canceled) {
		t.Fatal("leader did not return its own cancellation")
	}
	select {
	case <-readerCanceled:
		t.Fatal("owned load was canceled while a follower remained")
	default:
	}

	cancelFollower()
	followerResult := waitResult(t, follower, "last follower did not stop waiting")
	if !errors.Is(followerResult.err, context.Canceled) {
		t.Fatal("follower did not return its own cancellation")
	}
	waitSignal(t, readerCanceled, "owned load was not canceled after all waiters left")
	waitSignal(t, current.done, "all-waiter cancellation load did not complete")
	assertTerminalLoadState(t, cache, data.ActiveOptionSet, current, true)
	if calls.Load() != 1 {
		t.Fatal("all-waiter cancellation did not retain a single reader call")
	}
}

func TestAbandonedLoadLateCompletionCannotCorruptRetry(t *testing.T) {
	oldOptions := activeOptions("abandoned", "/abandoned.webp")
	newOptions := activeOptions("retry", "/retry.webp")
	oldStarted := make(chan struct{})
	oldCanceled := make(chan struct{})
	allowOldReturn := make(chan struct{})
	newStarted := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch calls.Add(1) {
		case 1:
			close(oldStarted)
			<-ctx.Done()
			close(oldCanceled)
			<-allowOldReturn
			return oldOptions, true, nil
		case 2:
			close(newStarted)
			return newOptions, true, nil
		default:
			return data.SiteOptions{}, false, errors.New("unexpected reader call")
		}
	}))

	oldContext, cancelOld := context.WithCancel(context.Background())
	oldResult := readAsync(cache, oldContext, data.ActiveOptionSet)
	waitSignal(t, oldStarted, "abandoned load did not start")
	oldLoad := registeredLoad(t, cache, data.ActiveOptionSet)
	cancelOld()
	canceledResult := waitResult(t, oldResult, "abandoned load caller did not stop waiting")
	if !errors.Is(canceledResult.err, context.Canceled) {
		t.Fatal("abandoned load caller did not return its own cancellation")
	}
	waitSignal(t, oldCanceled, "abandoned load context was not canceled")

	retryResult := readAsync(cache, context.Background(), data.ActiveOptionSet)
	waitSignal(t, newStarted, "retry joined the abandoned load instead of starting a new load")
	assertSuccessfulResult(t, waitResult(t, retryResult, "retry load did not finish"), newOptions)

	close(allowOldReturn)
	waitSignal(t, oldLoad.done, "abandoned load did not finish its delayed return")
	assertTerminalLoadState(t, cache, data.ActiveOptionSet, oldLoad, true)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, newOptions)
	if calls.Load() != 2 {
		t.Fatal("abandoned load late completion corrupted the retry cache entry")
	}
}

func TestInvalidateThenOldLastWaiterCancellationIsIsolated(t *testing.T) {
	newOptions := activeOptions("new generation", "/new-generation.webp")
	oldStarted := make(chan struct{})
	oldCanceled := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch calls.Add(1) {
		case 1:
			close(oldStarted)
			<-ctx.Done()
			close(oldCanceled)
			return data.SiteOptions{}, false, ctx.Err()
		case 2:
			return newOptions, true, nil
		default:
			return data.SiteOptions{}, false, errors.New("unexpected reader call")
		}
	}))

	oldContext, cancelOld := context.WithCancel(context.Background())
	oldResult := readAsync(cache, oldContext, data.ActiveOptionSet)
	waitSignal(t, oldStarted, "pre-invalidate load did not start")
	oldLoad := registeredLoad(t, cache, data.ActiveOptionSet)
	cache.Invalidate()
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, newOptions)

	cancelOld()
	canceledResult := waitResult(t, oldResult, "old generation waiter did not stop waiting")
	if !errors.Is(canceledResult.err, context.Canceled) {
		t.Fatal("old generation waiter did not return its own cancellation")
	}
	waitSignal(t, oldCanceled, "old generation load context was not canceled")
	waitSignal(t, oldLoad.done, "old generation canceled load did not complete")
	assertTerminalLoadState(t, cache, data.ActiveOptionSet, oldLoad, true)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, newOptions)
	if calls.Load() != 2 {
		t.Fatal("old generation cancellation disturbed the new generation")
	}
}

func TestCompletionCancellationRaceReleasesWaiterExactlyOnce(t *testing.T) {
	const iterations = 64
	for range iterations {
		want := activeOptions("race", "/race.webp")
		loadStarted := make(chan struct{})
		releaseLoad := make(chan struct{})
		var calls atomic.Int32
		cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
			if calls.Add(1) == 1 {
				close(loadStarted)
				<-releaseLoad
			}
			return want, true, nil
		}))

		callerContext, cancelCaller := context.WithCancel(context.Background())
		result := readAsync(cache, callerContext, data.ActiveOptionSet)
		waitSignal(t, loadStarted, "race load did not start")
		current := registeredLoad(t, cache, data.ActiveOptionSet)

		startRace := make(chan struct{})
		cancelIssued := make(chan struct{})
		releaseIssued := make(chan struct{})
		go func() {
			<-startRace
			cancelCaller()
			close(cancelIssued)
		}()
		go func() {
			<-startRace
			close(releaseLoad)
			close(releaseIssued)
		}()
		close(startRace)
		waitSignal(t, cancelIssued, "race cancellation was not issued")
		waitSignal(t, releaseIssued, "race completion was not released")

		got := waitResult(t, result, "completion/cancellation race caller did not finish")
		if got.err == nil {
			assertSuccessfulResult(t, got, want)
		} else if !errors.Is(got.err, context.Canceled) || got.found || got.options != (data.SiteOptions{}) {
			t.Fatal("completion/cancellation race returned an unsafe result")
		}
		waitSignal(t, current.done, "completion/cancellation race load did not finish")
		assertTerminalLoadReleased(t, cache, data.ActiveOptionSet, current)
		assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
		if calls.Load() < 1 || calls.Load() > 2 {
			t.Fatal("completion/cancellation race left invalid registry state")
		}
	}
}

func TestInitiatingCallerCancellationDoesNotLeakToWaiters(t *testing.T) {
	want := activeOptions("detached", "/detached.webp")
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	readerContext := make(chan context.Context, 1)
	var calls atomic.Int32
	cache := New(readerFunc(func(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		readerContext <- ctx
		close(loadStarted)
		<-releaseLoad
		return want, true, nil
	}))

	leaderContext, cancelLeader := context.WithCancel(context.Background())
	leader := readAsync(cache, leaderContext, data.ActiveOptionSet)
	waitSignal(t, loadStarted, "initiating caller load did not start")

	survivorContext, cancelSurvivor := context.WithCancel(context.Background())
	t.Cleanup(cancelSurvivor)
	survivorObserved := make(chan struct{})
	survivor := readAsync(cache, &observedContext{Context: survivorContext, doneObserved: survivorObserved}, data.ActiveOptionSet)
	waitSignal(t, survivorObserved, "surviving waiter did not enter the shared wait")

	cancelLeader()
	leaderResult := waitResult(t, leader, "initiating caller did not stop waiting")
	if !errors.Is(leaderResult.err, context.Canceled) || leaderResult.found || leaderResult.options != (data.SiteOptions{}) {
		t.Fatal("initiating caller did not return its own safe context result")
	}
	sharedContext := <-readerContext
	if sharedContext.Err() != nil {
		t.Fatal("initiating caller cancellation leaked into the shared reader context")
	}

	close(releaseLoad)
	assertSuccessfulResult(t, waitResult(t, survivor, "surviving waiter did not finish"), want)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	if calls.Load() != 1 {
		t.Fatal("initiating caller cancellation restarted or canceled the shared load")
	}
}

func TestActiveAndTestingLoadsProgressIndependently(t *testing.T) {
	activeStarted := make(chan struct{})
	testingStarted := make(chan struct{})
	releaseActive := make(chan struct{})
	releaseTesting := make(chan struct{})
	active := activeOptions("active", "/active.webp")
	testingOptions := testingOptions("testing", "/testing.webp")
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch selection {
		case data.ActiveOptionSet:
			close(activeStarted)
			<-releaseActive
			return active, true, nil
		case data.TestingOptionSet:
			close(testingStarted)
			<-releaseTesting
			return testingOptions, true, nil
		default:
			return data.SiteOptions{}, false, errInvalidSelection
		}
	}))

	activeResult := readAsync(cache, context.Background(), data.ActiveOptionSet)
	waitSignal(t, activeStarted, "active load did not start")
	testingResult := readAsync(cache, context.Background(), data.TestingOptionSet)
	waitSignal(t, testingStarted, "testing load was blocked by active load")
	close(releaseTesting)
	assertSuccessfulResult(t, waitResult(t, testingResult, "testing load did not finish"), testingOptions)
	close(releaseActive)
	assertSuccessfulResult(t, waitResult(t, activeResult, "active load did not finish"), active)
}

func TestInvalidateDuringLoadPreventsStaleCacheWrite(t *testing.T) {
	oldOptions := activeOptions("old", "/old.webp")
	newOptions := activeOptions("new", "/new.webp")
	oldStarted := make(chan struct{})
	releaseOld := make(chan struct{})
	newStarted := make(chan struct{})
	var calls atomic.Int32
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch calls.Add(1) {
		case 1:
			close(oldStarted)
			<-releaseOld
			return oldOptions, true, nil
		case 2:
			close(newStarted)
			return newOptions, true, nil
		default:
			return data.SiteOptions{}, false, errors.New("unexpected reader call")
		}
	}))

	oldResult := readAsync(cache, context.Background(), data.ActiveOptionSet)
	waitSignal(t, oldStarted, "old generation load did not start")
	cache.Invalidate()
	newResult := readAsync(cache, context.Background(), data.ActiveOptionSet)
	waitSignal(t, newStarted, "new generation load did not start")
	assertSuccessfulResult(t, waitResult(t, newResult, "new generation load did not finish"), newOptions)
	close(releaseOld)
	assertSuccessfulResult(t, waitResult(t, oldResult, "old generation caller did not finish"), oldOptions)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, newOptions)
	if calls.Load() != 2 {
		t.Fatal("stale load repopulated or displaced the new generation")
	}
}

func TestInvalidateForcesBothSelectionsToReload(t *testing.T) {
	var activeCalls, testingCalls int
	cache := New(readerFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch selection {
		case data.ActiveOptionSet:
			activeCalls++
			return activeOptions("active", "/active.webp"), true, nil
		case data.TestingOptionSet:
			testingCalls++
			return testingOptions("testing", "/testing.webp"), true, nil
		default:
			return data.SiteOptions{}, false, errInvalidSelection
		}
	}))

	assertSuccessfulRead(t, cache, data.ActiveOptionSet, activeOptions("active", "/active.webp"))
	assertSuccessfulRead(t, cache, data.TestingOptionSet, testingOptions("testing", "/testing.webp"))
	cache.Invalidate()
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, activeOptions("active", "/active.webp"))
	assertSuccessfulRead(t, cache, data.TestingOptionSet, testingOptions("testing", "/testing.webp"))
	if activeCalls != 2 || testingCalls != 2 {
		t.Fatal("invalidate did not clear both selections")
	}
}

func TestMultipleInvalidationsAdvanceGenerations(t *testing.T) {
	var calls int
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		return activeOptions("generation", "/generation.webp"), true, nil
	}))

	assertSuccessfulRead(t, cache, data.ActiveOptionSet, activeOptions("generation", "/generation.webp"))
	cache.Invalidate()
	cache.Invalidate()
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, activeOptions("generation", "/generation.webp"))
	cache.Invalidate()
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, activeOptions("generation", "/generation.webp"))
	if calls != 3 || cache.generation != 3 {
		t.Fatal("multiple invalidations did not create distinct generations")
	}
}

func TestCallerMutationDoesNotChangeCachedValue(t *testing.T) {
	want := activeOptions("immutable", "/immutable.webp")
	var calls int
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		return want, true, nil
	}))

	got, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	if err != nil || !found {
		t.Fatal("initial cache load failed")
	}
	got.SiteName = "caller mutation"
	got.SiteLogo.Path = "/caller-mutation.webp"
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	if calls != 1 {
		t.Fatal("caller mutation caused an unexpected reload")
	}
}

func TestReaderCallbackMayInvalidateWithoutDeadlock(t *testing.T) {
	var cache *Cache
	var calls atomic.Int32
	cache = New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls.Add(1)
		cache.Invalidate()
		return activeOptions("callback", "/callback.webp"), true, nil
	}))

	for range 2 {
		result := waitResult(t, readAsync(cache, context.Background(), data.ActiveOptionSet), "reader callback invalidation deadlocked")
		assertSuccessfulResult(t, result, activeOptions("callback", "/callback.webp"))
	}
	if calls.Load() != 2 {
		t.Fatal("callback invalidation allowed an old-generation cache write")
	}
}

func TestReaderRunsOutsideCacheMutex(t *testing.T) {
	callbackUnderLock := errors.New("reader called under cache mutex")
	var cache *Cache
	cache = New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		if !cache.mu.TryLock() {
			return data.SiteOptions{}, false, callbackUnderLock
		}
		cache.mu.Unlock()
		return activeOptions("outside lock", "/outside-lock.webp"), true, nil
	}))

	options, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	if errors.Is(err, callbackUnderLock) || err != nil || !found || options.SiteName != "outside lock" {
		t.Fatal("reader callback did not run outside the cache mutex")
	}
}

func TestMissingAndErrorLoadsReleaseInflightState(t *testing.T) {
	readerFailure := errors.New("safe transient failure")
	want := activeOptions("eventual success", "/eventual.webp")
	var calls int
	cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
		calls++
		switch calls {
		case 1:
			return data.SiteOptions{}, false, nil
		case 2:
			return data.SiteOptions{}, false, readerFailure
		default:
			return want, true, nil
		}
	}))

	options, found, err := cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	if err != nil || found || options != (data.SiteOptions{}) {
		t.Fatal("missing in-flight result was unsafe")
	}
	options, found, err = cache.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	if !errors.Is(err, readerFailure) || found || options != (data.SiteOptions{}) {
		t.Fatal("error in-flight result was unsafe")
	}
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
	if calls != 3 {
		t.Fatal("missing or error result left a stuck in-flight load")
	}
}

func TestMissingAndErrorLoadsReleaseCancelResources(t *testing.T) {
	readerFailure := errors.New("safe terminal failure")
	tests := []struct {
		name  string
		found bool
		err   error
	}{
		{name: "missing"},
		{name: "error", err: readerFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loadStarted := make(chan struct{})
			releaseLoad := make(chan struct{})
			want := activeOptions("retry after terminal", "/retry-terminal.webp")
			var calls atomic.Int32
			cache := New(readerFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
				if calls.Add(1) == 1 {
					close(loadStarted)
					<-releaseLoad
					return data.SiteOptions{}, test.found, test.err
				}
				return want, true, nil
			}))

			result := readAsync(cache, context.Background(), data.ActiveOptionSet)
			waitSignal(t, loadStarted, "terminal resource load did not start")
			current := registeredLoad(t, cache, data.ActiveOptionSet)
			close(releaseLoad)
			got := waitResult(t, result, "terminal resource load did not finish")
			if !errors.Is(got.err, test.err) || got.found != test.found || got.options != (data.SiteOptions{}) {
				t.Fatal("terminal resource load returned an unsafe result")
			}
			waitSignal(t, current.done, "terminal resource completion was not published")
			assertTerminalLoadState(t, cache, data.ActiveOptionSet, current, false)
			assertSuccessfulRead(t, cache, data.ActiveOptionSet, want)
			if calls.Load() != 2 {
				t.Fatal("terminal resource release prevented a fresh retry")
			}
		})
	}
}

func TestSiteOptionsRemainsValueOnly(t *testing.T) {
	assertValueOnlyType(t, reflect.TypeOf(data.SiteOptions{}), "SiteOptions")
}

func TestPublicAPIShape(t *testing.T) {
	readerType := reflect.TypeOf((*data.SiteOptionsReader)(nil)).Elem()
	cacheType := reflect.TypeOf((*Cache)(nil))
	newType := reflect.TypeOf(New)
	if newType.NumIn() != 1 || newType.In(0) != readerType || newType.NumOut() != 1 || newType.Out(0) != cacheType {
		t.Fatal("New does not have the required reader-to-cache shape")
	}
	readMethod, exists := cacheType.MethodByName("ReadSiteOptions")
	if !exists || readMethod.Type.NumIn() != 3 || readMethod.Type.NumOut() != 3 {
		t.Fatal("ReadSiteOptions does not have the required method shape")
	}
	invalidateMethod, exists := cacheType.MethodByName("Invalidate")
	if !exists || invalidateMethod.Type.NumIn() != 1 || invalidateMethod.Type.NumOut() != 0 {
		t.Fatal("Invalidate does not have the required method shape")
	}
}

func TestProductionImportAndProjectionBoundary(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source location is unavailable")
	}
	productionFile := filepath.Join(filepath.Dir(testFile), "cache.go")
	source, err := os.ReadFile(productionFile)
	if err != nil {
		t.Fatal("production source could not be read")
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), productionFile, source, parser.ImportsOnly)
	if err != nil {
		t.Fatal("production source could not be parsed")
	}
	allowed := map[string]bool{
		"context": true, "errors": true, "models/data": true, "reflect": true, "sync": true,
	}
	for _, imported := range parsed.Imports {
		path, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil || !allowed[path] {
			t.Fatal("production source has an import outside the cache boundary")
		}
	}
	for _, forbidden := range []string{
		"MailDeliveryOptions", "CaptchaVerificationOptions", "UploadPolicy", "PasswordPolicy", "OptionMediaReferences",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatal("production cache references a forbidden option projection")
		}
	}
}

func activeOptions(siteName, logoPath string) data.SiteOptions {
	return data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: "active", IsActive: true},
		SiteName: siteName,
		SiteLogo: data.PublicMedia{Path: logoPath, AltText: "active logo"},
	}
}

func testingOptions(siteName, logoPath string) data.SiteOptions {
	return data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: "testing", IsTesting: true},
		SiteName: siteName,
		SiteLogo: data.PublicMedia{Path: logoPath, AltText: "testing logo"},
	}
}

func assertSuccessfulRead(t *testing.T, cache *Cache, selection data.OptionSetSelection, want data.SiteOptions) {
	t.Helper()
	options, found, err := cache.ReadSiteOptions(context.Background(), selection)
	if err != nil || !found || options != want {
		t.Fatal("cache read did not return the expected successful result")
	}
}

func assertRejectedRead(t *testing.T, cache *Cache, ctx context.Context, selection data.OptionSetSelection, wantError error) {
	t.Helper()
	options, found, err := cache.ReadSiteOptions(ctx, selection)
	if !errors.Is(err, wantError) || found || options != (data.SiteOptions{}) {
		t.Fatal("cache rejection did not return the expected safe result")
	}
}

func readAsync(cache *Cache, ctx context.Context, selection data.OptionSetSelection) <-chan readResult {
	result := make(chan readResult, 1)
	go func() {
		options, found, err := cache.ReadSiteOptions(ctx, selection)
		result <- readResult{options: options, found: found, err: err}
	}()
	return result
}

func waitSignal(t *testing.T, signal <-chan struct{}, deadlockMessage string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(deadlockMessage)
	}
}

func waitResult(t *testing.T, result <-chan readResult, deadlockMessage string) readResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal(deadlockMessage)
		return readResult{}
	}
}

func assertSuccessfulResult(t *testing.T, result readResult, want data.SiteOptions) {
	t.Helper()
	if result.err != nil || !result.found || result.options != want {
		t.Fatal("asynchronous cache read did not return the expected successful result")
	}
}

func registeredLoad(t *testing.T, cache *Cache, selection data.OptionSetSelection) *load {
	t.Helper()
	cache.mu.Lock()
	defer cache.mu.Unlock()
	current := cache.loads[selection]
	if current == nil {
		t.Fatal("expected in-flight load is not registered")
	}
	return current
}

func assertTerminalLoadState(t *testing.T, cache *Cache, selection data.OptionSetSelection, current *load, wantAbandoned bool) {
	t.Helper()
	cache.mu.Lock()
	invalid := current.waiters != 0 || !current.completed || current.cancel != nil ||
		cache.loads[selection] == current || current.abandoned != wantAbandoned
	cache.mu.Unlock()
	if invalid {
		t.Fatal("terminal load ownership state is invalid")
	}
}

func assertTerminalLoadReleased(t *testing.T, cache *Cache, selection data.OptionSetSelection, current *load) {
	t.Helper()
	cache.mu.Lock()
	invalid := current.waiters != 0 || !current.completed || current.cancel != nil || cache.loads[selection] == current
	cache.mu.Unlock()
	if invalid {
		t.Fatal("terminal load retained waiter, cancel, or registry ownership")
	}
}

func assertValueOnlyType(t *testing.T, valueType reflect.Type, path string) {
	t.Helper()
	switch valueType.Kind() {
	case reflect.Struct:
		for index := 0; index < valueType.NumField(); index++ {
			field := valueType.Field(index)
			assertValueOnlyType(t, field.Type, path+"."+field.Name)
		}
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return
	default:
		t.Fatalf("%s must remain value-only", path)
	}
}
