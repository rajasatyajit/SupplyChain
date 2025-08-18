//go:build integration

package integration

import (
	"unsafe"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajasatyajit/SupplyChain/internal/database"
)

// dpoolAccessor uses unsafe to access the unexported pool for test migrations only.
// This avoids changing production code for test-only needs.
func dpoolAccessor(d *database.DB) *pgxpool.Pool {
	// The database.DB struct layout is known from internal/database/database.go (first field is *pgxpool.Pool)
	type dbLayout struct {
		Pool *pgxpool.Pool
		_    [0]byte // padding not used
	}
	// Convert pointer via unsafe (tests only)
	layout := (*dbLayout)(unsafe.Pointer(d))
	return layout.Pool
}
