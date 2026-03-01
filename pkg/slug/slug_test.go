package slug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate_BasicASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"foo bar baz", "foo-bar-baz"},
		{"Simple", "simple"},
		{"ALL UPPER CASE", "all-upper-case"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Generate(tt.input))
		})
	}
}

func TestGenerate_TurkishCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Kadın Giyim", "kadin-giyim"},
		{"Çocuk Ürünleri", "cocuk-urunleri"},
		{"Güneş Gözlüğü", "gunes-gozlugu"},
		{"Şeker Bayramı", "seker-bayrami"},
		{"İstanbul", "istanbul"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Generate(tt.input))
		})
	}
}

func TestGenerate_SpecialCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello!!! World???", "hello-world"},
		{"foo@bar#baz", "foo-bar-baz"},
		{"price: $100", "price-100"},
		{"one & two", "one-two"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Generate(tt.input))
		})
	}
}

func TestGenerate_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"leading spaces", "   hello world   ", "hello-world"},
		{"multiple spaces", "hello   world", "hello-world"},
		{"tabs and spaces", "hello\t\tworld", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Generate(tt.input))
		})
	}
}

func TestGenerate_EdgeCases(t *testing.T) {
	assert.Equal(t, "", Generate(""))
	assert.Equal(t, "", Generate("   "))
	assert.Equal(t, "", Generate("!!!"))
	assert.Equal(t, "a", Generate("a"))
	assert.Equal(t, "123", Generate("123"))
}

func TestGenerate_ConsecutiveHyphens(t *testing.T) {
	assert.Equal(t, "a-b", Generate("a---b"))
	assert.Equal(t, "a-b", Generate("a - - b"))
}

func TestGenerate_NoLeadingTrailingHyphens(t *testing.T) {
	assert.Equal(t, "hello", Generate("-hello-"))
	assert.Equal(t, "hello", Generate("!hello!"))
}
