package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ExpectedRel struct {
	SourceCollection string
	SourceField      string
	TargetCollection string
	TargetField      string
}

type Dataset struct {
	Name     string
	DB       string
	Expected []ExpectedRel
}

var datasets = []Dataset{
	{
		Name: "ecommerce",
		DB:   "inspect_ecommerce",
		Expected: []ExpectedRel{
			{"orders", "userId", "users", "_id"},
			{"reviews", "userId", "users", "_id"},
			{"reviews", "productId", "products", "_id"},
			{"payments", "orderId", "orders", "_id"},
			{"shipments", "orderId", "orders", "_id"},
		},
	},
	{
		Name: "saas",
		DB:   "inspect_saas",
		Expected: []ExpectedRel{
			{"users", "organizationId", "organizations", "_id"},
			{"projects", "organizationId", "organizations", "_id"},
			{"invoices", "organizationId", "organizations", "_id"},
			{"webhooks", "organizationId", "organizations", "_id"},
			{"events", "userId", "users", "_id"},
			{"events", "projectId", "projects", "_id"},
		},
	},
	{
		Name: "blog",
		DB:   "inspect_blog",
		Expected: []ExpectedRel{
			{"posts", "authorId", "authors", "_id"},
			{"comments", "postId", "posts", "_id"},
			{"comments", "authorId", "authors", "_id"},
			{"post_tags", "postId", "posts", "_id"},
			{"post_tags", "tagId", "tags", "_id"},
		},
	},
	{
		Name: "analytics",
		DB:   "inspect_analytics",
		Expected: []ExpectedRel{
			{"sessions", "userId", "users", "_id"},
			{"events", "sessionId", "sessions", "_id"},
			{"campaigns", "userId", "users", "_id"},
		},
	},
	{
		Name: "crm",
		DB:   "inspect_crm",
		Expected: []ExpectedRel{
			{"contacts", "companyId", "companies", "_id"},
			{"deals", "companyId", "companies", "_id"},
			{"deals", "contactId", "contacts", "_id"},
			{"deals", "ownerId", "users", "_id"},
			{"activities", "contactId", "contacts", "_id"},
			{"activities", "dealId", "deals", "_id"},
			{"activities", "userId", "users", "_id"},
			{"notes", "contactId", "contacts", "_id"},
			{"notes", "userId", "users", "_id"},
		},
	},
}

func main() {
	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	fmt.Println("=== MongoDB Inspector Validation ===")
	fmt.Printf("API: %s\n", baseURL)
	fmt.Printf("MongoDB: %s\n\n", mongoURI)

	healthResp, err := http.Get(baseURL + "/health")
	if err != nil {
		log.Fatalf("API not reachable: %v", err)
	}
	healthResp.Body.Close()
	fmt.Println("API is healthy")

	totalTP, totalFP, totalFN := 0, 0, 0

	for _, ds := range datasets {
		fmt.Printf("\n--- %s ---\n", ds.Name)
		tp, fp, fn := validateDataset(baseURL, mongoURI, ds)
		totalTP += tp
		totalFP += fp
		totalFN += fn
	}

	fmt.Println("\n=== TOTALS ===")
	fmt.Printf("True Positives:  %d\n", totalTP)
	fmt.Printf("False Positives: %d\n", totalFP)
	fmt.Printf("False Negatives: %d\n", totalFN)
	if totalTP+totalFP > 0 {
		precision := float64(totalTP) / float64(totalTP+totalFP) * 100
		fmt.Printf("Precision: %.1f%%\n", precision)
	}
	if totalTP+totalFN > 0 {
		recall := float64(totalTP) / float64(totalTP+totalFN) * 100
		fmt.Printf("Recall: %.1f%%\n", recall)
	}
}

func validateDataset(baseURL, mongoURI string, ds Dataset) (tp, fp, fn int) {
	connID := createConnection(baseURL, ds.Name, mongoURI, ds.DB)
	if connID == "" {
		fmt.Printf("  FAILED: could not create connection\n")
		return 0, 0, len(ds.Expected)
	}

	selectDB(baseURL, connID, ds.DB)

	scanID := startScan(baseURL, connID)
	if scanID == "" {
		fmt.Printf("  FAILED: could not start scan\n")
		return 0, 0, len(ds.Expected)
	}

	waitForScan(baseURL, scanID)

	discoverRelationships(baseURL, scanID)

	rels := listRelationships(baseURL, connID)

	fmt.Printf("  Expected: %d relationships\n", len(ds.Expected))
	fmt.Printf("  Discovered: %d relationships\n", len(rels))

	discovered := make(map[string]DiscoveredRel)
	for _, r := range rels {
		key := fmt.Sprintf("%s.%s→%s.%s", r.SourceCollection, r.SourceField, r.TargetCollection, r.TargetField)
		discovered[key] = r
		fmt.Printf("    %s → %s.%s (confidence: %.1f%%, status: %s)\n",
			r.SourceField, r.TargetCollection, r.TargetField, r.Confidence*100, r.Status)
	}

	for _, exp := range ds.Expected {
		key := fmt.Sprintf("%s.%s→%s.%s", exp.SourceCollection, exp.SourceField, exp.TargetCollection, exp.TargetField)
		if _, ok := discovered[key]; ok {
			fmt.Printf("  [TP] %s.%s → %s.%s ✓\n", exp.SourceCollection, exp.SourceField, exp.TargetCollection, exp.TargetField)
			tp++
		} else {
			fmt.Printf("  [FN] %s.%s → %s.%s MISSED\n", exp.SourceCollection, exp.SourceField, exp.TargetCollection, exp.TargetField)
			fn++
		}
	}

	for key, r := range discovered {
		found := false
		for _, exp := range ds.Expected {
			expKey := fmt.Sprintf("%s.%s→%s.%s", exp.SourceCollection, exp.SourceField, exp.TargetCollection, exp.TargetField)
			if key == expKey {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("  [FP] %s.%s → %s.%s UNEXPECTED\n", r.SourceCollection, r.SourceField, r.TargetCollection, r.TargetField)
			fp++
		}
	}

	return tp, fp, fn
}

type apiResponse struct {
	ID string `json:"id"`
}

type DiscoveredRel struct {
	ID               string  `json:"id"`
	SourceCollection string  `json:"source_collection"`
	SourceField      string  `json:"source_field"`
	TargetCollection string  `json:"target_collection"`
	TargetField      string  `json:"target_field"`
	Confidence       float64 `json:"confidence"`
	Status           string  `json:"status"`
}

func createConnection(baseURL, name, mongoURI, db string) string {
	body := fmt.Sprintf(`{"name":"%s","connection_string":"%s"}`, name, mongoURI)
	resp, err := http.Post(baseURL+"/api/connections", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("  create connection error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	id, ok := result["id"].(string)
	if !ok {
		fmt.Printf("  create connection failed: %v\n", result)
		return ""
	}
	return id
}

func selectDB(baseURL, connID, db string) {
	body := fmt.Sprintf(`{"database":"%s"}`, db)
	url := fmt.Sprintf("%s/api/connections/%s/select-db", baseURL, connID)
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("  select db error: %v\n", err)
		return
	}
	resp.Body.Close()
}

func startScan(baseURL, connID string) string {
	body := fmt.Sprintf(`{"connection_id":"%s","sample_size":1000}`, connID)
	resp, err := http.Post(baseURL+"/api/scans", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("  start scan error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	id, ok := result["id"].(string)
	if !ok {
		fmt.Printf("  start scan failed: %v\n", result)
		return ""
	}
	return id
}

func waitForScan(baseURL, scanID string) {
	for i := 0; i < 60; i++ {
		resp, err := http.Get(fmt.Sprintf("%s/api/scans/%s", baseURL, scanID))
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		status, _ := result["status"].(string)
		if status == "completed" {
			return
		}
		if status == "failed" {
			fmt.Printf("  scan failed: %v\n", result["error"])
			return
		}
		time.Sleep(time.Second)
	}
	fmt.Println("  scan timed out")
}

func discoverRelationships(baseURL, scanID string) {
	body := fmt.Sprintf(`{"scan_id":"%s"}`, scanID)
	resp, err := http.Post(baseURL+"/api/relationships/discover", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("  discover error: %v\n", err)
		return
	}
	resp.Body.Close()
}

func listRelationships(baseURL, connID string) []DiscoveredRel {
	url := fmt.Sprintf("%s/api/relationships?connection_id=%s", baseURL, connID)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("  list relationships error: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var rels []DiscoveredRel
	json.NewDecoder(bytes.NewReader(data)).Decode(&rels)
	return rels
}
