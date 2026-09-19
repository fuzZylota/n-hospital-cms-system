// Package optionscache owns a process-local cache for public and panel-safe
// site options. It is deliberately not wired into the running application.
package optionscache

import (
	"context"
	"models/data"
	"reflect"
	"sync"
)

type cacheError string

func (e cacheError) Error() string { return string(e) }

const (
	errInvalidSelection cacheError = "options cache: invalid option set selection"
	errNilContext       cacheError = "options cache: nil context"
	errNilReader        cacheError = "options cache: nil site options reader"
)

// Cache coalesces reads for each option-set selection within one generation.
// Its zero value rejects reads because it has no reader.
type Cache struct {
	reader    data.SiteOptionsReader
	readerNil bool

	mu         sync.Mutex
	generation uint64
	entries    map[data.OptionSetSelection]data.SiteOptions
	loads      map[data.OptionSetSelection]*load
}

type load struct {
	generation uint64
	done       chan struct{}
	cancel     context.CancelFunc
	waiters    int
	completed  bool
	abandoned  bool
	options    data.SiteOptions
	found      bool
	err        error
}

var _ data.SiteOptionsReader = (*Cache)(nil)

// New creates an unwired SiteOptions cache owner. It does not open, close, or
// otherwise manage the supplied reader's resources.
func New(reader data.SiteOptionsReader) *Cache {
	return &Cache{
		reader:    reader,
		readerNil: isNilReader(reader),
		entries:   make(map[data.OptionSetSelection]data.SiteOptions, 2),
		loads:     make(map[data.OptionSetSelection]*load, 2),
	}
}

// ReadSiteOptions returns a cached value or coalesces a load for the requested
// selection and current generation. Each caller may stop waiting independently;
// the shared reader call is canceled only after its last waiter leaves.
func (c *Cache) ReadSiteOptions(ctx context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
	if ctx == nil {
		return data.SiteOptions{}, false, errNilContext
	}
	if !validSelection(selection) {
		return data.SiteOptions{}, false, errInvalidSelection
	}
	if err := ctx.Err(); err != nil {
		return data.SiteOptions{}, false, err
	}
	if c == nil || c.readerNil || c.reader == nil {
		return data.SiteOptions{}, false, errNilReader
	}

	// Construct a candidate shared context before taking the cache lock. If this
	// caller finds a hit or joins another load, its unused context is canceled
	// immediately outside the lock.
	loadContext, loadCancel := context.WithCancel(context.WithoutCancel(ctx))
	c.mu.Lock()
	if options, ok := c.entries[selection]; ok {
		c.mu.Unlock()
		loadCancel()
		return options, true, nil
	}
	if current := c.loads[selection]; current != nil && current.generation == c.generation {
		current.waiters++
		c.mu.Unlock()
		loadCancel()
		return c.waitForLoad(ctx, selection, current)
	}
	current := &load{
		generation: c.generation,
		done:       make(chan struct{}),
		cancel:     loadCancel,
		waiters:    1,
	}
	c.loads[selection] = current
	c.mu.Unlock()

	go c.read(loadContext, selection, current)
	return c.waitForLoad(ctx, selection, current)
}

// Invalidate advances the cache generation and clears both active and testing
// entries. Loads from an older generation may finish for existing waiters, but
// cannot populate the new generation.
func (c *Cache) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.generation++
	clear(c.entries)
	c.loads = make(map[data.OptionSetSelection]*load, 2)
	c.mu.Unlock()
}

func (c *Cache) read(ctx context.Context, selection data.OptionSetSelection, current *load) {
	options, found, err := c.reader.ReadSiteOptions(ctx, selection)

	var cancel context.CancelFunc
	c.mu.Lock()
	current.options = options
	current.found = found
	current.err = err
	current.completed = true
	if c.loads[selection] == current {
		delete(c.loads, selection)
		if !current.abandoned && err == nil && found && c.generation == current.generation {
			c.entries[selection] = options
		}
	}
	cancel = current.cancel
	current.cancel = nil
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	close(current.done)
}

func (c *Cache) waitForLoad(ctx context.Context, selection data.OptionSetSelection, current *load) (data.SiteOptions, bool, error) {
	defer c.releaseWaiter(selection, current)
	select {
	case <-current.done:
		return current.options, current.found, current.err
	default:
	}
	select {
	case <-current.done:
		return current.options, current.found, current.err
	case <-ctx.Done():
		return data.SiteOptions{}, false, ctx.Err()
	}
}

func (c *Cache) releaseWaiter(selection data.OptionSetSelection, current *load) {
	var cancel context.CancelFunc
	c.mu.Lock()
	current.waiters--
	if current.waiters == 0 && !current.completed && !current.abandoned {
		current.abandoned = true
		if c.loads[selection] == current {
			delete(c.loads, selection)
		}
		cancel = current.cancel
		current.cancel = nil
	}
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func validSelection(selection data.OptionSetSelection) bool {
	return selection == data.ActiveOptionSet || selection == data.TestingOptionSet
}

func isNilReader(reader data.SiteOptionsReader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
