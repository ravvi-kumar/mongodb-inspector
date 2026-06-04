# Validation Results — Sprint 12

**Date:** 2026-06-05
**Seed Data:** 5 datasets, ~10K+ documents total
**Pipeline:** connect → scan → discover → approve → investigate → orphan

## Summary

| Metric | Value |
|--------|-------|
| True Positives | 28 |
| False Positives | 0 |
| False Negatives | 0 |
| **Precision** | **100%** |
| **Recall** | **100%** |

## Per-Dataset Results

### E-commerce (8 collections, ~2K documents)

| Expected | Found | Status |
|----------|-------|--------|
| orders.userId → users._id | 100% confidence | TP |
| reviews.userId → users._id | 100% confidence | TP |
| reviews.productId → products._id | 100% confidence | TP |
| payments.orderId → orders._id | 100% confidence | TP |
| shipments.orderId → orders._id | 100% confidence | TP |

5/5 found. 0 false positives.

### SaaS Multi-Tenant (6 collections, ~2.5K documents)

| Expected | Found | Status |
|----------|-------|--------|
| users.organizationId → organizations._id | 100% confidence | TP |
| projects.organizationId → organizations._id | 100% confidence | TP |
| invoices.organizationId → organizations._id | 100% confidence | TP |
| webhooks.organizationId → organizations._id | 100% confidence | TP |
| events.userId → users._id | 100% confidence | TP |
| events.projectId → projects._id | 100% confidence | TP |

6/6 found. 0 false positives.

### Blog/CMS (6 collections, ~1.4K documents)

| Expected | Found | Status |
|----------|-------|--------|
| posts.authorId → authors._id | 100% confidence | TP |
| comments.postId → posts._id | 100% confidence | TP |
| comments.authorId → authors._id | 100% confidence | TP |
| post_tags.postId → posts._id | 100% confidence | TP |
| post_tags.tagId → tags._id | 100% confidence | TP |

5/5 found. 0 false positives.

### Analytics (4 collections, ~6.7K documents)

| Expected | Found | Status |
|----------|-------|--------|
| sessions.userId → users._id | 100% confidence | TP |
| events.sessionId → sessions._id | 100% confidence | TP |
| campaigns.userId → users._id | 100% confidence | TP |

3/3 found. 0 false positives.

### CRM (6 collections, ~1.7K documents)

| Expected | Found | Status |
|----------|-------|--------|
| contacts.companyId → companies._id | 100% confidence | TP |
| deals.companyId → companies._id | 100% confidence | TP |
| deals.contactId → contacts._id | 100% confidence | TP |
| deals.ownerId → users._id | 100% confidence | TP |
| activities.contactId → contacts._id | 100% confidence | TP |
| activities.dealId → deals._id | 100% confidence | TP |
| activities.userId → users._id | 100% confidence | TP |
| notes.contactId → contacts._id | 100% confidence | TP |
| notes.userId → users._id | 100% confidence | TP |

9/9 found. 0 false positives.

## Analysis

### What works well
- All 28 expected relationships discovered with 100% confidence
- Zero false positives across all datasets
- Candidate detection correctly identifies all *Id, *By, and common reference field patterns
- Direct MongoDB query approach produces accurate confidence scores
- Auto-approve at 70% threshold is safe — all relationships scored 100%

### Limitations tested
- All relationships follow naming conventions (userId, orderId, etc.)
- All references point to `_id` fields only
- No nested field references tested
- No array-of-references tested
- Seed data uses ObjectIDs consistently (no mixed types)

### What to test next
1. **Non-convention fields** — `customer`, `owner`, `createdBy` (partial coverage in candidate heuristics)
2. **Non-_id targets** — references to unique fields like `users.email`
3. **Nested fields** — `customer.id`, `metadata.ownerId`
4. **Array references** — `tagIds: ["id1", "id2"]`
5. **Mixed types** — string IDs referencing ObjectID _id fields
6. **Sparse references** — fields that only sometimes contain references
7. **Real-world messy data** — inconsistent schemas, null values, missing fields
