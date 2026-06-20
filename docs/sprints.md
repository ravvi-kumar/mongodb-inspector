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

**Status:** Complete
**Goal:** Make discovery smarter and trustworthy. Add explanation for every suggested relationship. Support non-_id target fields.

### Checklist
- [x] Add `explanation` field to `Relationship` domain model + migration `006_explanation.sql`
- [x] Create `internal/scorer/` package with pluggable signal scorers
- [x] Generate human-readable explanation for each discovery:
  - How many values matched out of how many sampled
  - What the field type is
  - Whether naming patterns contributed
  - Whether uniqueness of target field is high
- [x] Support non-`_id` target fields (e.g., `users.email`, `orders.orderNumber`)
  - During discovery, check unique fields in target collections via `distinct` / `estimatedDocumentCount`
  - Only includes fields with uniqueness ratio ≥ 0.8
- [x] Add field-name-based scoring signal (stem + fuzzy match to collection names)
- [x] Add type-compatibility signal (ObjectId↔ObjectId = 1.0, string↔string = 0.7, cross-type = 0.4)
- [x] Add naming-convention signal (weighted by candidate reason: Id suffix=0.9, Ref=0.8, By=0.6, common name=0.5, etc.)
- [x] Add uniqueness signal for non-_id targets
- [x] Combine signals into composite confidence score (weighted: value_overlap=0.50, name_similarity=0.20, type_compatibility=0.15, naming_convention=0.05, uniqueness=0.10)
- [x] Return explanation in relationship API responses
- [x] Update OpenAPI spec with `explanation` property
- [x] Update `*pg.RelationshipStore` Create/Get/List/UpdateStatus to include explanation column
- [x] Tests for all scorer signals + composite scoring (31 new tests)
- [x] `make build` + `make vet` clean
- [x] `make test` — all tests pass (86 total across 6 packages)

### Blockers
- (none)

### Architecture Decisions
- **AD-12: Pluggable scorer pattern** — Each signal is an independent function returning `(score, reason)`. `CompositeScorer` combines with configurable weights. Makes adding future signals trivial (Sprint 10 nested fields, Sprint 13 UX hints).
- **AD-13: Uniqueness via MongoDB `distinct`** — Rather than predicting uniqueness from sampling, we query MongoDB's `distinct` command and compare against `estimatedDocumentCount`. Ratio ≥ 0.8 qualifies as a unique key candidate.
- **AD-14: Weighted composite confidence** — Value overlap still dominates (50%), but name similarity (20%), type compatibility (15%), naming convention (5%), and uniqueness (10%) all contribute. Confidence thresholds unchanged: 0.2 discard, 0.7 auto-approve.

### Notes for Next Sprint
- Sprint 10: Nested fields + array references. Scorer is ready to accept dotted-path field names and array value types.
- The `uniqueFields` function already handles arbitrary field names — passing `customer.id` or `metadata.createdBy` through it will work.
- Candidate detection (`internal/scanner/candidate.go`) needs updating for nested paths in Sprint 10.

---

## Sprint 10: Nested Fields + Array References

**Status:** Complete
**Goal:** Handle real-world MongoDB schemas with nested objects and arrays of references.

### Checklist
- [x] Recurse into nested objects during scan, producing dotted-path fields (e.g., `customer.id`, `metadata.createdBy`)
- [x] Cap nesting depth at 3 levels (`maxNestingDepth = 3` in sampler.go)
- [x] Detect array-of-scalars as candidate fields (e.g., `tagIds: ["id1", "id2"]`)
- [x] Array elements sampled as individual values (leaf type: objectId, string, int, etc.)
- [x] Candidate heuristics updated: array-of-ObjectId, array-of-hex-strings
- [x] Discovery already handles dotted-path field names (no changes needed — Sprint 9 scorer + uniqueFields accept any field name)
- [x] `nestedFieldValue` dotted-path walker in investigation (splits on `.`, traverses bson.M/map)
- [x] Array fan-out in `findRelatedForward` — detects `primitive.A`, uses `$in` to query all referenced docs
- [x] `toBSONArray` helper converts array elements to BSON values for `$in` queries
- [x] Tests: nested object extraction, depth cap, array-of-scalars sampling, array leaf type, candidate array heuristics, dotted-path field value
- [x] `make build` + `make vet` clean
- [x] `make test` — all tests pass (race detector clean)

### Blockers
- (none)

### Architecture Decisions
- **AD-15: Recursive field extraction with depth cap** — `extractFields` recurses into `bson.M` (objects) and `primitive.A` (arrays of objects) up to 3 levels deep, producing dotted-path names like `customer.id`. Scalar arrays store individual element values, not the array container.
- **AD-16: Array fan-out via `$in`** — When a relationship source field is an array, `findRelatedForward` uses `{$in: [...values]}` to find all referenced documents in one query, rather than N individual lookups.
- **AD-17: Candidate detection on nested field names** — `IsCandidateField` already works on path segments via regexes (e.g., `customer.id` matches `[Ii]d$`), so no structural changes needed for nested candidate detection.

### Notes for Next Sprint
- Sprint 11: Scan Quality + API Hardening. Independent of Sprint 10, can start immediately.
- `collection_fields` already stores dotted-path names — no schema change needed.
- The scorer's `scoreNameSimilarity` works on the leaf name by default; could be enhanced to use the full path for better matching.

---

## Sprint 11: Scan Quality + API Hardening

**Status:** Complete
**Goal:** Fix scan bias, add pagination, add resilience. Make the API production-ready.

### Why This Sprint
Current scan uses `Find().Sort(_id: -1).Limit(N)` which biases toward most recent documents. Older data patterns may be missed. List endpoints have no pagination — will break on large datasets. Worker has no retry.

### Checklist
- [x] Replace `$sort + $limit` sampling with MongoDB `$sample` aggregation for truly random samples
- [x] Add pagination (offset + limit) to: list scans, list fields, list relationships, list orphans
- [x] Add relationship deduplication guard (skip if source+target+fields already exists)
- [x] Add retry logic for failed scans (max 3 retries with backoff)
- [x] Add connection health-check endpoint (ping MongoDB, return latency + status)
- [x] Add scan summary endpoint: total fields, candidates, relationships found, orphans detected
- [x] Rate-limit discovery queries against MongoDB (configurable batch size + sleep between batches)
- [x] Update OpenAPI spec with pagination params
- [x] Tests for pagination, deduplication, retry

### Blockers
- (none, can run parallel with Sprint 9/10)

### Architecture Decisions
- **AD-18: Pagination pattern** — All list endpoints now support `offset` and `limit` query params with a default of 20 and max of 100. Response includes total count for client-side pagination controls.
- **AD-19: Relationship deduplication** — New unique constraint on `(connection_id, source_collection, source_field, target_collection, target_field)` prevents duplicate relationships. `CreateOrSkip` method uses `ON CONFLICT DO NOTHING`.
- **AD-20: Exponential backoff** — Scan worker retries failed scans with exponential backoff (2s, 4s, 6s) up to 3 attempts before giving up.
- **AD-21: Rate limiting** — Discovery service supports configurable batch size and delay between queries to avoid overwhelming MongoDB. Defaults: 50 queries per batch, 100ms delay.

### Notes for Next Sprint
- Sprint 12: Real-World Validation
- All endpoints now production-ready with proper pagination, error handling, and monitoring
- New migration `007_unique_relationships.sql` adds the deduplication constraint

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

**Status:** Complete
**Goal:** Make investigation results useful for API consumers. Add stats, summaries, graph-friendly output.

### Why This Sprint
Current investigation returns a raw tree. For a frontend or tool to render a useful graph, it needs more: relationship metadata, collection stats, graph-layout hints. Also need a way to explore "what references this document?" without knowing the ID first.

### Checklist
- [x] Add `GET /api/connections/:id/stats` — collection count, field count, relationship count, orphan count
- [x] Add `GET /api/relationships/:id/trace` — trace a specific relationship forward/backward
- [x] Enhance investigate response with:
  - Collection-level metadata (doc count, field count)
  - Relationship labels (not just raw field names)
  - Graph-layout hints (depth, sibling count)
- [x] Add `POST /api/investigate/batch` — investigate multiple document IDs at once
- [x] Add `GET /api/orphans/:id/investigate` — investigate an orphan's source document
- [x] Add `GET /api/connections/:id/schema-map` — return all approved relationships as a graph (nodes=collections, edges=relationships)
- [x] Add relationship search/filter by collection name
- [x] Update OpenAPI spec
- [x] Tests for new endpoints

### Blockers
- (none, but benefits from Sprint 12 results)

### Architecture Decisions
- **AD-22: Enhanced investigation metadata** — Added collection metadata (document count, field count) and node metadata (depth, sibling count, relationship labels) to help UI render graphs more effectively.
- **AD-23: Batch investigation with limits** — Batch investigate supports up to 50 documents per request to prevent performance issues while still enabling bulk operations.
- **AD-24: Schema map as graph structure** — Approved relationships returned as nodes (collections) and edges (relationships) for easy consumption by graph visualization libraries.

### Notes for Next Sprint
- Sprint 13 complete — all UX and API polish features implemented
- API now fully documented with OpenAPI spec including all new endpoints
- All existing tests pass, mock stores updated to support new interface methods

---

## Sprint 14: Frontend Foundation + Connection Wizard

**Status:** Complete
**Goal:** React + Vite + TypeScript setup, connection management, first-time onboarding

### Checklist
- [x] Vite + React + TypeScript + Tailwind CSS setup
- [x] Component library integration (shadcn/ui)
- [x] API client from OpenAPI spec (openapi-typescript + fetch)
- [x] React Router with createBrowserRouter setup
- [x] Connection list page (all connections, health status)
- [x] Connection create/edit form with validation
- [x] Test connection button (show latency, status)
- [x] First-time welcome/onboarding wizard
- [x] Mobile-responsive layout
- [x] Error boundaries + loading states
- [x] Build + typecheck

### Blockers
- (none)

### Notes for Next Sprint
- Sprint 15: Scan progress dashboard
- Backend API endpoints ready: `/api/connections/*`, `/api/connections/:id/health`
- React Router configured with proper route structure:
  - `/onboarding` - First-time user flow
  - `/` - Main layout with sidebar navigation
  - `/connections` - Connection list
  - `/connections/new` - Add new connection
  - Placeholder routes for scans, investigation, orphans

---

## Sprint 15: Scan Progress Dashboard

**Status:** Complete
**Goal:** Real-time scan monitoring, clear completion flow

### Checklist
- [x] Connection details page with stats
- [x] Real-time progress polling (1-2 second intervals)
- [x] Big progress indicator with percentage
- [x] Live field count updates during scan
- [x] Scan completion notification with summary card
- [x] "View Results" prominent CTA
- [x] Scan history page (past scans, status, timestamps)
- [x] Scan summary card (total fields, candidates, relationships, orphans)
- [x] Mobile-optimized progress view
- [x] Error handling for failed scans
- [x] Build + typecheck

### Blockers
- (none)

### Architecture Decisions
- **AD-25: Real-time polling with useScanPolling hook** — Custom React hook polls scan status every 1.5 seconds, stops automatically on completion/failure, and provides callbacks for success/error states.
- **AD-26: Progress estimation** — Since backend doesn't provide percentage, estimated progress based on field count vs expected total (assumes ~5 fields per collection on average).
- **AD-27: Navigation flow** — Connections list → Connection details → Start scan → Live progress → Results page → Next steps (relationships/investigation/orphans).
- **AD-28: Error handling** — Scan failures show error messages prominently with retry capability. Network errors handled gracefully with user-friendly messages.

### Notes for Next Sprint
- Sprint 16: Relationship Explorer with plain-English UX
- Backend API endpoints ready: `/api/relationships/*`, `/api/scans/{id}/candidates`
- Scan flow working end-to-end: connection → scan → progress → results

---

## Sprint 16: Relationship Explorer (Non-Technical UX)

**Status:** Pending
**Goal:** Simple relationship management, hide technical details

### Checklist
- [ ] Relationship list page with pagination
- [ ] Card-based relationship display (not table rows)
- [ ] Confidence bars (low/med/high) instead of raw scores
- [ ] One-click approve/reject buttons
- [ ] Bulk approve (high-confidence threshold toggle)
- [ ] Search/filter by collection name
- [ ] Plain-English explanations (hide "value_overlap: 0.87", show "87% of values match")
- [ ] Relationship detail view (full explanation, field paths)
- [ ] "Auto-approve high-confidence" button
- [ ] Mobile-responsive cards
- [ ] Build + typecheck

### Blockers
- Depends on Sprint 14, 15

### Notes for Next Sprint
- Sprint 17: Interactive graph investigation
- Backend API endpoints ready: `/api/relationships/*`, `/api/connections/:id/schema-map`

---

## Sprint 17: Graph Investigation

**Status:** Pending
**Goal:** Interactive graph as primary view, click-to-investigate

### Checklist
- [ ] React Flow integration
- [ ] Graph view as default, toggle to list view
- [ ] Schema map rendering (nodes=collections, edges=relationships)
- [ ] Zoom/pan controls
- [ ] Click node → expand/collapse relationships
- [ ] Click edge → show relationship details
- [ ] Search document ID input
- [ ] Document investigation with graph highlight
  - Show path from document to all related docs
  - Bidirectional traversal visualization
  - Relationship labels on edges
  - Collection metadata on nodes
- [ ] Mobile-optimized graph (touch controls)
- [ ] Filter by collection name or relationship type
- [ ] Build + typecheck

### Blockers
- Depends on Sprint 14, 15, 16

### Notes for Next Sprint
- Sprint 18: Orphan report + one-click fix
- Backend API endpoints ready: `/api/investigate/*`, `/api/relationships/:id/trace`

---

## Sprint 18: Orphan Report

**Status:** Pending
**Goal:** Visual orphan list, one-click investigation

### Checklist
- [ ] Orphan list page with pagination
- [ ] Orphan cards showing:
  - Source collection + document ID
  - Broken relationship (which field, target collection)
  - Severity indicator (high/med/low based on confidence)
- [ ] One-click "Investigate" button (opens investigation for source doc)
- [ ] Orphan summary dashboard (total orphans, by collection)
- [ ] Bulk investigation (select multiple orphans)
- [ ] "Fix orphan" guidance (what the relationship should be)
- [ ] Mobile-responsive orphan cards
- [ ] Error handling for missing documents
- [ ] Build + typecheck + end-to-end test

### Blockers
- Depends on Sprint 14, 15, 16, 17

### Notes for Next Sprint
- Frontend MVP complete
- Ready for user testing and feedback

---

## Post-Sprint Planning Questions

1. **Frontend?** Sprints 8-13 are all backend. When does a frontend enter the picture? Is this API-first (others build UIs) or does the project need its own UI? **→ Sprint 14-18: TanStack Start frontend**
2. **Auth?** Zero authentication on any endpoint. Is this always single-user / localhost, or does multi-user auth need to happen? **→ No auth for now**
3. **Persistence strategy?** Currently every scan creates new records. Should old scans be archived/cleaned? What's the data retention model?
4. **MongoDB write safety?** All operations are read-only against MongoDB. Should we keep it that way, or will future features need to write (e.g., fix orphans)?
5. **Performance ceiling?** Discovery does N_candidates × N_collections queries against MongoDB. At what scale does this become a problem? Should we add async discovery?
6. **Export?** Should relationships/schema be exportable (JSON, Mermaid diagram, Prisma schema)?