// Package main implements a standalone seed script that populates the
// EcommerceGo product service database with 10,000 realistic Modanisa-style
// fashion products, complete with brands, images, and variants.
//
// Run: go run scripts/seed_10k_products.go
//   (from the repo root, or: cd scripts && go run seed_10k_products.go)
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	totalProducts = 10000
	batchSize     = 500
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Deterministic UUID generation from an index
// ---------------------------------------------------------------------------

// deterministicUUID produces a stable UUID v5-like string from a namespace and
// an integer index so that re-runs always produce the same product IDs.
func deterministicUUID(namespace string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", namespace, index)))
	// Format as UUID v4 layout (xxxxxxxx-xxxx-4xxx-Nxxx-xxxxxxxxxxxx).
	// Use explicit hex encoding to guarantee 8-4-4-4-12 character layout.
	hex := fmt.Sprintf("%x", h[:16]) // 32 hex chars from first 16 bytes
	// Inject version nibble (4) and variant bits (10xx).
	return fmt.Sprintf("%s-%s-4%s-%x%s-%s",
		hex[0:8],
		hex[8:12],
		hex[13:16],                           // 3 nibbles after version
		0x8|(h[8]&0x3),                       // variant: 10xx
		hex[17:20],                           // 3 nibbles
		hex[20:32],                           // 12 nibbles
	)
}

// ---------------------------------------------------------------------------
// Brand definitions
// ---------------------------------------------------------------------------

type brandDef struct {
	Name    string
	Slug    string
	LogoURL string
}

var brands = []brandDef{
	{"Refka", "refka", "https://cdn.modanisa.com/brands/refka.png"},
	{"Benin", "benin", "https://cdn.modanisa.com/brands/benin.png"},
	{"Alia", "alia", "https://cdn.modanisa.com/brands/alia.png"},
	{"Tuva", "tuva", "https://cdn.modanisa.com/brands/tuva.png"},
	{"Nihan", "nihan", "https://cdn.modanisa.com/brands/nihan.png"},
	{"Armine", "armine", "https://cdn.modanisa.com/brands/armine.png"},
	{"Kayra", "kayra", "https://cdn.modanisa.com/brands/kayra.png"},
	{"Setrms", "setrms", "https://cdn.modanisa.com/brands/setrms.png"},
	{"Alvina", "alvina", "https://cdn.modanisa.com/brands/alvina.png"},
	{"Sefamerve", "sefamerve", "https://cdn.modanisa.com/brands/sefamerve.png"},
}

// ---------------------------------------------------------------------------
// Category mapping (UUIDs from seed_modanisa_categories.sql)
// ---------------------------------------------------------------------------

// categoryEntry holds a leaf-level (or level-1) category to assign products to.
type categoryEntry struct {
	ID   string
	Name string
	Slug string
}

// topCategory groups leaf categories under a top-level distribution bucket.
type topCategory struct {
	Name       string
	Weight     float64 // share of total products (sums to 1.0)
	Categories []categoryEntry
}

var topCategories = []topCategory{
	{
		Name:   "Elbise",
		Weight: 0.20,
		Categories: []categoryEntry{
			{"b1000000-0000-0000-0000-000000000001", "Tesettur Elbise", "tesettur-elbise"},
			{"b1000000-0000-0000-0000-000000000002", "Triko Elbise", "triko-elbise"},
			{"b1000000-0000-0000-0000-000000000003", "Buyuk Beden Elbise", "buyuk-beden-elbise"},
			{"b1000000-0000-0000-0000-000000000004", "Namaz Elbisesi", "namaz-elbisesi"},
		},
	},
	{
		Name:   "Giyim",
		Weight: 0.30,
		Categories: []categoryEntry{
			// Dis Giyim level-2
			{"c2100000-0000-0000-0000-000000000001", "Mont", "mont"},
			{"c2100000-0000-0000-0000-000000000002", "Kaban/Manto", "kaban-manto"},
			{"c2100000-0000-0000-0000-000000000003", "Trenckot", "trenckot"},
			{"c2100000-0000-0000-0000-000000000004", "Ceket", "ceket"},
			{"c2100000-0000-0000-0000-000000000005", "Yelek", "yelek"},
			// Takim level-1
			{"b2000000-0000-0000-0000-000000000002", "Takim", "takim"},
			// Ust Giyim level-2
			{"c2300000-0000-0000-0000-000000000001", "Tunik", "tunik"},
			{"c2300000-0000-0000-0000-000000000002", "Bluz/Gomlek", "bluz-gomlek"},
			{"c2300000-0000-0000-0000-000000000003", "Sweatshirt", "sweatshirt"},
			{"c2300000-0000-0000-0000-000000000004", "Tisort", "tisort"},
			// Alt Giyim level-2
			{"c2400000-0000-0000-0000-000000000001", "Pantolon", "pantolon"},
			{"c2400000-0000-0000-0000-000000000002", "Etek", "etek"},
			{"c2400000-0000-0000-0000-000000000003", "Kot Pantolon", "kot-pantolon"},
			// Ev & Ic Giyim level-1
			{"b2000000-0000-0000-0000-000000000005", "Ev & Ic Giyim", "ev-ic-giyim"},
		},
	},
	{
		Name:   "Abiye",
		Weight: 0.10,
		Categories: []categoryEntry{
			{"b3000000-0000-0000-0000-000000000001", "Abiye Elbise", "abiye-elbise"},
			{"b3000000-0000-0000-0000-000000000002", "Abiye Takim", "abiye-takim"},
			{"b3000000-0000-0000-0000-000000000003", "Buyuk Beden Abiye", "buyuk-beden-abiye"},
		},
	},
	{
		Name:   "Basortusu",
		Weight: 0.10,
		Categories: []categoryEntry{
			{"b4000000-0000-0000-0000-000000000001", "Sal", "sal"},
			{"b4000000-0000-0000-0000-000000000002", "Esarp", "esarp"},
			{"b4000000-0000-0000-0000-000000000003", "Hazir Turban", "hazir-turban"},
			{"b4000000-0000-0000-0000-000000000004", "Bone", "bone"},
		},
	},
	{
		Name:   "Ayakkabi & Canta",
		Weight: 0.10,
		Categories: []categoryEntry{
			{"b7000000-0000-0000-0000-000000000001", "Bot/Cizme", "bot-cizme"},
			{"b7000000-0000-0000-0000-000000000002", "Spor Ayakkabi", "spor-ayakkabi"},
			{"b7000000-0000-0000-0000-000000000003", "Topuklu Ayakkabi", "topuklu-ayakkabi"},
			{"b7000000-0000-0000-0000-000000000004", "Omuz Cantasi", "omuz-cantasi"},
			{"b7000000-0000-0000-0000-000000000005", "Sirt Cantasi", "sirt-cantasi"},
		},
	},
	{
		Name:   "Aksesuar",
		Weight: 0.10,
		Categories: []categoryEntry{
			{"c6100000-0000-0000-0000-000000000001", "Kolye", "kolye"},
			{"c6100000-0000-0000-0000-000000000002", "Kupe", "kupe"},
			{"c6100000-0000-0000-0000-000000000003", "Bileklik", "bileklik"},
			{"c6100000-0000-0000-0000-000000000004", "Kemer", "kemer"},
			{"b6000000-0000-0000-0000-000000000002", "Kozmetik", "kozmetik"},
		},
	},
	{
		Name:   "Cocuk",
		Weight: 0.05,
		Categories: []categoryEntry{
			{"b8000000-0000-0000-0000-000000000001", "Kiz Cocuk", "kiz-cocuk"},
			{"b8000000-0000-0000-0000-000000000002", "Erkek Cocuk", "erkek-cocuk"},
		},
	},
	{
		Name:   "Buyuk Beden",
		Weight: 0.05,
		Categories: []categoryEntry{
			// Re-use the dedicated buyuk beden sub-categories
			{"b1000000-0000-0000-0000-000000000003", "Buyuk Beden Elbise", "buyuk-beden-elbise"},
			{"b3000000-0000-0000-0000-000000000003", "Buyuk Beden Abiye", "buyuk-beden-abiye"},
			// Also use the top-level Buyuk Beden category directly
			{"a0000000-0000-0000-0000-000000000005", "Buyuk Beden", "buyuk-beden"},
		},
	},
}

// ---------------------------------------------------------------------------
// Product name generation data
// ---------------------------------------------------------------------------

var prefixes = []string{
	"Duz Renk", "Cicek Desenli", "Puantiyeli", "Cizgili", "Nakisli",
	"Dantel Detayli", "Pileli", "Kusakli", "Dugmeli", "Fermuarli",
	"Bagcikli", "Tasli", "Payetli", "Kadife", "Yun Efektli",
	"Saten", "Ipek Dokulu", "Krep", "Triko", "Dokuma",
	"Baskili", "Jakar Desenli", "Simli", "Pul Islemeli", "Flok Baskili",
}

// typesPerCategory maps each leaf category slug to relevant product type names.
var typesPerCategory = map[string][]string{
	// Elbise
	"tesettur-elbise":    {"Tesettur Elbise", "Maxi Elbise", "Midi Elbise", "Uzun Elbise"},
	"triko-elbise":       {"Triko Elbise", "Orgu Elbise", "Kazak Elbise"},
	"buyuk-beden-elbise": {"Buyuk Beden Elbise", "Buyuk Beden Maxi Elbise", "Buyuk Beden Midi Elbise"},
	"namaz-elbisesi":     {"Namaz Elbisesi", "Tek Parca Namaz Elbisesi", "Fermuarli Namaz Elbisesi"},
	// Giyim - Dis Giyim
	"mont":       {"Mont", "Parka", "Sisisme Mont", "Puf Mont"},
	"kaban-manto": {"Kaban", "Manto", "Uzun Kaban", "Yun Manto"},
	"trenckot":   {"Trenckot", "Uzun Trenckot", "Kisa Trenckot"},
	"ceket":      {"Ceket", "Blazer", "Bomber Ceket", "Deri Ceket"},
	"yelek":      {"Yelek", "Sisisme Yelek", "Triko Yelek", "Deri Yelek"},
	// Giyim - Takim
	"takim": {"Ikili Takim", "Uclu Takim", "Tunik Pantolon Takim", "Tesettur Takim"},
	// Giyim - Ust Giyim
	"tunik":       {"Tunik", "Uzun Tunik", "Asimetrik Tunik", "Dugmeli Tunik"},
	"bluz-gomlek": {"Bluz", "Gomlek", "Sifon Bluz", "Poplin Gomlek"},
	"sweatshirt":  {"Sweatshirt", "Kapsonlu Sweatshirt", "Oversize Sweatshirt"},
	"tisort":      {"Tisort", "Basic Tisort", "Baskili Tisort", "Uzun Kollu Tisort"},
	// Giyim - Alt Giyim
	"pantolon":    {"Pantolon", "Kumas Pantolon", "Jogger Pantolon", "Palazzo Pantolon"},
	"etek":        {"Etek", "Midi Etek", "Maxi Etek", "Pileli Etek"},
	"kot-pantolon": {"Kot Pantolon", "Mom Jean", "Wide Leg Jean", "Skinny Jean"},
	// Giyim - Ev & Ic Giyim
	"ev-ic-giyim": {"Pijama Takimi", "Sabahlik", "Ev Elbisesi", "Gecelik"},
	// Abiye
	"abiye-elbise":      {"Abiye Elbise", "Uzun Abiye", "Balik Abiye", "Kabarik Abiye"},
	"abiye-takim":       {"Abiye Takim", "Tulum Abiye", "Abiye Kaftanli Takim"},
	"buyuk-beden-abiye": {"Buyuk Beden Abiye", "Buyuk Beden Uzun Abiye", "Buyuk Beden Tulum"},
	// Basortusu
	"sal":           {"Sal", "Pamuklu Sal", "Ipek Sal", "Yun Sal"},
	"esarp":         {"Esarp", "Twill Esarp", "Medine Ipegi Esarp", "Polyester Esarp"},
	"hazir-turban":  {"Hazir Turban", "Pileli Turban", "Simli Turban", "Kadife Turban"},
	"bone":          {"Bone", "Penye Bone", "Dantelli Bone", "Hazir Bone"},
	// Ayakkabi & Canta
	"bot-cizme":        {"Bot", "Cizme", "Deri Bot", "Suet Bot", "Biker Bot"},
	"spor-ayakkabi":    {"Spor Ayakkabi", "Sneaker", "Yuruyus Ayakkabisi"},
	"topuklu-ayakkabi": {"Topuklu Ayakkabi", "Stiletto", "Kalin Topuklu", "Platform Ayakkabi"},
	"omuz-cantasi":     {"Omuz Cantasi", "El Cantasi", "Askili Canta", "Zincir Askili Canta"},
	"sirt-cantasi":     {"Sirt Cantasi", "Mini Sirt Cantasi", "Laptop Cantasi"},
	// Aksesuar
	"kolye":    {"Kolye", "Zincir Kolye", "Tasli Kolye", "Inci Kolye"},
	"kupe":     {"Kupe", "Halka Kupe", "Sarkik Kupe", "Kucuk Tas Kupe"},
	"bileklik": {"Bileklik", "Boncuklu Bileklik", "Zincir Bileklik", "Kelepce Bileklik"},
	"kemer":    {"Kemer", "Deri Kemer", "Tokali Kemer", "Ince Kemer"},
	"kozmetik": {"Ruj", "Fondoten", "Maskara", "Goz Fari Paleti", "Allik"},
	// Cocuk
	"kiz-cocuk":   {"Kiz Cocuk Elbise", "Kiz Cocuk Takim", "Kiz Cocuk Tunik", "Kiz Cocuk Ceket"},
	"erkek-cocuk":  {"Erkek Cocuk Takim", "Erkek Cocuk Gomlek", "Erkek Cocuk Pantolon", "Erkek Cocuk Ceket"},
	// Buyuk Beden (top level)
	"buyuk-beden": {"Buyuk Beden Tunik", "Buyuk Beden Pantolon", "Buyuk Beden Gomlek", "Buyuk Beden Ceket"},
}

var colors = []string{
	"Siyah", "Lacivert", "Murdum", "Ekru", "Pembe",
	"Gri", "Haki", "Bordo", "Mavi", "Bej",
	"Kirmizi", "Yesil", "Kahverengi", "Krem", "Indigo",
	"Vizon", "Pudra", "Mint", "Hardal", "Kiremit",
}

// ---------------------------------------------------------------------------
// Description templates
// ---------------------------------------------------------------------------

var descriptionTemplates = []string{
	"Sik ve rahat %s. Gunluk ve ozel gunlerde rahatlkla kullanabilirsiniz. Kaliteli kumasiyla uzun omurlu kullanim sunar.",
	"Zarif tasarimi ile dikkat ceken %s. Her mevsim dolabinizin vazgecilmezi olacak. Kolay yikanabilir ve utulenebilir.",
	"Modern kesimi ve sik detaylariyla one cikan %s. Farkli kombinlerle kullanabilirsiniz. Tesetture uygun tasarim.",
	"Rahat kalibi ve kaliteli kumasiyla begenilen %s. Gunluk hayatta ve ozel davetlerde tercih edebilirsiniz.",
	"%s - Ozenle secilmis kumaslar ve ozenli dikis detaylari ile hazirlanmistir. Vucut tipine uygun rahat kalip.",
	"Trend %s modeli. Sezonun en populer renklerinde. Yuksek kalite standartlarinda uretilmistir.",
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// slugify converts a Turkish product name to a URL-safe slug.
func slugify(s string) string {
	replacer := strings.NewReplacer(
		" ", "-",
		"&", "ve",
		"/", "-",
	)
	lower := strings.ToLower(replacer.Replace(s))
	// Remove any non-ascii-safe characters for slug safety.
	var b strings.Builder
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			// Map common Turkish chars to ASCII equivalents.
			switch r {
			case '\u00e7': // c-cedilla
				b.WriteRune('c')
			case '\u015f': // s-cedilla
				b.WriteRune('s')
			case '\u011f': // g-breve
				b.WriteRune('g')
			case '\u0131': // dotless i
				b.WriteRune('i')
			case '\u00f6': // o-umlaut
				b.WriteRune('o')
			case '\u00fc': // u-umlaut
				b.WriteRune('u')
			case '\u00e2': // a-circumflex
				b.WriteRune('a')
			case '\u00ee': // i-circumflex
				b.WriteRune('i')
			}
		}
	}
	result := b.String()
	// Collapse multiple dashes.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}

// ---------------------------------------------------------------------------
// Product generation
// ---------------------------------------------------------------------------

type generatedProduct struct {
	ID          string
	Name        string
	Slug        string
	Description string
	BrandIdx    int
	CategoryID  string
	BasePrice   int64
	Currency    string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	TopCatName  string // for variant generation
	CatSlug     string // for variant generation
}

func generateProducts(rng *rand.Rand) []generatedProduct {
	products := make([]generatedProduct, 0, totalProducts)
	now := time.Now().UTC()

	// Build distribution: how many products per top-level category.
	type catAlloc struct {
		topIdx int
		count  int
	}
	allocs := make([]catAlloc, len(topCategories))
	remaining := totalProducts
	for i, tc := range topCategories {
		if i == len(topCategories)-1 {
			allocs[i] = catAlloc{topIdx: i, count: remaining}
		} else {
			n := int(float64(totalProducts) * tc.Weight)
			allocs[i] = catAlloc{topIdx: i, count: n}
			remaining -= n
		}
	}

	globalIdx := 0
	for _, alloc := range allocs {
		tc := topCategories[alloc.topIdx]
		for j := 0; j < alloc.count; j++ {
			// Pick a leaf category within this top-level bucket.
			cat := tc.Categories[j%len(tc.Categories)]

			// Pick product name components.
			prefix := prefixes[rng.Intn(len(prefixes))]
			types, ok := typesPerCategory[cat.Slug]
			if !ok {
				types = []string{cat.Name}
			}
			productType := types[rng.Intn(len(types))]
			color := colors[rng.Intn(len(colors))]

			name := fmt.Sprintf("%s %s - %s", prefix, productType, color)

			// Ensure slug uniqueness by appending the global index.
			baseSlug := slugify(name)
			slug := fmt.Sprintf("%s-%d", baseSlug, globalIdx)

			// Description.
			descTpl := descriptionTemplates[rng.Intn(len(descriptionTemplates))]
			description := fmt.Sprintf(descTpl, productType)

			// Price: 9900-499900 cents (TRY 99 - TRY 4,999).
			basePrice := int64(9900 + rng.Intn(490000))
			// Round to nearest 100 (e.g., 12345 -> 12300).
			basePrice = (basePrice / 100) * 100

			// Brand: deterministic rotation through all 10 brands.
			brandIdx := globalIdx % len(brands)

			// Metadata with color info.
			metadata := map[string]interface{}{
				"color":    color,
				"season":   []string{"Ilkbahar/Yaz", "Sonbahar/Kis", "Tum Sezon"}[rng.Intn(3)],
				"material": []string{"Pamuk", "Polyester", "Viskon", "Krep", "Saten", "Triko", "Deri", "Suet", "Yun"}[rng.Intn(9)],
			}

			// Random created_at within last 90 days.
			daysAgo := rng.Intn(90)
			hoursAgo := rng.Intn(24)
			minutesAgo := rng.Intn(60)
			createdAt := now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(hoursAgo)*time.Hour - time.Duration(minutesAgo)*time.Minute)

			productID := deterministicUUID("ecommerce-product", globalIdx)

			products = append(products, generatedProduct{
				ID:          productID,
				Name:        name,
				Slug:        slug,
				Description: description,
				BrandIdx:    brandIdx,
				CategoryID:  cat.ID,
				BasePrice:   basePrice,
				Currency:    "TRY",
				Metadata:    metadata,
				CreatedAt:   createdAt,
				TopCatName:  tc.Name,
				CatSlug:     cat.Slug,
			})

			globalIdx++
		}
	}

	return products
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[seed-10k] ")

	dbURL := getEnv("DATABASE_URL", "postgres://ecommerce:ecommerce_secret@localhost:5432/product_db?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// -------------------------------------------------------------------
	// 1. Connect to database
	// -------------------------------------------------------------------
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

	// -------------------------------------------------------------------
	// 2. Seed brands (idempotent via ON CONFLICT)
	// -------------------------------------------------------------------
	log.Println("Seeding brands...")
	brandIDs := make([]string, len(brands))
	for i, b := range brands {
		var id string
		err := pool.QueryRow(ctx,
			`INSERT INTO brands (name, slug, logo_url)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			 RETURNING id`,
			b.Name, b.Slug, b.LogoURL,
		).Scan(&id)
		if err != nil {
			log.Printf("  WARNING: brand %q: %v (trying SELECT)", b.Name, err)
			_ = pool.QueryRow(ctx, `SELECT id FROM brands WHERE slug = $1`, b.Slug).Scan(&id)
		}
		brandIDs[i] = id
		log.Printf("  Brand: %s (id=%s)", b.Name, id)
	}

	// -------------------------------------------------------------------
	// 3. Generate 10,000 products
	// -------------------------------------------------------------------
	log.Printf("Generating %d products...", totalProducts)
	rng := rand.New(rand.NewSource(42)) // deterministic seed
	products := generateProducts(rng)
	log.Printf("Generated %d products.", len(products))

	// -------------------------------------------------------------------
	// 4. Clean up previously seeded products (idempotent re-run)
	// -------------------------------------------------------------------
	log.Println("Cleaning up previous seed data (if any)...")
	// Collect all deterministic IDs.
	allProductIDs := make([]string, len(products))
	for i, p := range products {
		allProductIDs[i] = p.ID
	}

	// Delete in batches to avoid parameter limits.
	for start := 0; start < len(allProductIDs); start += batchSize {
		end := start + batchSize
		if end > len(allProductIDs) {
			end = len(allProductIDs)
		}
		batch := allProductIDs[start:end]

		// Build a parameterized DELETE.
		placeholders := make([]string, len(batch))
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		query := fmt.Sprintf(
			"DELETE FROM products WHERE id IN (%s)",
			strings.Join(placeholders, ", "),
		)
		_, err := pool.Exec(ctx, query, args...)
		if err != nil {
			log.Printf("  WARNING: cleanup batch %d-%d: %v", start, end, err)
		}
	}
	log.Println("  Cleanup complete.")

	// -------------------------------------------------------------------
	// 5. Insert products in batches
	// -------------------------------------------------------------------
	log.Printf("Inserting %d products in batches of %d...", totalProducts, batchSize)

	for start := 0; start < len(products); start += batchSize {
		end := start + batchSize
		if end > len(products) {
			end = len(products)
		}
		batch := products[start:end]

		// Build batch INSERT.
		var sb strings.Builder
		sb.WriteString("INSERT INTO products (id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at) VALUES ")

		args := make([]interface{}, 0, len(batch)*12)
		for i, p := range batch {
			if i > 0 {
				sb.WriteString(", ")
			}
			base := i * 12
			sb.WriteString(fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6,
				base+7, base+8, base+9, base+10, base+11, base+12,
			))

			metadataJSON, _ := json.Marshal(p.Metadata)

			args = append(args,
				p.ID,
				p.Name,
				p.Slug,
				p.Description,
				brandIDs[p.BrandIdx],
				p.CategoryID,
				"published",
				p.BasePrice,
				p.Currency,
				string(metadataJSON),
				p.CreatedAt,
				p.CreatedAt,
			)
		}

		sb.WriteString(" ON CONFLICT (id) DO NOTHING")
		_, err := pool.Exec(ctx, sb.String(), args...)
		if err != nil {
			log.Fatalf("  FATAL: insert products batch %d-%d: %v", start, end, err)
		}

		if end%1000 == 0 || end == len(products) {
			log.Printf("  Inserted %d / %d products", end, len(products))
		}
	}

	// -------------------------------------------------------------------
	// 6. Seed product images (3-5 per product)
	// -------------------------------------------------------------------
	log.Println("Inserting product images...")
	imgBatchArgs := make([]interface{}, 0, batchSize*6)
	var imgSB strings.Builder
	imgCount := 0
	imgBatchNum := 0

	flushImages := func() {
		if imgBatchNum == 0 {
			return
		}
		_, err := pool.Exec(ctx, imgSB.String(), imgBatchArgs...)
		if err != nil {
			log.Printf("  WARNING: insert images batch: %v", err)
		}
		imgBatchArgs = imgBatchArgs[:0]
		imgSB.Reset()
		imgBatchNum = 0
	}

	for i, p := range products {
		numImages := 3 + rng.Intn(3) // 3-5 images
		for j := 0; j < numImages; j++ {
			if imgBatchNum == 0 {
				imgSB.WriteString("INSERT INTO product_images (product_id, url, alt_text, sort_order, is_primary, created_at) VALUES ")
			} else {
				imgSB.WriteString(", ")
			}

			base := imgBatchNum * 6
			imgSB.WriteString(fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6,
			))

			imgURL := fmt.Sprintf("https://picsum.photos/seed/%s-%d/600/800", p.Slug, j)
			altText := fmt.Sprintf("%s - Gorsel %d", p.Name, j+1)
			isPrimary := j == 0

			imgBatchArgs = append(imgBatchArgs,
				p.ID, imgURL, altText, j, isPrimary, p.CreatedAt,
			)

			imgBatchNum++
			imgCount++

			if imgBatchNum >= batchSize {
				imgSB.WriteString(" ON CONFLICT DO NOTHING")
				flushImages()
			}
		}

		if (i+1)%2000 == 0 {
			// Flush current batch before logging.
			if imgBatchNum > 0 {
				imgSB.WriteString(" ON CONFLICT DO NOTHING")
				flushImages()
			}
			log.Printf("  Images: processed %d / %d products (%d images so far)", i+1, len(products), imgCount)
		}
	}
	// Flush remaining.
	if imgBatchNum > 0 {
		imgSB.WriteString(" ON CONFLICT DO NOTHING")
		flushImages()
	}
	log.Printf("  Inserted %d product images.", imgCount)

	// -------------------------------------------------------------------
	// 7. Seed product variants (2-6 per product)
	// -------------------------------------------------------------------
	log.Println("Inserting product variants...")

	// Size systems.
	numericSizes := []string{"36", "38", "40", "42", "44", "46"}
	letterSizes := []string{"XS", "S", "M", "L", "XL", "XXL"}
	shoeSizes := []string{"36", "37", "38", "39", "40", "41"}

	// Categories that use which size system.
	shoeCategories := map[string]bool{
		"bot-cizme": true, "spor-ayakkabi": true, "topuklu-ayakkabi": true,
	}
	noSizeCategories := map[string]bool{
		"sal": true, "esarp": true, "hazir-turban": true, "bone": true,
		"kolye": true, "kupe": true, "bileklik": true, "kemer": true,
		"kozmetik": true, "omuz-cantasi": true, "sirt-cantasi": true,
	}
	casualCategories := map[string]bool{
		"tisort": true, "sweatshirt": true, "ev-ic-giyim": true,
		"kiz-cocuk": true, "erkek-cocuk": true,
	}

	varBatchArgs := make([]interface{}, 0, batchSize*8)
	var varSB strings.Builder
	varBatchNum := 0
	varCount := 0

	flushVariants := func() {
		if varBatchNum == 0 {
			return
		}
		_, err := pool.Exec(ctx, varSB.String(), varBatchArgs...)
		if err != nil {
			log.Printf("  WARNING: insert variants batch: %v", err)
		}
		varBatchArgs = varBatchArgs[:0]
		varSB.Reset()
		varBatchNum = 0
	}

	for i, p := range products {
		brandSlug := strings.ToUpper(brands[p.BrandIdx].Slug[:3])
		catSlugShort := strings.ToUpper(p.CatSlug)
		if len(catSlugShort) > 6 {
			catSlugShort = catSlugShort[:6]
		}

		// Determine sizes and count.
		var sizes []string
		switch {
		case noSizeCategories[p.CatSlug]:
			// No sizes: create 2-3 color-only variants.
			numVariants := 2 + rng.Intn(2)
			sizes = make([]string, numVariants)
			for vi := 0; vi < numVariants; vi++ {
				sizes[vi] = "STD"
			}
		case shoeCategories[p.CatSlug]:
			numVariants := 3 + rng.Intn(4) // 3-6
			if numVariants > len(shoeSizes) {
				numVariants = len(shoeSizes)
			}
			sizes = shoeSizes[:numVariants]
		case casualCategories[p.CatSlug]:
			numVariants := 3 + rng.Intn(4) // 3-6
			if numVariants > len(letterSizes) {
				numVariants = len(letterSizes)
			}
			sizes = letterSizes[:numVariants]
		default:
			numVariants := 3 + rng.Intn(4) // 3-6
			if numVariants > len(numericSizes) {
				numVariants = len(numericSizes)
			}
			sizes = numericSizes[:numVariants]
		}

		// Decide if this product has a discounted variant (30% of products).
		hasDiscount := rng.Float64() < 0.30
		discountPct := 0.0
		if hasDiscount {
			discountPct = 0.10 + rng.Float64()*0.30 // 10-40% off
		}

		// Pick variant color (from product metadata or random).
		variantColor := p.Metadata["color"].(string)

		for vi, size := range sizes {
			if varBatchNum == 0 {
				varSB.WriteString("INSERT INTO product_variants (product_id, sku, name, price, attributes, weight_grams, is_active, created_at, updated_at) VALUES ")
			} else {
				varSB.WriteString(", ")
			}

			base := varBatchNum * 9
			varSB.WriteString(fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9,
			))

			sku := fmt.Sprintf("%s-%s-%05d-%s", brandSlug, catSlugShort, i, size)
			if noSizeCategories[p.CatSlug] {
				// For no-size categories, use color in SKU to differentiate.
				extraColor := colors[rng.Intn(len(colors))]
				colorSlug := slugify(extraColor)
				sku = fmt.Sprintf("%s-%s-%05d-%s", brandSlug, catSlugShort, i, colorSlug)
				variantColor = extraColor
			}

			varName := size
			if noSizeCategories[p.CatSlug] {
				varName = variantColor
			}

			// Variant price: nil means same as base_price, or discounted.
			var varPrice *int64
			if hasDiscount && vi == 0 {
				discounted := int64(float64(p.BasePrice) * (1 - discountPct))
				discounted = (discounted / 100) * 100 // round
				varPrice = &discounted
			}

			// Attributes JSONB.
			attrs := map[string]string{
				"color": variantColor,
			}
			if !noSizeCategories[p.CatSlug] {
				attrs["size"] = size
			}
			attrsJSON, _ := json.Marshal(attrs)

			// Weight: 100-2000 grams depending on category.
			weight := 200 + rng.Intn(800)
			if shoeCategories[p.CatSlug] {
				weight = 400 + rng.Intn(600)
			}

			varBatchArgs = append(varBatchArgs,
				p.ID, sku, varName, varPrice, string(attrsJSON), weight, true, p.CreatedAt, p.CreatedAt,
			)

			varBatchNum++
			varCount++

			if varBatchNum >= batchSize {
				varSB.WriteString(" ON CONFLICT (sku) DO NOTHING")
				flushVariants()
			}
		}

		if (i+1)%2000 == 0 {
			if varBatchNum > 0 {
				varSB.WriteString(" ON CONFLICT (sku) DO NOTHING")
				flushVariants()
			}
			log.Printf("  Variants: processed %d / %d products (%d variants so far)", i+1, len(products), varCount)
		}
	}
	// Flush remaining.
	if varBatchNum > 0 {
		varSB.WriteString(" ON CONFLICT (sku) DO NOTHING")
		flushVariants()
	}
	log.Printf("  Inserted %d product variants.", varCount)

	// -------------------------------------------------------------------
	// 8. Update category product_count
	// -------------------------------------------------------------------
	log.Println("Updating category product counts...")
	_, err = pool.Exec(ctx, `
		UPDATE categories c SET product_count = (
			SELECT COUNT(*) FROM products p WHERE p.category_id = c.id
		)
	`)
	if err != nil {
		log.Printf("  WARNING: update category counts: %v", err)
	} else {
		log.Println("  Category product counts updated.")
	}

	// -------------------------------------------------------------------
	// Done
	// -------------------------------------------------------------------
	log.Printf("Seed complete! Inserted %d products, ~%d images, ~%d variants.", len(products), imgCount, varCount)
}
