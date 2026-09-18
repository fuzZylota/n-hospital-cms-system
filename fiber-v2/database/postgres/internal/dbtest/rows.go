package dbtest

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"
)

// ErrInvalidFixture identifies invalid row data, not a database/sql Scan error.
// QueryContext returns it before opening an iterator, without exposing values.
var ErrInvalidFixture = errors.New("dbtest: invalid rows fixture configuration")

// Rows is a reusable row script. Each opened iterator has independent data and
// position. Iterators sharing this Rows (even across connectors) share CloseCount.
type Rows struct {
	columns    []string
	values     [][]driver.Value
	fixtureErr error
	errAfter   int
	err        error

	mu         sync.Mutex
	closeCount int
}

// NewRows snapshots columns and values, including byte slices. It accepts nil,
// string, bool, []byte, time.Time, built-in signed/unsigned integers and floats.
// Integers normalize to int64 (unsigned values must fit); float32 to float64.
// Invalid types or column counts cause QueryContext to return ErrInvalidFixture.
func NewRows(columns []string, rows ...[]any) *Rows {
	result := &Rows{
		columns:  append([]string(nil), columns...),
		values:   make([][]driver.Value, len(rows)),
		errAfter: -1,
	}
	for rowIndex, row := range rows {
		if len(row) != len(columns) {
			result.fixtureErr = fmt.Errorf("%w: column/value count mismatch", ErrInvalidFixture)
			return result
		}
		result.values[rowIndex] = make([]driver.Value, len(row))
		for columnIndex, value := range row {
			normalized, err := normalizeValue(value)
			if err != nil {
				result.fixtureErr = err
				return result
			}
			result.values[rowIndex][columnIndex] = normalized
		}
	}
	return result
}

// WithErrorAfter makes iteration fail after rowCount rows have been returned.
// Configuration changes affect future opens only. Concurrent opens take a
// consistent snapshot under the same mutex; already opened iterators are stable.
func (r *Rows) WithErrorAfter(rowCount int, err error) *Rows {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errAfter = rowCount
	r.err = err
	return r
}

// Closed reports whether database/sql closed at least one opened row iterator.
func (r *Rows) Closed() bool { return r.CloseCount() > 0 }

// CloseCount reports how many opened row iterators were closed.
func (r *Rows) CloseCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeCount
}

func (r *Rows) open() (driver.Rows, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fixtureErr != nil {
		return nil, r.fixtureErr
	}
	values := make([][]driver.Value, len(r.values))
	for rowIndex, row := range r.values {
		values[rowIndex] = make([]driver.Value, len(row))
		for columnIndex, value := range row {
			if payload, ok := value.([]byte); ok {
				values[rowIndex][columnIndex] = bytes.Clone(payload)
			} else {
				values[rowIndex][columnIndex] = value
			}
		}
	}
	return &scriptedRows{
		owner:    r,
		columns:  append([]string(nil), r.columns...),
		values:   values,
		errAfter: r.errAfter,
		err:      r.err,
	}, nil
}

type scriptedRows struct {
	owner    *Rows
	columns  []string
	values   [][]driver.Value
	index    int
	errAfter int
	err      error
	closed   bool
}

func (r *scriptedRows) Columns() []string {
	return append([]string(nil), r.columns...)
}

func (r *scriptedRows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.owner.mu.Lock()
	r.owner.closeCount++
	r.owner.mu.Unlock()
	return nil
}

func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.errAfter >= 0 && r.index == r.errAfter {
		if r.err == nil {
			return errors.New("dbtest: scripted rows iteration error")
		}
		return r.err
	}
	if r.index >= len(r.values) {
		return io.EOF
	}
	row := r.values[r.index]
	if len(row) != len(dest) {
		return fmt.Errorf("%w: column/value count mismatch", ErrInvalidFixture)
	}
	copy(dest, row)
	r.index++
	return nil
}

func normalizeValue(value any) (driver.Value, error) {
	switch typed := value.(type) {
	case nil, string, int64, float64, bool, time.Time:
		return typed, nil
	case []byte:
		return bytes.Clone(typed), nil
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case uint:
		return normalizeValue(uint64(typed))
	case uint64:
		if typed > math.MaxInt64 {
			return nil, fmt.Errorf("%w: unsigned integer overflows int64", ErrInvalidFixture)
		}
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case float32:
		return float64(typed), nil
	default:
		return nil, fmt.Errorf("%w: unsupported value type", ErrInvalidFixture)
	}
}
