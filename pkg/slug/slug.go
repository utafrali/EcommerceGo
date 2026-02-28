package slug

import (
	"regexp"
	"strings"
)

var slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)

// Generate creates a URL-friendly slug from the given name.
// Supports Turkish characters by transliterating them to ASCII equivalents.
//
// Examples:
//   - "Kadın Giyim" → "kadin-giyim"
//   - "Çocuk Ürünleri" → "cocuk-urunleri"
//   - "Hello   World!" → "hello-world"
func Generate(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))

	// Transliterate Turkish characters to ASCII
	replacer := strings.NewReplacer(
		"ç", "c", "ğ", "g", "ı", "i", "ö", "o", "ş", "s", "ü", "u",
		"\u00e7", "c", // ç (Unicode escape)
		"\u011f", "g", // ğ
		"\u0131", "i", // ı
		"\u00f6", "o", // ö
		"\u015f", "s", // ş
		"\u00fc", "u", // ü
	)
	slug = replacer.Replace(slug)

	// Replace any non-alphanumeric characters with hyphens
	slug = slugRegexp.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Collapse consecutive hyphens into single hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	return slug
}
