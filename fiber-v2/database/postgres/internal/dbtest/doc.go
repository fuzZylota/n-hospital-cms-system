// Package dbtest provides a small, per-test database/sql connector for
// repository tests. It is intended to be imported only by _test.go files.
//
// Each Connector owns its script and recorder. It uses no registered driver,
// network connection, or global script/recorder state. Recorded arguments are
// available to assertions through Events; callers must use synthetic,
// non-sensitive test data because assertion failures may print those values.
//
// This helper observes SQL calls and lifecycle order, not PostgreSQL isolation
// or atomicity. Events do not identify connections/transactions; order alone
// cannot prove that a query used the transaction's connection. A shared Rows
// fixture has separate iterators but one shared close counter.
package dbtest
