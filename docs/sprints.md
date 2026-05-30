# Sprint Tracker — MongoDB Investigation Engine

---

## Sprint 1: Foundation

**Status:** Complete
**Goal:** Project skeleton, PG store, migrations, health endpoint

### Checklist
- [x] Go module init + directory structure
- [x] Domain models
- [x] Config (env-based)
- [x] PG store with goose migrations (5 tables)
- [x] Chi HTTP server with health endpoint
- [x] Entrypoint (cmd/server/main.go)
- [x] Makefile + .env.example
- [x] Build compiles cleanly
- [x] Health endpoint verified

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 2: MongoDB connector (connect, list databases, list collections, select DB)

---

## Sprint 2: MongoDB Connector

**Status:** Complete
**Goal:** Connect to MongoDB, list databases/collections, select database

### Checklist
- [x] MongoDB connection CRUD in PG store
- [x] MongoDB connector (connect, list DBs, list collections)
- [x] Connection HTTP handlers (CRUD + databases + select-db + collections)
- [x] Routes wired in server.go
- [x] Connection string validation on create (pings MongoDB before saving)
- [x] Scaler/OpenAPI docs (served at /docs and /docs/json)
- [x] Build compiles + vet passes

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 3: Collection scanner (sample docs, extract fields, detect candidates)

---

## Sprint 3: Collection Scanner

**Status:** Complete
**Goal:** Sample documents, extract fields, detect candidate relationships

### Checklist
- [x] Sample N documents from each collection (configurable, default 1000)
- [x] Extract top-level field names, types, sample values
- [x] Store in collection_fields table
- [x] Candidate field detection heuristics (Id/_id suffix, Ref suffix, By suffix, common names, ObjectId type, hex string patterns)
- [x] Async scan with status polling (goroutine worker)
- [x] Scan HTTP handlers (start, list, get, fields, candidates)
- [x] OpenAPI spec updated with scan endpoints
- [x] Build + vet clean

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 4: Relationship discovery (value matching, confidence scoring)

---

## Sprint 4: Relationship Discovery

**Status:** Complete
**Goal:** Match candidate values against _id fields, calculate confidence

### Checklist
- [x] Value matching against _id of other collections
- [x] Confidence calculation (matched / sampled)
- [x] 50% threshold filter
- [x] Store suggested relationships
- [x] Relationship CRUD + approve/reject
- [x] OpenAPI spec updated
- [x] Build + vet clean

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 5: Relationship management (CRUD, approve/reject)

---

## Sprint 5: Relationship Management

**Status:** Not Started
**Goal:** CRUD for relationships, approve/reject workflow

### Checklist
- [ ] List/filter relationships
- [ ] Approve endpoint
- [ ] Reject endpoint

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 6: Investigation graph traversal

---

## Sprint 6: Investigation Graph

**Status:** Not Started
**Goal:** Trace document by ID across all collections, bidirectional traversal

### Checklist
- [ ] Find document by ID across all collections
- [ ] Load approved relationships
- [ ] Bidirectional traversal
- [ ] Cycle detection (visited set)
- [ ] Max depth 5
- [ ] Tree + flattened response

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 7: Orphan detection

---

## Sprint 7: Orphan Detection

**Status:** Not Started
**Goal:** Find broken relationships, report dangling references

### Checklist
- [ ] Scan approved relationships for missing targets
- [ ] Store orphan records
- [ ] List orphans endpoint

### Blockers
- (none)

---

## Retrospectives

### Sprint 1 Retro
- (to be filled after completion)

---

## Architecture Decisions Log

| # | Decision | Rationale | Date |
|---|----------|-----------|------|
| 1 | Single Go binary, internal packages | Simpler for MVP, can split later | 2026-05-30 |
| 2 | chi router | Lightweight, stdlib-compatible | 2026-05-30 |
| 3 | goose for migrations | SQL-first, embeddable | 2026-05-30 |
| 4 | pgx driver for Postgres | Performance, stdlib-compatible | 2026-05-30 |
| 5 | Async scans via goroutine + PG status | No external queue needed for MVP | 2026-05-30 |
| 6 | Top-level fields only for MVP | Simplicity, add nested later | 2026-05-30 |
| 7 | Multiple connections supported | Users often have staging + prod | 2026-05-30 |
| 8 | New scan = new snapshot | Preserve history | 2026-05-30 |
| 9 | Bidirectional investigation | Trace from any direction | 2026-05-30 |
| 10 | Tree + flat list response | Tree for display, flat for API consumers | 2026-05-30 |
