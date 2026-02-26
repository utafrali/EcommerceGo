// Package main implements a standalone seed script that populates the
// EcommerceGo platform with realistic test data. It uses direct SQL for
// entities without HTTP endpoints (categories, brands, images, variants)
// and HTTP calls to the running backend services for products, inventory,
// and campaigns.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// --------------------------------------------------------------------------
// Configuration helpers
// --------------------------------------------------------------------------

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --------------------------------------------------------------------------
// HTTP helpers
// --------------------------------------------------------------------------

func httpPost(url, token string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

func httpPut(url, token string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

// --------------------------------------------------------------------------
// Seed data definitions
// --------------------------------------------------------------------------

type categoryDef struct {
	name string
	slug string
	id   string // populated after insert
}

type brandDef struct {
	name string
	slug string
	id   string // populated after insert
}

type productDef struct {
	name         string
	description  string
	categorySlug string
	brandSlug    string
	price        int64 // cents
}

type variantDef struct {
	name       string
	sku        string
	price      *int64
	attributes map[string]string
}

// --------------------------------------------------------------------------
// main
// --------------------------------------------------------------------------

func main() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[seed] ")

	dbURL := getEnv("DATABASE_URL", "postgres://ecommerce:ecommerce_secret@localhost:5432/product_db?sslmode=disable")
	gatewayURL := getEnv("GATEWAY_URL", "http://localhost:8080")
	inventoryURL := getEnv("INVENTORY_URL", "http://localhost:8007")
	campaignURL := getEnv("CAMPAIGN_URL", "http://localhost:8008")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// ---------------------------------------------------------------
	// 1. Connect to product database
	// ---------------------------------------------------------------
	log.Println("Connecting to product database...")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Println("Connected to product database.")

	// ---------------------------------------------------------------
	// 2. Seed categories via direct SQL
	// ---------------------------------------------------------------
	categories := []categoryDef{
		{name: "Electronics", slug: "electronics"},
		{name: "Clothing", slug: "clothing"},
		{name: "Home & Kitchen", slug: "home-kitchen"},
		{name: "Sports & Outdoors", slug: "sports-outdoors"},
		{name: "Books", slug: "books"},
	}

	log.Println("Seeding categories...")
	for i := range categories {
		var id string
		err := pool.QueryRow(ctx,
			`INSERT INTO categories (name, slug, sort_order, is_active)
			 VALUES ($1, $2, $3, true)
			 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			 RETURNING id`,
			categories[i].name, categories[i].slug, i+1,
		).Scan(&id)
		if err != nil {
			log.Printf("  WARNING: category %q: %v", categories[i].name, err)
			// Try to fetch existing ID.
			_ = pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, categories[i].slug).Scan(&id)
		}
		categories[i].id = id
		log.Printf("  Category: %s (id=%s)", categories[i].name, id)
	}

	// Build slug-to-id map for categories.
	categoryMap := make(map[string]string)
	for _, c := range categories {
		categoryMap[c.slug] = c.id
	}

	// ---------------------------------------------------------------
	// 3. Seed brands via direct SQL
	// ---------------------------------------------------------------
	brands := []brandDef{
		{name: "TechBrand", slug: "techbrand"},
		{name: "StyleCo", slug: "styleco"},
		{name: "HomeEssentials", slug: "homeessentials"},
		{name: "SportPro", slug: "sportpro"},
		{name: "BookWorld", slug: "bookworld"},
	}

	log.Println("Seeding brands...")
	for i := range brands {
		var id string
		err := pool.QueryRow(ctx,
			`INSERT INTO brands (name, slug)
			 VALUES ($1, $2)
			 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			 RETURNING id`,
			brands[i].name, brands[i].slug,
		).Scan(&id)
		if err != nil {
			log.Printf("  WARNING: brand %q: %v", brands[i].name, err)
			_ = pool.QueryRow(ctx, `SELECT id FROM brands WHERE slug = $1`, brands[i].slug).Scan(&id)
		}
		brands[i].id = id
		log.Printf("  Brand: %s (id=%s)", brands[i].name, id)
	}

	// Build slug-to-id map for brands.
	brandMap := make(map[string]string)
	for _, b := range brands {
		brandMap[b.slug] = b.id
	}

	// ---------------------------------------------------------------
	// 4. Seed products via HTTP (gateway or direct product service)
	// ---------------------------------------------------------------
	products := []productDef{
		// Electronics
		{"Wireless Bluetooth Headphones", "Premium noise-cancelling over-ear headphones with 30-hour battery life and active noise cancellation technology.", "electronics", "techbrand", 7999},
		{"USB-C Hub Adapter", "7-in-1 USB-C hub with HDMI 4K output, 3x USB 3.0 ports, SD card reader, and 100W power delivery.", "electronics", "techbrand", 3499},
		{"Mechanical Keyboard", "RGB backlit mechanical keyboard with Cherry MX Blue switches, full-size layout, and detachable wrist rest.", "electronics", "techbrand", 8999},
		{"4K Webcam", "Ultra HD webcam for streaming and video conferencing with auto-focus, built-in ring light, and privacy shutter.", "electronics", "techbrand", 12999},
		{"Portable SSD 1TB", "Fast external solid state drive with USB 3.2 Gen 2 interface, up to 1050MB/s read speed, and shock-resistant design.", "electronics", "techbrand", 9999},
		{"Smart Watch Pro", "Advanced fitness tracker with heart rate monitoring, GPS, sleep tracking, and 7-day battery life.", "electronics", "techbrand", 19999},
		// Clothing
		{"Classic Cotton T-Shirt", "Comfortable everyday tee made from 100% organic cotton with a relaxed fit and pre-shrunk fabric.", "clothing", "styleco", 2499},
		{"Slim Fit Jeans", "Modern slim fit denim jeans with stretch technology for all-day comfort and classic 5-pocket styling.", "clothing", "styleco", 4999},
		{"Running Shoes", "Lightweight performance running shoes with responsive cushioning, breathable mesh upper, and rubber outsole.", "clothing", "styleco", 8999},
		{"Wool Sweater", "Warm merino wool pullover sweater with ribbed cuffs and hem, perfect for layering in cold weather.", "clothing", "styleco", 5999},
		{"Rain Jacket", "Waterproof breathable jacket with sealed seams, adjustable hood, and zippered pockets for outdoor adventures.", "clothing", "styleco", 7999},
		{"Casual Sneakers", "Everyday comfort sneakers with memory foam insole, canvas upper, and vulcanized rubber sole.", "clothing", "styleco", 6499},
		// Home & Kitchen
		{"Stainless Steel Cookware Set", "10-piece professional-grade stainless steel cookware set with tri-ply construction and tempered glass lids.", "home-kitchen", "homeessentials", 14999},
		{"Coffee Maker", "12-cup programmable drip coffee brewer with built-in grinder, thermal carafe, and auto-shutoff feature.", "home-kitchen", "homeessentials", 4999},
		{"Knife Set", "8-piece chef knife collection with forged high-carbon stainless steel blades and ergonomic handles.", "home-kitchen", "homeessentials", 7999},
		{"Blender Pro", "High-speed countertop blender with 1500W motor, 64oz BPA-free pitcher, and variable speed control.", "home-kitchen", "homeessentials", 6999},
		{"Ceramic Plate Set", "6-piece artisan dinner plate set with reactive glaze finish, microwave and dishwasher safe.", "home-kitchen", "homeessentials", 3999},
		{"Cast Iron Skillet", "Pre-seasoned 12-inch cast iron skillet with helper handle, oven safe to 500F, and lifetime warranty.", "home-kitchen", "homeessentials", 3499},
		// Sports & Outdoors
		{"Yoga Mat Premium", "Non-slip 6mm thick exercise mat with alignment markings, carry strap, and eco-friendly TPE material.", "sports-outdoors", "sportpro", 2999},
		{"Resistance Bands Set", "5-level progressive workout bands with door anchor, handles, and ankle straps for full-body training.", "sports-outdoors", "sportpro", 1999},
		{"Camping Tent 4-Person", "Waterproof family tent with instant setup, full-coverage rainfly, and mesh ventilation panels.", "sports-outdoors", "sportpro", 19999},
		{"Hiking Backpack 50L", "Durable adventure backpack with adjustable suspension, rain cover, and multiple access points.", "sports-outdoors", "sportpro", 8999},
		{"Water Bottle Insulated", "Double-wall vacuum insulated bottle that keeps drinks cold 24 hours or hot 12 hours, BPA-free.", "sports-outdoors", "sportpro", 2499},
		{"Dumbbell Set", "Adjustable 5-50lb dumbbell set with quick-change mechanism and compact storage tray.", "sports-outdoors", "sportpro", 29999},
		// Books
		{"The Go Programming Language", "Comprehensive guide to Go by Donovan and Kernighan covering the fundamentals and advanced topics.", "books", "bookworld", 3999},
		{"Clean Code", "A handbook of agile software craftsmanship by Robert C. Martin with practical coding principles.", "books", "bookworld", 3499},
		{"System Design Interview", "Step-by-step framework for tackling system design questions with real-world architecture examples.", "books", "bookworld", 2999},
		{"Designing Data-Intensive Apps", "Big ideas behind reliable, scalable, and maintainable data systems by Martin Kleppmann.", "books", "bookworld", 4499},
		{"The Pragmatic Programmer", "Journey to mastery covering topics from personal responsibility to architecture design.", "books", "bookworld", 3999},
		{"Domain-Driven Design", "Tackling complexity in the heart of software by Eric Evans with strategic and tactical patterns.", "books", "bookworld", 4999},
	}

	// Determine product creation URL.
	// Try the gateway first, fall back to direct product service.
	productURL := gatewayURL + "/api/v1/products"

	// Try to get a JWT token for authenticated endpoints.
	token := ""
	log.Println("Attempting to register/login test admin user...")
	regBody := map[string]any{
		"email":      "admin@ecommerce.test",
		"password":   "SecurePass123",
		"first_name": "Admin",
		"last_name":  "User",
	}
	_, regErr := httpPost(gatewayURL+"/api/v1/auth/register", "", regBody)
	if regErr != nil {
		log.Printf("  Register: %v (may already exist, continuing)", regErr)
	} else {
		log.Println("  Registered test admin user.")
	}

	loginBody := map[string]any{
		"email":    "admin@ecommerce.test",
		"password": "SecurePass123",
	}
	loginResp, loginErr := httpPost(gatewayURL+"/api/v1/auth/login", "", loginBody)
	if loginErr != nil {
		log.Printf("  Login failed: %v", loginErr)
		log.Println("  Falling back to direct product service (no auth).")
		productURL = getEnv("PRODUCT_URL", "http://localhost:8001") + "/api/v1/products"
	} else {
		if data, ok := loginResp["data"].(map[string]any); ok {
			if tokens, ok := data["tokens"].(map[string]any); ok {
				if t, ok := tokens["access_token"].(string); ok {
					token = t
					log.Println("  Logged in, got JWT token.")
				}
			}
		}
		if token == "" {
			log.Println("  WARNING: Login succeeded but could not extract token.")
			log.Println("  Falling back to direct product service (no auth).")
			productURL = getEnv("PRODUCT_URL", "http://localhost:8001") + "/api/v1/products"
		}
	}

	log.Printf("Seeding %d products via %s ...", len(products), productURL)

	type createdProduct struct {
		id   string
		slug string
		def  productDef
	}
	var createdProducts []createdProduct

	for _, p := range products {
		catID := categoryMap[p.categorySlug]
		brandID := brandMap[p.brandSlug]

		body := map[string]any{
			"name":        p.name,
			"description": p.description,
			"base_price":  p.price,
			"currency":    "USD",
		}
		if catID != "" {
			body["category_id"] = catID
		}
		if brandID != "" {
			body["brand_id"] = brandID
		}

		resp, err := httpPost(productURL, token, body)
		if err != nil {
			log.Printf("  WARNING: create product %q: %v", p.name, err)
			continue
		}

		// Extract product ID from response.
		var productID, productSlug string
		if data, ok := resp["data"].(map[string]any); ok {
			if id, ok := data["id"].(string); ok {
				productID = id
			}
			if slug, ok := data["slug"].(string); ok {
				productSlug = slug
			}
		}

		if productID == "" {
			log.Printf("  WARNING: no product ID in response for %q", p.name)
			continue
		}

		createdProducts = append(createdProducts, createdProduct{
			id:   productID,
			slug: productSlug,
			def:  p,
		})
		log.Printf("  Product: %s (id=%s)", p.name, productID)
	}

	// ---------------------------------------------------------------
	// 5. Seed product images via direct SQL
	// ---------------------------------------------------------------
	log.Println("Seeding product images...")
	for _, cp := range createdProducts {
		imgCount := 2 + rand.Intn(3) // 2-4 images per product
		for j := 0; j < imgCount; j++ {
			imgURL := fmt.Sprintf("https://picsum.photos/seed/%s-%d/800/800", cp.slug, j+1)
			altText := fmt.Sprintf("%s - Image %d", cp.def.name, j+1)
			isPrimary := j == 0

			_, err := pool.Exec(ctx,
				`INSERT INTO product_images (product_id, url, alt_text, sort_order, is_primary)
				 VALUES ($1, $2, $3, $4, $5)
				 ON CONFLICT DO NOTHING`,
				cp.id, imgURL, altText, j, isPrimary,
			)
			if err != nil {
				log.Printf("  WARNING: image for %q: %v", cp.def.name, err)
			}
		}
	}
	log.Println("  Product images seeded.")

	// ---------------------------------------------------------------
	// 6. Seed product variants via direct SQL
	// ---------------------------------------------------------------
	log.Println("Seeding product variants...")
	for _, cp := range createdProducts {
		variants := getVariantsForCategory(cp.slug, cp.def.categorySlug, cp.def.price)
		for _, v := range variants {
			attrsJSON, _ := json.Marshal(v.attributes)
			_, err := pool.Exec(ctx,
				`INSERT INTO product_variants (product_id, sku, name, price, attributes, is_active)
				 VALUES ($1, $2, $3, $4, $5, true)
				 ON CONFLICT (sku) DO NOTHING`,
				cp.id, v.sku, v.name, v.price, attrsJSON,
			)
			if err != nil {
				log.Printf("  WARNING: variant %q for %q: %v", v.name, cp.def.name, err)
			}
		}
	}
	log.Println("  Product variants seeded.")

	// ---------------------------------------------------------------
	// 7. Publish all products (set status to "published")
	// ---------------------------------------------------------------
	log.Println("Publishing all products...")
	for _, cp := range createdProducts {
		publishBody := map[string]any{
			"status": "published",
		}
		_, err := httpPut(productURL+"/"+cp.id, token, publishBody)
		if err != nil {
			log.Printf("  WARNING: publish %q: %v", cp.def.name, err)
		}
	}
	log.Println("  All products published.")

	// ---------------------------------------------------------------
	// 8. Seed inventory via HTTP
	// ---------------------------------------------------------------
	log.Println("Seeding inventory...")
	for _, cp := range createdProducts {
		variants := getVariantsForCategory(cp.slug, cp.def.categorySlug, cp.def.price)
		for _, v := range variants {
			// Look up the variant ID from the database.
			var variantID string
			err := pool.QueryRow(ctx,
				`SELECT id FROM product_variants WHERE sku = $1`, v.sku,
			).Scan(&variantID)
			if err != nil {
				log.Printf("  WARNING: lookup variant %q: %v", v.sku, err)
				continue
			}

			qty := 50 + rand.Intn(151) // 50-200
			invBody := map[string]any{
				"product_id": cp.id,
				"variant_id": variantID,
				"quantity":   qty,
			}

			_, err = httpPost(inventoryURL+"/api/v1/inventory", token, invBody)
			if err != nil {
				log.Printf("  WARNING: inventory for %q variant %q: %v", cp.def.name, v.name, err)
				continue
			}
			log.Printf("  Inventory: %s / %s = %d units", cp.def.name, v.name, qty)
		}
	}

	// ---------------------------------------------------------------
	// 9. Seed campaigns via HTTP
	// ---------------------------------------------------------------
	log.Println("Seeding campaigns...")
	now := time.Now().UTC()
	campaignDefs := []map[string]any{
		{
			"name":           "Welcome Discount",
			"description":    "10% off for new customers on their first order",
			"type":           "percentage",
			"discount_value": 10,
			"code":           "WELCOME10",
			"start_date":     now.Format(time.RFC3339),
			"end_date":       now.AddDate(1, 0, 0).Format(time.RFC3339),
			"max_usage_count": 1000,
		},
		{
			"name":             "Summer Sale",
			"description":      "20% off on orders over $50",
			"type":             "percentage",
			"discount_value":   20,
			"min_order_amount": 5000,
			"code":             "SUMMER20",
			"start_date":       now.Format(time.RFC3339),
			"end_date":         now.AddDate(0, 3, 0).Format(time.RFC3339),
			"max_usage_count":  500,
		},
		{
			"name":           "Free Shipping",
			"description":    "Flat $9.99 discount (covers standard shipping)",
			"type":           "fixed_amount",
			"discount_value": 999,
			"code":           "FREESHIP",
			"start_date":     now.Format(time.RFC3339),
			"end_date":       now.AddDate(0, 6, 0).Format(time.RFC3339),
			"max_usage_count": 2000,
		},
	}

	for _, c := range campaignDefs {
		_, err := httpPost(campaignURL+"/api/v1/campaigns", token, c)
		if err != nil {
			log.Printf("  WARNING: campaign %q: %v", c["name"], err)
			continue
		}
		log.Printf("  Campaign: %s (code=%s)", c["name"], c["code"])
	}

	// ---------------------------------------------------------------
	// Done
	// ---------------------------------------------------------------
	log.Printf("Seed complete! Created %d products with images, variants, and inventory.", len(createdProducts))
}

// getVariantsForCategory returns variant definitions appropriate for the product
// category. Clothing products get size variants; electronics get color variants;
// other categories get a default "Standard" variant.
func getVariantsForCategory(productSlug, categorySlug string, basePrice int64) []variantDef {
	switch categorySlug {
	case "clothing":
		sizes := []string{"S", "M", "L", "XL"}
		variants := make([]variantDef, len(sizes))
		for i, size := range sizes {
			variants[i] = variantDef{
				name:       size,
				sku:        fmt.Sprintf("%s-%s", productSlug, size),
				price:      nil, // same as base price
				attributes: map[string]string{"size": size},
			}
		}
		return variants
	case "electronics":
		colors := []struct {
			name string
			adj  int64
		}{
			{"Black", 0},
			{"Silver", 500},
			{"White", 0},
		}
		variants := make([]variantDef, len(colors))
		for i, c := range colors {
			var price *int64
			if c.adj != 0 {
				p := basePrice + c.adj
				price = &p
			}
			variants[i] = variantDef{
				name:       c.name,
				sku:        fmt.Sprintf("%s-%s", productSlug, c.name),
				price:      price,
				attributes: map[string]string{"color": c.name},
			}
		}
		return variants
	default:
		return []variantDef{
			{
				name:       "Standard",
				sku:        fmt.Sprintf("%s-standard", productSlug),
				price:      nil,
				attributes: map[string]string{"type": "standard"},
			},
		}
	}
}
