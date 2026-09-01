# Wire API compatibility suite

This suite freezes the ColonyOS wire RPC API. It starts the colonies server
binary as a subprocess and exercises the core orchestration flows strictly
through the Go client SDK (`pkg/client`) over HTTP. It never imports server
internals, so internal refactoring cannot silently weaken it.

Rules:

- This suite must pass unmodified through every phase of the modernization.
  If a change makes it fail, the wire API broke: fix the change, not the suite.
- Additions to the suite are welcome. Changing or deleting existing assertions
  requires an explicit wire-API break decision, called out in the pull request.
- The companion guard for message schemas is `pkg/rpc/wire_golden_test.go`.

Running:

    make test-compat

Requirements: Postgres on localhost:5432 (make startdb). The suite uses the
server binary's default database settings (database "postgres", table prefix
PROD_) and resets those tables, so do not point it at a database you care
about. CI runs it against a throwaway service container.

The server binary is built from the current tree by default; set
COLONIES_SERVER_BINARY to test a different binary (for example an older
release) against the same frozen expectations.
