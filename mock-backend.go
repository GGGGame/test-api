// mock-backend.go
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// API endpoints for testing different scenarios
	http.HandleFunc("/api/users", corsMiddleware(handleUsers))
	http.HandleFunc("/api/products", corsMiddleware(handleProducts))
	http.HandleFunc("/api/orders", corsMiddleware(handleOrders))
	http.HandleFunc("/api/health", corsMiddleware(handleHealth))
	http.HandleFunc("/api/slow", corsMiddleware(handleSlow))
	http.HandleFunc("/api/unreliable", corsMiddleware(handleUnreliable))
	http.HandleFunc("/api/graphql", corsMiddleware(handleGraphQL))
	http.HandleFunc("/api/xml", corsMiddleware(handleXML))
	http.HandleFunc("/api/legacy-soap", corsMiddleware(handleLegacySOAP))
	http.HandleFunc("/api/secure", corsMiddleware(handleSecure))
	http.HandleFunc("/api/echo", corsMiddleware(handleEcho))
	http.HandleFunc("/api/error", corsMiddleware(handleError))
	http.HandleFunc("/api/error/", corsMiddleware(handleError))
	http.HandleFunc("/", corsMiddleware(handleRoot))

	log.Println("🚀 Mock Backend started on :4000")
	log.Println("Available endpoints:")
	log.Println("  GET  /api/users        - User management (100+ items, supports auth headers)")
	log.Println("  GET  /api/products     - Product catalog (Large payload for compression testing)")
	log.Println("  GET  /api/orders       - Order management (Simulates DB latency)")
	log.Println("  GET  /api/health       - Health check")
	log.Println("  GET  /api/slow         - Simulates slow backend (2-5s latency)")
	log.Println("  GET  /api/unreliable   - Random failures (50% 503 Service Unavailable)")
	log.Println("  POST /api/graphql      - GraphQL endpoint (Simulates introspection & queries)")
	log.Println("  POST /api/xml          - Accepts JSON/XML, returns XML")
	log.Println("  GET  /api/legacy-soap  - Returns realistic SOAP XML")
	log.Println("  ALL  /api/secure       - Requires X-Nolxy-Secret header")
	log.Println("  ALL  /api/echo         - Echoes headers, method, and body (Pipeline testing)")
	log.Println("  GET  /api/error        - Simulates specific HTTP errors (?code=500)")

	server := &http.Server{
		Addr:         ":4000",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second, // Allow slow responses
	}
	log.Fatal(server.ListenAndServe())
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func sanitizeHeaders(h http.Header) map[string]string {
	safe := make(map[string]string)
	for k, v := range h {
		safe[k] = strings.Join(v, ", ")
	}
	return safe
}

// ─── CORS Middleware ─────────────────────────────────────────────────────────

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin (for testing purposes)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Nolxy-Api-Key, X-Nolxy-Secret, X-User-Id, X-User-Role")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the actual handler
		next(w, r)
	}
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "mock_backend_active",
		"method":    r.Method,
		"url":       r.URL.String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleEcho(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := io.ReadAll(r.Body)

	// Try to parse body as JSON for prettier output, otherwise string
	var jsonBody interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &jsonBody); err != nil {
			jsonBody = string(body)
		}
	}

	response := map[string]interface{}{
		"method":  r.Method,
		"url":     r.URL.String(),
		"headers": sanitizeHeaders(r.Header),
		"query":   r.URL.Query(),
		"body":    jsonBody,
	}
	json.NewEncoder(w).Encode(response)
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate backend secret if present
	secret := r.Header.Get("X-Nolxy-Secret")
	if secret != "" {
		w.Header().Set("X-Backend-Auth", "valid")
	}

	// Forwarded user info from JWT claims
	userID := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")

	// Generate a realistic list of users
	users := make([]map[string]interface{}, 0, 150)
	for i := 1; i <= 150; i++ {
		users = append(users, map[string]interface{}{
			"id":         i,
			"uuid":       fmt.Sprintf("usr_%d%d", time.Now().Unix(), i),
			"name":       fmt.Sprintf("User %d", i),
			"email":      fmt.Sprintf("user%d@enterprise.local", i),
			"role":       "standard_user",
			"status":     "active",
			"created_at": time.Now().AddDate(0, -randInt(12), -randInt(28)).Format(time.RFC3339),
			"preferences": map[string]interface{}{
				"theme":         "dark",
				"notifications": true,
				"language":      "en-US",
			},
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"metadata": map[string]interface{}{
			"total":         150,
			"page":          1,
			"caller_id":     userID,
			"caller_role":   role,
			"authenticated": secret != "",
		},
		"data": users,
	})
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Create a very large payload to effectively test compression (GZIP/Brotli)
	// ~1000 products with long descriptions
	products := make([]map[string]interface{}, 0, 1000)
	longDesc := strings.Repeat("This is a detailed product description designed to take up space and compress well. ", 10)

	for i := 1; i <= 1000; i++ {
		products = append(products, map[string]interface{}{
			"id":          fmt.Sprintf("PROD-%05d", i),
			"name":        fmt.Sprintf("Enterprise Server Grade Product %d", i),
			"description": longDesc,
			"price":       float64(randInt(100000)) / 100,
			"stock":       randInt(500),
			"category":    "Enterprise Hardware",
			"tags":        []string{"server", "rack", "enterprise", "compute"},
			"specs": map[string]string{
				"weight":     fmt.Sprintf("%d kg", randInt(20)+5),
				"dimensions": "19x40x2",
				"power":      fmt.Sprintf("%dW", randInt(500)+200),
			},
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": 1000,
		"items": products,
	})
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Simulate realistic DB latency (50ms - 200ms)
	time.Sleep(time.Duration(50+randInt(150)) * time.Millisecond)

	// Contextual orders based on requested user
	reqUser := r.Header.Get("X-User-Id")
	if reqUser == "" {
		reqUser = "anonymous"
	}

	orders := make([]map[string]interface{}, 0, 20)
	statuses := []string{"processing", "shipped", "delivered"}
	for i := 1; i <= 20; i++ {
		orders = append(orders, map[string]interface{}{
			"order_id":   fmt.Sprintf("ORD-%s-%d", reqUser, i),
			"user_id":    reqUser,
			"amount":     float64(randInt(50000)) / 100,
			"status":     statuses[randInt(3)],
			"created_at": time.Now().Add(-time.Duration(randInt(720)) * time.Hour).Format(time.RFC3339),
			"items": []map[string]interface{}{
				{"sku": "PROD-A", "qty": randInt(5) + 1},
				{"sku": "PROD-B", "qty": randInt(2) + 1},
			},
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":   reqUser,
		"orders": orders,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"uptime":  time.Since(time.Now().Add(-24 * time.Hour)).Seconds(),
	})
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Simulate slow processing (2.5s to 4s) to trigger Gateway timeouts
	delay := time.Duration(2500+randInt(1500)) * time.Millisecond
	time.Sleep(delay)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "This request took a long time to process",
		"delay":   delay.String(),
	})
}

func handleUnreliable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 50% chance of failure to test Circuit Breaker and Retries
	if randInt(100) < 50 {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Service Unavailable",
			"code":  503,
			"retry_after": 5,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Request succeeded this time",
		"status":  "success",
	})
}

func handleGraphQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := io.ReadAll(r.Body)
	query := string(body)

	// Simulate Introspection query response
	if strings.Contains(query, "__schema") {
		// Mock a basic GraphQL introspection response
		response := `{"data":{"__schema":{"queryType":{"name":"Query"},"mutationType":null,"subscriptionType":null,"types":[{"kind":"OBJECT","name":"Query","fields":[{"name":"users","type":{"kind":"LIST","ofType":{"name":"User"}}}]},{"kind":"OBJECT","name":"User","fields":[{"name":"id","type":{"name":"ID"}},{"name":"name","type":{"name":"String"}}]}]}}}`
		w.Write([]byte(response))
		return
	}

	// Normal query response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": "1", "name": "GraphQL User"},
			},
		},
	})
}

func handleLegacySOAP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	
	// Realistic SOAP XML Response (to test XML -> JSON pipelines on response)
	xml := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Body>
    <GetUserDetailsResponse xmlns="http://enterprise.local/api/v1">
      <GetUserDetailsResult>
        <ID>10001</ID>
        <FullName>Jane Enterprise</FullName>
        <Email>jane.enterprise@legacy.system.com</Email>
        <Department>Finance</Department>
        <SecretToken>XYZ-987654321-SUPER-SECRET</SecretToken>
        <Permissions>
          <Permission>READ_FINANCE</Permission>
          <Permission>WRITE_FINANCE</Permission>
        </Permissions>
      </GetUserDetailsResult>
    </GetUserDetailsResponse>
  </soap:Body>
</soap:Envelope>`

	w.Write([]byte(xml))
}

func handleXML(w http.ResponseWriter, r *http.Request) {
	// Reads JSON or XML and returns XML to test pipelines
	body, _ := io.ReadAll(r.Body)
	
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	
	// Echo back some of what was sent to prove it worked
	escapedBody := strings.ReplaceAll(string(body), "<", "&lt;")
	escapedBody = strings.ReplaceAll(escapedBody, ">", "&gt;")

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<response>
    <status>success</status>
    <message>Data processed</message>
    <received_payload>
        <length>%d</length>
        <content>%s</content>
    </received_payload>
    <sensitive_data>
        <password>my_super_secret_password_that_should_be_hidden</password>
        <token>jwt_token_123456</token>
    </sensitive_data>
</response>`, len(body), escapedBody)

	w.Write([]byte(xml))
}

func handleSecure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	secret := r.Header.Get("X-Nolxy-Secret")

	if secret == "my-secret-key" || secret == "auto" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Authenticated successfully",
			"secret":  secret,
			"data": map[string]interface{}{
				"api_keys": []string{"key1", "key2", "key3"},
				"config": map[string]interface{}{
					"rate_limit": 1000,
					"timeout":    30,
				},
			},
		})
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Invalid or missing secret",
			"detail": "Endpoint requires valid X-Nolxy-Secret header",
			"received_secret": secret,
		})
	}
}

func handleError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	codeStr := r.URL.Query().Get("code")
	if codeStr == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			codeStr = parts[len(parts)-1]
		}
	}
	code := http.StatusInternalServerError
	if c, err := strconv.Atoi(codeStr); err == nil && c >= 400 && c <= 599 {
		code = c
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":       http.StatusText(code),
		"status_code": code,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"path":        r.URL.Path,
		"suggestion":  "Verify your request parameters and headers",
	})
}