// Command migrate-announcement-ref is the one-shot, idempotent cutover for
// error-node rows: nrv:error_nodes: -> graph:error_nodes:, and nrvSource /
// nrv_source -> announcementRef inside each row. Test-suite rows move in the
// same run: nrv:error_tests: -> graph:error_tests:, with separate counts. It
// refuses to run twice. A run that skips any row does not write the marker and
// exits 4, so the legacy rows stay tracked; fix or remove the skipped rows and
// run again.
//
// Usage:
//
//	migrate-announcement-ref --db <store-dir> [--engine bluntdb|leveldb]
//
// --db is required and has no default. The store directory must already exist;
// it is never created. The command prints the path it opens before touching it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// Exit codes: 0 migrated, 1 error, 2 usage or path refused, 3 already migrated,
// 4 rows skipped (marker not written; repeat after fixing them).
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("migrate-announcement-ref", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "store directory to migrate (required; no default)")
	engine := fs.String("engine", "bluntdb", "store engine: bluntdb (the KNIRVGRAPH backend) or leveldb")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dbPath == "" {
		fmt.Fprintln(stderr, "migrate-announcement-ref: --db is required; refusing to choose a store for you")
		return 2
	}
	info, err := os.Stat(*dbPath)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "migrate-announcement-ref: %s is not an existing directory; refusing to create a store\n", *dbPath)
		return 2
	}

	fmt.Fprintf(stdout, "opening %s store: %s\n", *engine, *dbPath)
	kv, closeStore, err := openStore(*engine, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "migrate-announcement-ref: open store: %v\n", err)
		return 1
	}
	defer closeStore()

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if errors.Is(err, locator.ErrAnnouncementRefMigrationDone) {
		fmt.Fprintf(stderr, "refused: marker %q is already present in %s; no changes made\n", locator.AnnouncementRefMigrationMarker, *dbPath)
		return 3
	}
	if err != nil {
		fmt.Fprintf(stderr, "migrate-announcement-ref: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "scanned=%d rewritten=%d already-new=%d skipped-unreadable=%d\n",
		rep.Scanned, rep.Rewritten, rep.AlreadyNew, rep.SkippedUnreadable)
	fmt.Fprintf(stdout, "old-field-rows=%d duplicates-removed=%d skipped-conflict=%d\n",
		rep.OldFieldRows, rep.DuplicatesRemoved, rep.SkippedConflict)
	fmt.Fprintf(stdout, "test-suites: scanned=%d rewritten=%d already-new=%d duplicates-removed=%d skipped-unreadable=%d skipped-conflict=%d superseded-removed=%d\n",
		rep.SuiteScanned, rep.SuiteRewritten, rep.SuiteAlreadyNew, rep.SuiteDuplicatesRemoved,
		rep.SuiteSkippedUnreadable, rep.SuiteSkippedConflict, rep.SuiteSupersededRemoved)
	for _, key := range rep.Skipped {
		fmt.Fprintf(stdout, "skipped: %s\n", key)
	}
	if !rep.MarkerWritten {
		fmt.Fprintf(stderr, "marker %q NOT written: %d row(s) skipped and left in place; fix or remove them and run again\n",
			locator.AnnouncementRefMigrationMarker, len(rep.Skipped))
		return 4
	}
	fmt.Fprintf(stdout, "marker %q written\n", locator.AnnouncementRefMigrationMarker)
	return 0
}

func openStore(engine, path string) (locator.MigrationKV, func(), error) {
	switch engine {
	case "bluntdb":
		s, err := storage.NewBluntDBStorage(path)
		if err != nil {
			return nil, nil, err
		}
		return s, func() { _ = s.Close() }, nil
	case "leveldb":
		s, err := storage.NewLevelDBStorage(path)
		if err != nil {
			return nil, nil, err
		}
		return s, func() { _ = s.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("unknown --engine %q (want bluntdb or leveldb)", engine)
	}
}
