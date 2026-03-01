package database

import (
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

// NewMockPool creates a new pgxmock pool for testing. The returned pool
// satisfies the DBTX interface and can be passed to any repository constructor.
// Call ExpectationsWereMet() at the end of each test to verify all expectations were fulfilled.
func NewMockPool() (pgxmock.PgxPoolIface, error) {
	return pgxmock.NewPool()
}
