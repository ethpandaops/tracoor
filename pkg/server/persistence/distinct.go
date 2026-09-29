package persistence

import (
	"context"
	"fmt"
)

// Distinct dropdown values are resolved one field at a time rather than through a joint
// GROUP BY across every requested field. The joint scan built a temporary b-tree over the
// whole table (~500ms at 1M rows), and its LIMIT bounded joint groups rather than per-field
// values, so individual fields were truncated arbitrarily before deduplication.
//
// Fields whose backing index leads (network, <field>, ...) are walked with a recursive-CTE
// loose index scan: each recursion step is a single index seek, so cost is proportional to
// the number of distinct values rather than the number of rows. The CTE syntax used here
// (WITH RECURSIVE, correlated scalar subqueries, LIMIT) is standard SQL and runs unchanged
// on both SQLite and Postgres. Every other field falls back to a plain per-field
// SELECT DISTINCT, which still beats the joint GROUP BY and no longer truncates other
// fields' values.
//
// Table and column identifiers interpolated into these queries only ever come from the
// hardcoded Key* constants and the distinctTable definitions below — never from
// user-supplied strings. Filter values (the network) are always bound parameters.

// distinctValuesLimit caps the number of values returned per field, matching the LIMIT the
// joint GROUP BY used to apply.
const distinctValuesLimit = 1000

// distinctStrategy says how one column's distinct values are fetched.
type distinctStrategy int

const (
	// distinctLooseScan walks an index leading (network, <column>, ...) via a recursive CTE,
	// one seek per distinct value.
	distinctLooseScan distinctStrategy = iota
	// distinctFullScan reads the column with a plain SELECT DISTINCT.
	distinctFullScan
)

// distinctTable binds a table name to the strategy for each column a caller may request.
// The field map doubles as a whitelist: anything absent is rejected before it can reach SQL.
type distinctTable struct {
	name   string
	fields map[string]distinctStrategy
}

// quoteIdent double-quotes an identifier. Double quotes are the standard SQL quoting form,
// accepted by both SQLite and Postgres, and required for reserved words such as "index".
func quoteIdent(ident string) string {
	return `"` + ident + `"`
}

// distinctFieldValues resolves the distinct values of a single column, choosing the
// strategy declared by the table definition.
func (i *Indexer) distinctFieldValues(ctx context.Context, table distinctTable, field, network string) ([]any, error) {
	strategy, ok := table.fields[field]
	if !ok {
		return nil, fmt.Errorf("unknown distinct field %q for table %s", field, table.name)
	}

	if strategy == distinctFullScan {
		return i.distinctFullScanValues(ctx, table.name, field, network)
	}

	if network != "" {
		// A network-prefixed loose scan also covers field == network: the anchor returns the
		// filtered network itself and the recursive step immediately terminates.
		return i.distinctLooseScanValues(ctx, table.name, field, &network)
	}

	if field == KeyNetwork {
		return i.distinctLooseScanValues(ctx, table.name, field, nil)
	}

	// No network filter on a network-prefixed index: loose-scan the networks first and union
	// the per-network scans. Each sub-scan stays on the index and the network count is tiny,
	// so this remains index-only where a joint DISTINCT would walk the whole table — and it
	// reuses the exact per-network query already proven correct above.
	networks, err := i.distinctLooseScanValues(ctx, table.name, KeyNetwork, nil)
	if err != nil {
		return nil, err
	}

	seen := make(map[any]bool)
	values := make([]any, 0)

	for _, n := range networks {
		nw, ok := n.(string)
		if !ok {
			continue
		}

		perNetwork, err := i.distinctLooseScanValues(ctx, table.name, field, &nw)
		if err != nil {
			return nil, err
		}

		for _, v := range perNetwork {
			if seen[v] {
				continue
			}

			seen[v] = true

			values = append(values, v)
			if len(values) >= distinctValuesLimit {
				return values, nil
			}
		}
	}

	return values, nil
}

// distinctLooseScanValues walks the distinct values of a column through its backing index.
// With a network the scan stays inside that index prefix; without one it walks the leading
// column itself (only ever used for the network column). The recursion terminates when
// MIN() returns NULL, and NULL rows are filtered from the result.
func (i *Indexer) distinctLooseScanValues(ctx context.Context, tableName, column string, network *string) ([]any, error) {
	var (
		query string
		args  []any
	)

	if network != nil {
		query = fmt.Sprintf(`WITH RECURSIVE vals(v) AS (
			SELECT (SELECT MIN(%[1]s) FROM %[2]s WHERE network = ?)
			UNION ALL
			SELECT (SELECT MIN(%[1]s) FROM %[2]s WHERE network = ? AND %[1]s > v) FROM vals WHERE v IS NOT NULL
		)
		SELECT v FROM vals WHERE v IS NOT NULL LIMIT %[3]d`,
			quoteIdent(column), quoteIdent(tableName), distinctValuesLimit)
		args = []any{*network, *network}
	} else {
		query = fmt.Sprintf(`WITH RECURSIVE vals(v) AS (
			SELECT (SELECT MIN(%[1]s) FROM %[2]s)
			UNION ALL
			SELECT (SELECT MIN(%[1]s) FROM %[2]s WHERE %[1]s > v) FROM vals WHERE v IS NOT NULL
		)
		SELECT v FROM vals WHERE v IS NOT NULL LIMIT %[3]d`,
			quoteIdent(column), quoteIdent(tableName), distinctValuesLimit)
	}

	return i.scanDistinctRows(ctx, query, args...)
}

// distinctFullScanValues reads a column's distinct values with a plain DISTINCT scan, for
// columns with no index leading (network, <column>, ...). NULLs are excluded: they are not
// meaningful dropdown values (only the nullable execution bad block columns can hold them).
func (i *Indexer) distinctFullScanValues(ctx context.Context, tableName, column, network string) ([]any, error) {
	var (
		query string
		args  []any
	)

	if network != "" {
		query = fmt.Sprintf(`SELECT DISTINCT %[1]s FROM %[2]s WHERE network = ? AND %[1]s IS NOT NULL LIMIT %[3]d`,
			quoteIdent(column), quoteIdent(tableName), distinctValuesLimit)
		args = []any{network}
	} else {
		query = fmt.Sprintf(`SELECT DISTINCT %[1]s FROM %[2]s WHERE %[1]s IS NOT NULL LIMIT %[3]d`,
			quoteIdent(column), quoteIdent(tableName), distinctValuesLimit)
	}

	return i.scanDistinctRows(ctx, query, args...)
}

// scanDistinctRows runs a single-column query and collects the non-NULL values.
func (i *Indexer) scanDistinctRows(ctx context.Context, query string, args ...any) ([]any, error) {
	rows, err := i.db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]any, 0)

	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}

		if v == nil {
			continue
		}

		values = append(values, v)
	}

	return values, rows.Err()
}

// distinctStrings converts scanned values to strings, dropping anything else.
func distinctStrings(values []any) []string {
	out := make([]string, 0, len(values))

	for _, v := range values {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}

	return out
}

// distinctUint64s converts scanned values to uint64s. The drivers hand integer columns back
// as int64 (SQLite has no unsigned type); anything else is dropped.
func distinctUint64s(values []any) []uint64 {
	out := make([]uint64, 0, len(values))

	for _, v := range values {
		if n, ok := v.(int64); ok {
			//nolint:gosec // not worried about int64 overflow here
			out = append(out, uint64(n))
		}
	}

	return out
}

// distinctInt64s converts scanned values to int64s, dropping anything else.
func distinctInt64s(values []any) []int64 {
	out := make([]int64, 0, len(values))

	for _, v := range values {
		if n, ok := v.(int64); ok {
			out = append(out, n)
		}
	}

	return out
}
