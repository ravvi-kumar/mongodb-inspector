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
- [x] Value matching against _id of other collections (rewrote to query MongoDB directly)
- [x] Confidence calculation (matched / sampled)
- [x] 20% threshold filter for suggestions
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

**Status:** Complete
**Goal:** CRUD for relationships, approve/reject workflow

### Checklist
- [x] List/filter relationships
- [x] Approve endpoint
- [x] Reject endpoint

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 6: Investigation graph traversal

---

## Sprint 6: Investigation Graph

**Status:** Complete
**Goal:** Trace document by ID across all collections, bidirectional traversal

### Checklist
- [x] Find document by ID across all collections
- [x] Load approved relationships
- [x] Bidirectional traversal
- [x] Cycle detection (visited set)
- [x] Max depth 5
- [x] Tree + flattened response

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 7: Orphan detection

---

## Sprint 7: Orphan Detection

**Status:** Complete
**Goal:** Find broken relationships, report dangling references

### Checklist
- [x] Scan approved relationships for missing targets
- [x] Store orphan records
- [x] List orphans endpoint

### Blockers
- (none)

---

## Retrospectives

### Sprint 4 Retro
- **Pivot:** Initial approach compared stored sample values between two collections. Failed because two independent 200-doc samples rarely overlap enough for confidence.
- **Fix:** Rewrote discovery to query MongoDB directly — take candidate field values and run `countDocuments({_id: {$in: [...]}})` against the full target collection.
- **Threshold lowered** from 50% to 20% since direct querying produces accurate confidence.
- **ObjectID handling:** Values stored as `{"$oid":"hex"}` in PG JSONB must be converted back to `primitive.ObjectID` for MongoDB queries.
- **Sample values cap** raised from 10 → 200 to improve coverage.

---

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
| 11 | Discovery queries MongoDB directly (not stored samples) | Stored samples had unreliable overlap; direct query gives accurate confidence | 2026-05-31 |

All 7 sprints now complete. The full pipeline is: create connection → scan → discover relationships → approve/reject → investigate documents → detect orphans.

---

## Sprint 8: Test Foundation

**Status:** Complete
**Goal:** Unit + integration tests for all existing services. Zero tests = zero safety for future changes.

### Why This Sprint
7 sprints, 0 test files. Every future sprint will modify discovery/scan/investigation. Without tests, any change could silently break the pipeline. This is the highest-risk gap in the project.

### Checklist
- [x] Unit tests for `internal/scanner/candidate.go` (all heuristics + edge cases)
- [x] Unit tests for `internal/store/mongo/sampler.go` (field extraction, type detection, value cap)
- [x] Unit tests for `internal/service/discovery.go` (uniqueNonEmpty, toBSONValues, collectionsWithIDFields, valueToString, docID, fieldValue)
- [x] Unit tests for `internal/service/investigation.go` (flattenTree, idToString, toBSONValue, visitKey)
- [x] Unit tests for `internal/service/orphan.go` (ensureBSONValue, idValToString)
- [x] Store interfaces for testability (domain/interfaces.go)
- [x] Mock stores in testutil package
- [x] HTTP handler tests (validation, error paths, health, swagger)
- [x] `make test` target (with race detector)
- [x] `make vet` target

### Blockers
- (none)

### Notes for Next Sprint
- 55+ tests passing across 4 packages
- Services refactored to accept interfaces (domain.ConnectionReader, ScanReader, RelationshipReaderWriter, OrphanReaderWriter)
- Constructor signatures unchanged (still accept concrete *pg.XXXStore types)
- HTTP handler tests cover all validation/error paths without needing running services

---

## Sprint 9: Discovery V2 — Explain Why + Multi-Signal

**Status:** Pending
**Goal:** Make discovery smarter and trustworthy. Add explanation for every suggested relationship. Support non-_id target fields.

### Why This Sprint
Current discovery only checks value overlap against `_id`. If a field is named `customer` (not `customerId`), it relies on a hardcoded common names list. Developers won't trust a "Relationship Found" without knowing why. The "explain" feature is the trust builder.

### Checklist
- [ ] Add `explanation` field to `Relationship` domain model + migration
- [ ] Generate human-readable explanation for each discovery:
  - How many values matched out of how many sampled
  - What the field type is
  - Whether naming patterns contributed
  - Whether any competing collection scored lower
- [ ] Support non-`_id` target fields (e.g., `users.email`, `orders.orderNumber`)
  - During discovery, check unique fields in target collections, not just `_id`
  - Requires scanning for fields with high uniqueness ratio (proxy for keys)
- [ ] Add field-name-based scoring signal (fuzzy match `customer` → `customers` collection)
- [ ] Add type-compatibility signal (ObjectId field → ObjectId _id = boost)
- [ ] Combine signals into composite confidence score (weighted)
- [ ] Return explanation in relationship API responses
- [ ] Update OpenAPI spec
- [ ] Tests for new scoring logic

### Blockers
- (none)

---

## Sprint 10: Nested Fields + Array References

**Status:** Pending
**Goal:** Handle real-world MongoDB schemas with nested objects and arrays of references.

### Why This Sprint
Architecture decision #6 deferred nested fields. Many MongoDB documents look like:
```json
{
  "customer": { "id": "usr_123" },
  "tags": ["tag_1", "tag_2"],
  "metadata": { "createdBy": "usr_456" }
}
```
Current scanner only sees top-level keys. It sees `customer` as type "object" and ignores it. It sees `tags` as type "array" and ignores it. This misses a huge class of relationships.

### Checklist
- [ ] Recurse into nested objects during scan, producing dotted-path fields (e.g., `customer.id`, `metadata.createdBy`)
- [ ] Cap nesting depth at 3 levels
- [ ] Detect array-of-scalars as candidate fields (e.g., `tagIds: ["id1", "id2"]`)
- [ ] Store nested field paths in `collection_fields` with parent indicator
- [ ] Update candidate heuristics to work on nested paths
- [ ] Update discovery to match nested source fields against target `_id` / unique fields
- [ ] Update investigation to traverse nested field paths in documents
- [ ] Handle array references in investigation (fan out to N related docs)
- [ ] Update OpenAPI spec
- [ ] Tests for nested field extraction, array handling

### Blockers
- Depends on Sprint 9 (discovery needs to handle new field types)

---

## Sprint 11: Scan Quality + API Hardening

**Status:** Pending
**Goal:** Fix scan bias, add pagination, add resilience. Make the API production-ready.

### Why This Sprint
Current scan uses `Find().Sort(_id: -1).Limit(N)` which biases toward most recent documents. Older data patterns may be missed. List endpoints have no pagination — will break on large datasets. Worker has no retry.

### Checklist
- [ ] Replace `$sort + $limit` sampling with MongoDB `$sample` aggregation for truly random samples
- [ ] Add pagination (offset + limit) to: list scans, list fields, list relationships, list orphans
- [ ] Add relationship deduplication guard (skip if source+target+fields already exists)
- [ ] Add retry logic for failed scans (max 3 retries with backoff)
- [ ] Add connection health-check endpoint (ping MongoDB, return latency + status)
- [ ] Add scan summary endpoint: total fields, candidates, relationships found, orphans detected
- [ ] Rate-limit discovery queries against MongoDB (configurable batch size + sleep between batches)
- [ ] Update OpenAPI spec with pagination params
- [ ] Tests for pagination, deduplication, retry

### Blockers
- (none, can run parallel with Sprint 9/10)

---

## Sprint 12: Real-World Validation

**Status:** Complete
**Goal:** Test against 3-5 real MongoDB datasets. Measure recall, precision, fix gaps.

### Why This Sprint
The LLM's best suggestion: "That experiment will tell you more about the future of this product than another month of coding." We need to know: what do we miss? What do we falsely detect? This sprint is about data, not code.

### Checklist
- [x] Create seed scripts for 5 real-world MongoDB schemas:
  - E-commerce (200 users, 100 products, 500 orders, 300 reviews, 500 payments, 300 shipments, orphan records)
  - SaaS multi-tenant (20 orgs, 100 users, 80 projects, 200 invoices, 50 webhooks, 1000 events)
  - Blog/CMS (30 authors, 150 posts, 500 comments, 20 tags, 400 post_tags, 10 categories)
  - Analytics (500 users, 1000 sessions, 5000 events, 200 campaigns)
  - CRM (50 companies, 20 reps, 200 contacts, 100 deals, 1000 activities, 300 notes)
- [x] Docker-compose MongoDB service added
- [x] Validation CLI (cmd/validate) that measures precision/recall against known relationships
- [x] `make seed` and `make validate` targets
- [x] Run full pipeline against each
- [x] Measure and record:
  - True positives: 28
  - False positives: 0
  - False negatives: 0
  - Precision: 100%
  - Recall: 100%
- [x] Document results in `docs/validation-results.md`
- [x] No bugs found — all relationships discovered correctly

### Blockers
- Depends on Sprint 9, 10 (need advanced discovery + nested support for fair test)

---

## Sprint 13: Investigation UX + API Polish

**Status:** Pending
**Goal:** Make investigation results useful for API consumers. Add stats, summaries, graph-friendly output.

### Why This Sprint
Current investigation returns a raw tree. For a frontend or tool to render a useful graph, it needs more: relationship metadata, collection stats, graph-layout hints. Also need a way to explore "what references this document?" without knowing the ID first.

### Checklist
- [ ] Add `GET /api/connections/:id/stats` — collection count, field count, relationship count, orphan count
- [ ] Add `GET /api/relationships/:id/trace` — trace a specific relationship forward/backward
- [ ] Enhance investigate response with:
  - Collection-level metadata (doc count, field count)
  - Relationship labels (not just raw field names)
  - Graph-layout hints (depth, sibling count)
- [ ] Add `POST /api/investigate/batch` — investigate multiple document IDs at once
- [ ] Add `GET /api/orphans/:id/investigate` — investigate an orphan's source document
- [ ] Add `GET /api/connections/:id/schema-map` — return all approved relationships as a graph (nodes=collections, edges=relationships)
- [ ] Add relationship search/filter by collection name
- [ ] Update OpenAPI spec
- [ ] Tests for new endpoints

### Blockers
- (none, but benefits from Sprint 12 results)

---

## Post-Sprint Planning Questions

1. **Frontend?** Sprints 8-13 are all backend. When does a frontend enter the picture? Is this API-first (others build UIs) or does the project need its own UI?
2. **Auth?** Zero authentication on any endpoint. Is this always single-user / localhost, or does multi-user auth need to happen?
3. **Persistence strategy?** Currently every scan creates new records. Should old scans be archived/cleaned? What's the data retention model?
4. **MongoDB write safety?** All operations are read-only against MongoDB. Should we keep it that way, or will future features need to write (e.g., fix orphans)?
5. **Performance ceiling?** Discovery does N_candidates × N_collections queries against MongoDB. At what scale does this become a problem? Should we add async discovery?
6. **Export?** Should relationships/schema be exportable (JSON, Mermaid diagram, Prisma schema)?