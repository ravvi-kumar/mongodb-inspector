package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

const maxDepth = 5

type InvestigationService struct {
	connStore   domain.ConnectionReader
	relStore    domain.RelationshipReaderWriter
	orphanStore domain.OrphanReaderWriter
}

func NewInvestigationService(connStore *pg.ConnectionStore, relStore *pg.RelationshipStore, orphanStore *pg.OrphanStore) *InvestigationService {
	return &InvestigationService{connStore: connStore, relStore: relStore, orphanStore: orphanStore}
}

func (s *InvestigationService) Investigate(ctx context.Context, connectionID string, documentID string) (*domain.InvestigateResult, error) {
	conn, err := s.connStore.Get(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	rels, err := s.relStore.GetApproved(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get approved relationships: %w", err)
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	db := mongoConn.Database(conn.Database)

	visited := make(map[visitKey]bool)

	rootDoc, rootColl, err := findDocumentAcrossCollections(ctx, db, documentID)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}
	if rootDoc == nil {
		return nil, fmt.Errorf("document %s not found in any collection", documentID)
	}

	log.Printf("investigation: found document %s in collection %s", documentID, rootColl)

	visitK := visitKey{collection: rootColl, id: documentID}
	visited[visitK] = true

	tree := &domain.InvestigationNode{
		Collection: rootColl,
		ID:         documentID,
		Document:   rootDoc,
		Metadata: &domain.NodeMetadata{
			Depth: 0,
		},
	}

	traverse(ctx, db, tree, rootColl, documentID, rels, visited, 0)

	var flat []domain.FlatDocument
	flattenTree(tree, &flat)

	collectionMetadata, err := s.getCollectionMetadata(ctx, db, rootColl)
	if err != nil {
		log.Printf("warning: failed to get collection metadata: %v", err)
		collectionMetadata = &domain.CollectionMetadata{}
	}

	return &domain.InvestigateResult{
		Root:      *tree,
		Tree:      tree,
		Documents: flat,
		Metadata:  collectionMetadata,
	}, nil
}

func (s *InvestigationService) getCollectionMetadata(ctx context.Context, db *mongo.Database, collectionName string) (*domain.CollectionMetadata, error) {
	count, err := db.Collection(collectionName).EstimatedDocumentCount(ctx)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]struct{})
	cursor, err := db.Collection(collectionName).Find(ctx, bson.M{}, options.Find().SetLimit(100).SetProjection(bson.M{"_id": 0}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		extractFieldsFromDoc(doc, "", fields)
	}

	return &domain.CollectionMetadata{
		DocumentCount: int(count),
		FieldCount:    len(fields),
	}, nil
}

func extractFieldsFromDoc(doc bson.M, prefix string, fields map[string]struct{}) {
	for key, val := range doc {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		fields[fullKey] = struct{}{}

		if nested, ok := val.(bson.M); ok {
			extractFieldsFromDoc(nested, fullKey, fields)
		}
	}
}

func traverse(
	ctx context.Context,
	db *mongo.Database,
	parent *domain.InvestigationNode,
	collection string,
	documentID string,
	rels []domain.Relationship,
	visited map[visitKey]bool,
	depth int,
) {
	if depth >= maxDepth {
		return
	}

	for _, rel := range rels {
		if rel.SourceCollection == collection {
			childDocs, err := findRelatedForward(ctx, db, parent.Document, rel)
			if err != nil {
				log.Printf("warning: forward lookup failed for %s.%s: %v", rel.SourceCollection, rel.SourceField, err)
				continue
			}
			for _, child := range childDocs {
				childID := docID(child)
				if childID == "" {
					continue
				}
				vk := visitKey{collection: rel.TargetCollection, id: childID}
				if visited[vk] {
					continue
				}
				visited[vk] = true

				node := &domain.InvestigationNode{
					Collection:   rel.TargetCollection,
					ID:           childID,
					Document:     child,
					Relationship: fmt.Sprintf("%s.%s → %s.%s", rel.SourceCollection, rel.SourceField, rel.TargetCollection, rel.TargetField),
					Metadata: &domain.NodeMetadata{
						Depth:             depth + 1,
						SiblingCount:      len(childDocs),
						RelationshipLabel: fmt.Sprintf("has %s in %s", rel.TargetField, rel.TargetCollection),
					},
				}
				parent.Children = append(parent.Children, node)
				traverse(ctx, db, node, rel.TargetCollection, childID, rels, visited, depth+1)
			}
		}

		if rel.TargetCollection == collection {
			childDocs, err := findRelatedReverse(ctx, db, documentID, rel)
			if err != nil {
				log.Printf("warning: reverse lookup failed for %s.%s: %v", rel.TargetCollection, rel.TargetField, err)
				continue
			}
			for _, child := range childDocs {
				childID := docID(child)
				if childID == "" {
					continue
				}
				vk := visitKey{collection: rel.SourceCollection, id: childID}
				if visited[vk] {
					continue
				}
				visited[vk] = true

				node := &domain.InvestigationNode{
					Collection:   rel.SourceCollection,
					ID:           childID,
					Document:     child,
					Relationship: fmt.Sprintf("%s.%s ← %s.%s", rel.TargetCollection, rel.TargetField, rel.SourceCollection, rel.SourceField),
					Metadata: &domain.NodeMetadata{
						Depth:             depth + 1,
						SiblingCount:      len(childDocs),
						RelationshipLabel: fmt.Sprintf("referenced by %s in %s", rel.SourceField, rel.SourceCollection),
					},
				}
				parent.Children = append(parent.Children, node)
				traverse(ctx, db, node, rel.SourceCollection, childID, rels, visited, depth+1)
			}
		}
	}
}

func findRelatedForward(ctx context.Context, db *mongo.Database, doc any, rel domain.Relationship) ([]bson.M, error) {
	val := nestedFieldValue(doc, rel.SourceField)
	if val == nil {
		return nil, nil
	}

	if arr, ok := val.(primitive.A); ok {
		bsonVals := toBSONArray(arr)
		if len(bsonVals) == 0 {
			return nil, nil
		}
		bsonFilter := bson.M{rel.TargetField: bson.M{"$in": bsonVals}}
		return queryDocs(ctx, db, rel.TargetCollection, bsonFilter)
	}

	bsonFilter := bson.M{rel.TargetField: toBSONValue(val)}
	return queryDocs(ctx, db, rel.TargetCollection, bsonFilter)
}

func findRelatedReverse(ctx context.Context, db *mongo.Database, documentID string, rel domain.Relationship) ([]bson.M, error) {
	idVal := toBSONValue(documentID)
	bsonFilter := bson.M{rel.SourceField: idVal}
	return queryDocs(ctx, db, rel.SourceCollection, bsonFilter)
}

func queryDocs(ctx context.Context, db *mongo.Database, collection string, filter bson.M) ([]bson.M, error) {
	cursor, err := db.Collection(collection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func findDocumentAcrossCollections(ctx context.Context, db *mongo.Database, documentID string) (bson.M, string, error) {
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, "", fmt.Errorf("list collections: %w", err)
	}

	bsonID := toBSONValue(documentID)

	for _, coll := range collections {
		var result bson.M
		err := db.Collection(coll).FindOne(ctx, bson.M{"_id": bsonID}).Decode(&result)
		if err == nil {
			return result, coll, nil
		}
	}

	return nil, "", nil
}

func toBSONValue(v any) any {
	switch val := v.(type) {
	case string:
		if len(val) == 24 {
			if objID, err := primitive.ObjectIDFromHex(val); err == nil {
				return objID
			}
		}
		return val
	case map[string]any:
		if oid, ok := val["$oid"]; ok {
			hexStr := fmt.Sprintf("%v", oid)
			if objID, err := primitive.ObjectIDFromHex(hexStr); err == nil {
				return objID
			}
		}
		return val
	default:
		return val
	}
}

func nestedFieldValue(doc any, field string) any {
	parts := strings.Split(field, ".")
	cur := doc
	for _, part := range parts {
		switch m := cur.(type) {
		case bson.M:
			cur = m[part]
		case map[string]any:
			cur = m[part]
		default:
			return nil
		}
		if cur == nil {
			return nil
		}
	}
	return cur
}

func toBSONArray(arr primitive.A) []any {
	result := make([]any, 0, len(arr))
	for _, elem := range arr {
		result = append(result, toBSONValue(elem))
	}
	return result
}

func docID(doc any) string {
	if m, ok := doc.(bson.M); ok {
		return idToString(m["_id"])
	}
	if m, ok := doc.(map[string]any); ok {
		return idToString(m["_id"])
	}
	return ""
}

func idToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case primitive.ObjectID:
		return val.Hex()
	default:
		return fmt.Sprintf("%v", val)
	}
}

type visitKey struct {
	collection string
	id         string
}

func flattenTree(node *domain.InvestigationNode, flat *[]domain.FlatDocument) {
	*flat = append(*flat, domain.FlatDocument{
		Collection: node.Collection,
		ID:         node.ID,
		Document:   node.Document,
	})
	for _, child := range node.Children {
		flattenTree(child, flat)
	}
}

func (s *InvestigationService) BatchInvestigate(ctx context.Context, connectionID string, documentIDs []string) (*domain.BatchInvestigateResult, error) {
	results := make(map[string]domain.InvestigateResult)

	for _, docID := range documentIDs {
		docCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := s.Investigate(docCtx, connectionID, docID)
		cancel()
		if err != nil {
			log.Printf("warning: failed to investigate document %s: %v", docID, err)
			continue
		}
		results[docID] = *result
	}

	return &domain.BatchInvestigateResult{Results: results}, nil
}

func (s *InvestigationService) TraceRelationship(ctx context.Context, relationshipID string, limit int) (*domain.RelationshipTrace, error) {
	rel, err := s.relStore.Get(ctx, relationshipID)
	if err != nil {
		return nil, fmt.Errorf("get relationship: %w", err)
	}

	conn, err := s.connStore.Get(ctx, rel.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	db := mongoConn.Database(conn.Database)

	forwardDocs, err := s.traceForward(ctx, db, *rel, limit)
	if err != nil {
		log.Printf("warning: forward trace failed: %v", err)
	}

	reverseDocs, err := s.traceReverse(ctx, db, *rel, limit)
	if err != nil {
		log.Printf("warning: reverse trace failed: %v", err)
	}

	return &domain.RelationshipTrace{
		RelationshipID:   rel.ID,
		SourceCollection: rel.SourceCollection,
		SourceField:      rel.SourceField,
		TargetCollection: rel.TargetCollection,
		TargetField:      rel.TargetField,
		ForwardDocs:      forwardDocs,
		ReverseDocs:      reverseDocs,
	}, nil
}

func (s *InvestigationService) traceForward(ctx context.Context, db *mongo.Database, rel domain.Relationship, limit int) ([]domain.FlatDocument, error) {
	bsonFilter := bson.M{rel.SourceField: bson.M{"$ne": nil}}

	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := db.Collection(rel.SourceCollection).Find(ctx, bsonFilter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.FlatDocument
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		docID := docID(doc)
		if docID == "" {
			continue
		}

		results = append(results, domain.FlatDocument{
			Collection: rel.SourceCollection,
			ID:         docID,
			Document:   doc,
		})
	}
	return results, cursor.Err()
}

func (s *InvestigationService) traceReverse(ctx context.Context, db *mongo.Database, rel domain.Relationship, limit int) ([]domain.FlatDocument, error) {
	bsonFilter := bson.M{rel.TargetField: bson.M{"$ne": nil}}

	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := db.Collection(rel.TargetCollection).Find(ctx, bsonFilter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.FlatDocument
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		docID := docID(doc)
		if docID == "" {
			continue
		}

		results = append(results, domain.FlatDocument{
			Collection: rel.TargetCollection,
			ID:         docID,
			Document:   doc,
		})
	}
	return results, cursor.Err()
}

func (s *InvestigationService) GetSchemaMap(ctx context.Context, connectionID string) (*domain.SchemaMap, error) {
	rels, err := s.relStore.GetApproved(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get approved relationships: %w", err)
	}

	collectionSet := make(map[string]bool)
	for _, rel := range rels {
		collectionSet[rel.SourceCollection] = true
		collectionSet[rel.TargetCollection] = true
	}

	var nodes []domain.SchemaNode
	for coll := range collectionSet {
		nodes = append(nodes, domain.SchemaNode{
			Collection: coll,
			FieldCount: 0,
		})
	}

	var edges []domain.SchemaEdge
	for _, rel := range rels {
		edges = append(edges, domain.SchemaEdge{
			ID:               rel.ID,
			SourceCollection: rel.SourceCollection,
			SourceField:      rel.SourceField,
			TargetCollection: rel.TargetCollection,
			TargetField:      rel.TargetField,
			Confidence:       rel.Confidence,
		})
	}

	return &domain.SchemaMap{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
