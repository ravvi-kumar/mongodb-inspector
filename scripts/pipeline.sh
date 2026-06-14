#!/usr/bin/env bash
set -euo pipefail

MONGO_URI="${1:?Usage: $0 <mongodb-uri> <database> [sample_doc_id]}"
DATABASE="${2:?Usage: $0 <mongodb-uri> <database> [sample_doc_id]}"
DOC_ID="${3:-}"
API="${API_URL:-http://localhost:8080}"

echo "=== MongoDB Inspector Pipeline ==="
echo "MongoDB: $MONGO_URI"
echo "Database: $DATABASE"
echo "API: $API"
echo ""

# Health check
health=$(curl -sf "$API/health" 2>/dev/null) || { echo "ERROR: API not reachable at $API"; exit 1; }

# Create connection
conn=$(curl -sf -X POST "$API/api/connections" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$DATABASE\",\"connection_string\":\"$MONGO_URI\"}")
conn_id=$(echo "$conn" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "[1/6] Connection created: $conn_id"

# Select database
curl -sf -X POST "$API/api/connections/$conn_id/select-db" \
  -H "Content-Type: application/json" \
  -d "{\"database\":\"$DATABASE\"}" > /dev/null
echo "[2/6] Database selected: $DATABASE"

# Start scan
scan=$(curl -sf -X POST "$API/api/scans" \
  -H "Content-Type: application/json" \
  -d "{\"connection_id\":\"$conn_id\",\"sample_size\":1000}")
scan_id=$(echo "$scan" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "[3/6] Scan started: $scan_id"

# Wait for scan
echo -n "      Waiting for scan"
for i in $(seq 1 60); do
  status=$(curl -sf "$API/api/scans/$scan_id" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
  if [ "$status" = "completed" ]; then
    echo " done"
    break
  fi
  if [ "$status" = "failed" ]; then
    echo " FAILED"
    curl -sf "$API/api/scans/$scan_id" | python3 -m json.tool
    exit 1
  fi
  echo -n "."
  sleep 1
done

# Show candidates
cand_count=$(curl -sf "$API/api/scans/$scan_id/candidates" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
echo "      Found $cand_count candidate fields"

# Discover relationships
curl -sf -X POST "$API/api/relationships/discover" \
  -H "Content-Type: application/json" \
  -d "{\"scan_id\":\"$scan_id\"}" > /dev/null
echo "[4/6] Discovery complete"

# List relationships
echo ""
echo "=== Relationships ==="
curl -sf "$API/api/relationships?connection_id=$conn_id" | python3 -c "
import sys, json
response = json.load(sys.stdin)
rels = response['data']
approved = [r for r in rels if r['status'] == 'approved']
suggested = [r for r in rels if r['status'] == 'suggested']
print(f'  Approved: {len(approved)}  Suggested: {len(suggested)}')
print()
for r in rels:
    icon = '  '
    print(f\"{icon} {r['source_collection']}.{r['source_field']} --> {r['target_collection']}.{r['target_field']}  ({r['confidence']*100:.0f}%)  [{r['status']}]\")
"

# Detect orphans
echo ""
echo "[5/6] Detecting orphans..."
orphan_count=$(curl -sf -X POST "$API/api/orphans/detect" \
  -H "Content-Type: application/json" \
  -d "{\"connection_id\":\"$conn_id\"}" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
echo "      Orphans found: $orphan_count"

# Investigate a document
if [ -n "$DOC_ID" ]; then
  echo ""
  echo "[6/6] Investigating document: $DOC_ID"
  curl -sf -X POST "$API/api/investigate" \
    -H "Content-Type: application/json" \
    -d "{\"connection_id\":\"$conn_id\",\"document_id\":\"$DOC_ID\"}" | python3 -c "
import sys, json
result = json.load(sys.stdin)
def print_tree(node, indent=0):
    prefix = '  ' * indent
    rel = f'  ({node[\"relationship\"]})' if node.get('relationship') else ''
    print(f'{prefix}{node[\"collection\"]} [{node[\"id\"]}]{rel}')
    for child in node.get('children', []):
        print_tree(child, indent + 1)
print_tree(result['tree'])
print(f'\nTotal documents: {len(result[\"documents\"])}')
"
else
  echo ""
  echo "[6/6] Skipping investigation (pass a document ID as 3rd arg to investigate)"
fi

echo ""
echo "=== Done ==="
echo "API docs: $API/docs"
echo "Connection ID: $conn_id"
echo "Scan ID: $scan_id"
