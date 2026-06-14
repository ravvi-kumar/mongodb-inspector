package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("ping: %v", err)
	}

	rng := rand.New(rand.NewSource(42))

	datasets := []struct {
		name string
		seed func(ctx context.Context, db *mongo.Database, rng *rand.Rand)
	}{
		{"ecommerce", seedEcommerce},
		{"saas", seedSaaS},
		{"blog", seedBlog},
		{"analytics", seedAnalytics},
		{"crm", seedCRM},
	}

	for _, ds := range datasets {
		fmt.Printf("seeding %s...\n", ds.name)
		db := client.Database("inspect_" + ds.name)
		db.Drop(ctx)
		ds.seed(ctx, db, rng)
		fmt.Printf("  done: %s\n", ds.name)
	}

	fmt.Println("all datasets seeded")
}

func newOID() primitive.ObjectID {
	return primitive.NewObjectID()
}

func oids(n int) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, n)
	for i := range ids {
		ids[i] = newOID()
	}
	return ids
}

func pick(rng *rand.Rand, slice []primitive.ObjectID) primitive.ObjectID {
	return slice[rng.Intn(len(slice))]
}

func bulkInsert(ctx context.Context, coll *mongo.Collection, docs []any) {
	for i := 0; i < len(docs); i += 500 {
		end := i + 500
		if end > len(docs) {
			end = len(docs)
		}
		if _, err := coll.InsertMany(ctx, docs[i:end]); err != nil {
			log.Fatalf("insert into %s: %v", coll.Name(), err)
		}
	}
}

// ==================== ECOMMERCE ====================
// Expected relationships:
//
//	orders.userId → users._id
//	orders.items.productId → products._id
//	reviews.userId → users._id
//	reviews.productId → products._id
//	payments.orderId → orders._id
//	shipments.orderId → orders._id
func seedEcommerce(ctx context.Context, db *mongo.Database, rng *rand.Rand) {
	userIDs := oids(200)
	productIDs := oids(100)

	users := make([]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = bsonM{
			"_id":   id,
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
	}
	bulkInsert(ctx, db.Collection("users"), users)

	products := make([]any, len(productIDs))
	for i, id := range productIDs {
		products[i] = bsonM{
			"_id":      id,
			"name":     fmt.Sprintf("Product %d", i),
			"price":    float64(rng.Intn(10000)) / 100,
			"category": []string{"electronics", "clothing", "food", "books", "toys"}[rng.Intn(5)],
		}
	}
	bulkInsert(ctx, db.Collection("products"), products)

	type orderItem struct {
		ProductID primitive.ObjectID `bson:"productId"`
		Qty       int                `bson:"qty"`
	}

	orderIDs := oids(500)
	orders := make([]any, len(orderIDs))
	for i, id := range orderIDs {
		itemCount := rng.Intn(4) + 1
		items := make([]orderItem, itemCount)
		for j := range items {
			items[j] = orderItem{ProductID: pick(rng, productIDs), Qty: rng.Intn(5) + 1}
		}
		orders[i] = bsonM{
			"_id":    id,
			"userId": pick(rng, userIDs),
			"items":  items,
			"total":  float64(rng.Intn(50000)) / 100,
			"status": []string{"pending", "shipped", "delivered", "cancelled"}[rng.Intn(4)],
		}
	}
	bulkInsert(ctx, db.Collection("orders"), orders)

	reviews := make([]any, 300)
	for i := range reviews {
		reviews[i] = bsonM{
			"_id":       newOID(),
			"userId":    pick(rng, userIDs),
			"productId": pick(rng, productIDs),
			"rating":    rng.Intn(5) + 1,
			"comment":   fmt.Sprintf("Review %d", i),
		}
	}
	bulkInsert(ctx, db.Collection("reviews"), reviews)

	payments := make([]any, len(orderIDs))
	for i, orderID := range orderIDs {
		payments[i] = bsonM{
			"_id":     newOID(),
			"orderId": orderID,
			"amount":  float64(rng.Intn(50000)) / 100,
			"method":  []string{"credit_card", "paypal", "bank_transfer"}[rng.Intn(3)],
			"status":  []string{"completed", "failed", "refunded"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("payments"), payments)

	shipments := make([]any, 300)
	for i := range shipments {
		shipments[i] = bsonM{
			"_id":     newOID(),
			"orderId": pick(rng, orderIDs),
			"carrier": []string{"fedex", "ups", "dhl", "usps"}[rng.Intn(4)],
			"status":  []string{"in_transit", "delivered", "returned"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("shipments"), shipments)

	orphanUsers := make([]any, 10)
	for i := range orphanUsers {
		orphanUsers[i] = bsonM{
			"_id":   newOID(),
			"name":  fmt.Sprintf("Orphan User %d", i),
			"email": fmt.Sprintf("orphan%d@example.com", i),
		}
	}
	bulkInsert(ctx, db.Collection("orphan_users"), orphanUsers)

	orphanOrders := make([]any, 5)
	for i := range orphanOrders {
		orphanOrders[i] = bsonM{
			"_id":    newOID(),
			"userId": newOID(),
			"total":  99.99,
			"status": "pending",
		}
	}
	bulkInsert(ctx, db.Collection("orphan_orders"), orphanOrders)
}

// ==================== SAAS ====================
// Expected relationships:
//
//	projects.organizationId → organizations._id
//	users.organizationId → organizations._id
//	invoices.organizationId → organizations._id
//	webhooks.organizationId → organizations._id
//	events.userId → users._id
//	events.projectId → projects._id
func seedSaaS(ctx context.Context, db *mongo.Database, rng *rand.Rand) {
	orgIDs := oids(20)
	userIDs := oids(100)
	projectIDs := oids(80)

	orgs := make([]any, len(orgIDs))
	for i, id := range orgIDs {
		orgs[i] = bsonM{
			"_id":    id,
			"name":   fmt.Sprintf("Org %d", i),
			"plan":   []string{"free", "starter", "pro", "enterprise"}[rng.Intn(4)],
			"active": rng.Intn(10) > 1,
		}
	}
	bulkInsert(ctx, db.Collection("organizations"), orgs)

	users := make([]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = bsonM{
			"_id":            id,
			"organizationId": pick(rng, orgIDs),
			"name":           fmt.Sprintf("Member %d", i),
			"email":          fmt.Sprintf("member%d@org.com", i),
			"role":           []string{"admin", "member", "viewer"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("users"), users)

	projects := make([]any, len(projectIDs))
	for i, id := range projectIDs {
		projects[i] = bsonM{
			"_id":            id,
			"organizationId": pick(rng, orgIDs),
			"name":           fmt.Sprintf("Project %d", i),
			"status":         []string{"active", "archived", "deleted"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("projects"), projects)

	invoices := make([]any, 200)
	for i := range invoices {
		invoices[i] = bsonM{
			"_id":            newOID(),
			"organizationId": pick(rng, orgIDs),
			"amount":         float64(rng.Intn(500000)) / 100,
			"status":         []string{"paid", "pending", "overdue"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("invoices"), invoices)

	webhooks := make([]any, 50)
	for i := range webhooks {
		webhooks[i] = bsonM{
			"_id":            newOID(),
			"organizationId": pick(rng, orgIDs),
			"url":            fmt.Sprintf("https://example%d.com/webhook", i),
			"events":         []string{"create", "update", "delete"},
		}
	}
	bulkInsert(ctx, db.Collection("webhooks"), webhooks)

	events := make([]any, 1000)
	for i := range events {
		events[i] = bsonM{
			"_id":       newOID(),
			"userId":    pick(rng, userIDs),
			"projectId": pick(rng, projectIDs),
			"action":    []string{"create", "read", "update", "delete"}[rng.Intn(4)],
			"timestamp": time.Now().Add(-time.Duration(rng.Intn(86400)) * time.Second),
		}
	}
	bulkInsert(ctx, db.Collection("events"), events)
}

// ==================== BLOG ====================
// Expected relationships:
//
//	posts.authorId → authors._id
//	comments.postId → posts._id
//	comments.authorId → authors._id
//	post_tags.tagId → tags._id
//	post_tags.postId → posts._id
func seedBlog(ctx context.Context, db *mongo.Database, rng *rand.Rand) {
	authorIDs := oids(30)
	postIDs := oids(150)
	tagIDs := oids(20)

	authors := make([]any, len(authorIDs))
	for i, id := range authorIDs {
		authors[i] = bsonM{
			"_id":   id,
			"name":  fmt.Sprintf("Author %d", i),
			"email": fmt.Sprintf("author%d@blog.com", i),
			"bio":   fmt.Sprintf("Bio of author %d", i),
		}
	}
	bulkInsert(ctx, db.Collection("authors"), authors)

	posts := make([]any, len(postIDs))
	for i, id := range postIDs {
		posts[i] = bsonM{
			"_id":       id,
			"title":     fmt.Sprintf("Post %d", i),
			"authorId":  pick(rng, authorIDs),
			"body":      fmt.Sprintf("Content of post %d...", i),
			"published": rng.Intn(10) > 2,
		}
	}
	bulkInsert(ctx, db.Collection("posts"), posts)

	comments := make([]any, 500)
	for i := range comments {
		comments[i] = bsonM{
			"_id":      newOID(),
			"postId":   pick(rng, postIDs),
			"authorId": pick(rng, authorIDs),
			"body":     fmt.Sprintf("Comment %d", i),
			"approved": rng.Intn(10) > 2,
		}
	}
	bulkInsert(ctx, db.Collection("comments"), comments)

	tags := make([]any, len(tagIDs))
	for i, id := range tagIDs {
		tags[i] = bsonM{
			"_id":  id,
			"name": fmt.Sprintf("tag-%d", i),
			"slug": fmt.Sprintf("tag-%d", i),
		}
	}
	bulkInsert(ctx, db.Collection("tags"), tags)

	postTags := make([]any, 400)
	for i := range postTags {
		postTags[i] = bsonM{
			"_id":    newOID(),
			"postId": pick(rng, postIDs),
			"tagId":  pick(rng, tagIDs),
		}
	}
	bulkInsert(ctx, db.Collection("post_tags"), postTags)

	categories := make([]any, 10)
	for i := range categories {
		categories[i] = bsonM{
			"_id":  newOID(),
			"name": fmt.Sprintf("Category %d", i),
		}
	}
	bulkInsert(ctx, db.Collection("categories"), categories)
}

// ==================== ANALYTICS ====================
// Expected relationships:
//
//	events.sessionId → sessions._id
//	sessions.userId → users._id
//	campaigns.userId → users._id
func seedAnalytics(ctx context.Context, db *mongo.Database, rng *rand.Rand) {
	userIDs := oids(500)
	sessionIDs := oids(1000)

	users := make([]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = bsonM{
			"_id":      id,
			"name":     fmt.Sprintf("Visitor %d", i),
			"email":    fmt.Sprintf("visitor%d@analytics.com", i),
			"country":  []string{"US", "UK", "DE", "FR", "JP", "IN"}[rng.Intn(6)],
			"signedUp": rng.Intn(10) > 3,
		}
	}
	bulkInsert(ctx, db.Collection("users"), users)

	sessions := make([]any, len(sessionIDs))
	for i, id := range sessionIDs {
		sessions[i] = bsonM{
			"_id":      id,
			"userId":   pick(rng, userIDs),
			"device":   []string{"mobile", "desktop", "tablet"}[rng.Intn(3)],
			"browser":  []string{"chrome", "firefox", "safari", "edge"}[rng.Intn(4)],
			"duration": rng.Intn(3600),
		}
	}
	bulkInsert(ctx, db.Collection("sessions"), sessions)

	events := make([]any, 5000)
	for i := range events {
		events[i] = bsonM{
			"_id":       newOID(),
			"sessionId": pick(rng, sessionIDs),
			"type":      []string{"pageview", "click", "scroll", "purchase", "signup"}[rng.Intn(5)],
			"path":      fmt.Sprintf("/page/%d", rng.Intn(50)),
			"timestamp": time.Now().Add(-time.Duration(rng.Intn(86400*7)) * time.Second),
		}
	}
	bulkInsert(ctx, db.Collection("events"), events)

	campaigns := make([]any, 200)
	for i := range campaigns {
		campaigns[i] = bsonM{
			"_id":       newOID(),
			"userId":    pick(rng, userIDs),
			"source":    []string{"google", "facebook", "twitter", "email", "direct"}[rng.Intn(5)],
			"medium":    []string{"cpc", "organic", "referral", "email"}[rng.Intn(4)],
			"converted": rng.Intn(10) > 6,
		}
	}
	bulkInsert(ctx, db.Collection("campaigns"), campaigns)
}

// ==================== CRM ====================
// Expected relationships:
//
//	contacts.companyId → companies._id
//	deals.companyId → companies._id
//	deals.contactId → contacts._id
//	deals.ownerId → users._id
//	activities.contactId → contacts._id
//	activities.dealId → deals._id
//	activities.userId → users._id
//	notes.contactId → contacts._id
//	notes.userId → users._id
func seedCRM(ctx context.Context, db *mongo.Database, rng *rand.Rand) {
	companyIDs := oids(50)
	userIDs := oids(20)
	contactIDs := oids(200)
	dealIDs := oids(100)

	companies := make([]any, len(companyIDs))
	for i, id := range companyIDs {
		companies[i] = bsonM{
			"_id":      id,
			"name":     fmt.Sprintf("Company %d Inc.", i),
			"industry": []string{"tech", "finance", "healthcare", "retail", "education"}[rng.Intn(5)],
			"size":     []string{"small", "medium", "large", "enterprise"}[rng.Intn(4)],
		}
	}
	bulkInsert(ctx, db.Collection("companies"), companies)

	users := make([]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = bsonM{
			"_id":   id,
			"name":  fmt.Sprintf("Rep %d", i),
			"email": fmt.Sprintf("rep%d@crm.com", i),
			"role":  []string{"sales", "manager", "admin"}[rng.Intn(3)],
		}
	}
	bulkInsert(ctx, db.Collection("users"), users)

	contacts := make([]any, len(contactIDs))
	for i, id := range contactIDs {
		contacts[i] = bsonM{
			"_id":       id,
			"companyId": pick(rng, companyIDs),
			"name":      fmt.Sprintf("Contact %d", i),
			"email":     fmt.Sprintf("contact%d@company.com", i),
			"phone":     fmt.Sprintf("+1-555-%04d", rng.Intn(10000)),
		}
	}
	bulkInsert(ctx, db.Collection("contacts"), contacts)

	deals := make([]any, len(dealIDs))
	for i, id := range dealIDs {
		deals[i] = bsonM{
			"_id":       id,
			"companyId": pick(rng, companyIDs),
			"contactId": pick(rng, contactIDs),
			"ownerId":   pick(rng, userIDs),
			"title":     fmt.Sprintf("Deal %d", i),
			"value":     float64(rng.Intn(500000)) / 100,
			"stage":     []string{"lead", "qualified", "proposal", "negotiation", "closed_won", "closed_lost"}[rng.Intn(6)],
		}
	}
	bulkInsert(ctx, db.Collection("deals"), deals)

	activities := make([]any, 1000)
	for i := range activities {
		activities[i] = bsonM{
			"_id":       newOID(),
			"contactId": pick(rng, contactIDs),
			"dealId":    pick(rng, dealIDs),
			"userId":    pick(rng, userIDs),
			"type":      []string{"call", "email", "meeting", "note"}[rng.Intn(4)],
			"notes":     fmt.Sprintf("Activity %d notes", i),
			"timestamp": time.Now().Add(-time.Duration(rng.Intn(86400*30)) * time.Second),
		}
	}
	bulkInsert(ctx, db.Collection("activities"), activities)

	notes := make([]any, 300)
	for i := range notes {
		notes[i] = bsonM{
			"_id":       newOID(),
			"contactId": pick(rng, contactIDs),
			"userId":    pick(rng, userIDs),
			"body":      fmt.Sprintf("Note %d about this contact", i),
			"created":   time.Now().Add(-time.Duration(rng.Intn(86400*30)) * time.Second),
		}
	}
	bulkInsert(ctx, db.Collection("notes"), notes)
}

type bsonM = map[string]any
