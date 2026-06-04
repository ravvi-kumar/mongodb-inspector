package service

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

const maxDepth = 5

type InvestigationService struct {
	connStore domain.ConnectionReader
	relStore  domain.RelationshipReaderWriter
}

func NewInvestigationService(connStore *pg.ConnectionStore, relStore *pg.RelationshipStore) *InvestigationService {
	return &InvestigationService{connStore: connStore, relStore: relStore}
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
	}

	traverse(ctx, db, tree, rootColl, documentID, rels, visited, 0)

	var flat []domain.FlatDocument
	flattenTree(tree, &flat)

	return &domain.InvestigateResult{
		Root:      *tree,
		Tree:      tree,
		Documents: flat,
	}, nil
}

func traverse(
	ctx context.Context,
	db *mongodriver.Database,
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
				}
				parent.Children = append(parent.Children, node)
				traverse(ctx, db, node, rel.SourceCollection, childID, rels, visited, depth+1)
			}
		}
	}
}

func findRelatedForward(ctx context.Context, db *mongodriver.Database, doc any, rel domain.Relationship) ([]bson.M, error) {
	val := fieldValue(doc, rel.SourceField)
	if val == nil {
		return nil, nil
	}

	bsonFilter := bson.M{rel.TargetField: toBSONValue(val)}
	return queryDocs(ctx, db, rel.TargetCollection, bsonFilter)
}

func findRelatedReverse(ctx context.Context, db *mongodriver.Database, documentID string, rel domain.Relationship) ([]bson.M, error) {
	idVal := toBSONValue(documentID)
	bsonFilter := bson.M{rel.SourceField: idVal}
	return queryDocs(ctx, db, rel.SourceCollection, bsonFilter)
}

func queryDocs(ctx context.Context, db *mongodriver.Database, collection string, filter bson.M) ([]bson.M, error) {
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

func findDocumentAcrossCollections(ctx context.Context, db *mongodriver.Database, documentID string) (bson.M, string, error) {
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

func fieldValue(doc any, field string) any {
	if m, ok := doc.(bson.M); ok {
		return m[field]
	}
	if m, ok := doc.(map[string]any); ok {
		return m[field]
	}
	return nil
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
