package api

import (
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/api"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
)

// paginationFromRequest builds the cursor forwarded to the indexer, substituting defaults when
// the request carries none. A rejected cursor must surface to the caller rather than fall back
// to a default, so a bad sort column is visible instead of silently reordering the results.
func paginationFromRequest(requested *api.PaginationCursor) (*indexer.PaginationCursor, error) {
	cursor := &indexer.PaginationCursor{
		Limit:   persistence.DefaultPageLimit,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if requested != nil {
		cursor = &indexer.PaginationCursor{
			Limit:   requested.GetLimit(),
			Offset:  requested.GetOffset(),
			OrderBy: requested.GetOrderBy(),
		}
	}

	page := &persistence.PaginationCursor{
		Limit:   int(cursor.GetLimit()),
		Offset:  int(cursor.GetOffset()),
		OrderBy: cursor.GetOrderBy(),
	}

	if err := page.Validate(); err != nil {
		return nil, err
	}

	return cursor, nil
}
