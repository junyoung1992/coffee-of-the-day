package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// StringsToJSON
// ---------------------------------------------------------------------------

func TestStringsToJSON_NilSlice_ReturnsEmptyJSONArray(t *testing.T) {
	result := StringsToJSON(nil)
	assert.Equal(t, "[]", result)
}

func TestStringsToJSON_EmptySlice_ReturnsEmptyJSONArray(t *testing.T) {
	result := StringsToJSON([]string{})
	assert.Equal(t, "[]", result)
}

func TestStringsToJSON_SingleElement_ReturnsValidJSONArray(t *testing.T) {
	result := StringsToJSON([]string{"espresso"})
	assert.Equal(t, `["espresso"]`, result)

	// Result must be valid JSON.
	assert.True(t, json.Valid([]byte(result)), "output should be valid JSON")
}

func TestStringsToJSON_MultipleElements_ReturnsValidJSONArray(t *testing.T) {
	result := StringsToJSON([]string{"fruity", "sweet", "floral"})
	assert.Equal(t, `["fruity","sweet","floral"]`, result)

	assert.True(t, json.Valid([]byte(result)), "output should be valid JSON")
}

func TestStringsToJSON_SpecialCharacters_EscapedProperly(t *testing.T) {
	input := []string{
		`say "hello"`,    // double quotes
		`back\slash`,     // backslash
		"café ☕",        // unicode
		"line\nnewline", // newline
	}

	result := StringsToJSON(input)

	// Must be valid JSON.
	assert.True(t, json.Valid([]byte(result)), "output should be valid JSON")

	// Round-trip: unmarshal back and compare.
	var decoded []string
	err := json.Unmarshal([]byte(result), &decoded)
	assert.NoError(t, err)
	assert.Equal(t, input, decoded)
}

// ---------------------------------------------------------------------------
// JSONToStrings
// ---------------------------------------------------------------------------

func TestJSONToStrings_ValidJSONArray_ReturnsSlice(t *testing.T) {
	result := JSONToStrings(`["fruity","sweet","floral"]`)
	assert.Equal(t, []string{"fruity", "sweet", "floral"}, result)
}

func TestJSONToStrings_EmptyJSONArray_ReturnsEmptySlice(t *testing.T) {
	result := JSONToStrings("[]")
	assert.Empty(t, result)
	assert.NotNil(t, result, "should return empty slice, not nil")
}

func TestJSONToStrings_NullLiteral_ReturnsEmptySlice(t *testing.T) {
	result := JSONToStrings("null")
	assert.Empty(t, result)
	assert.NotNil(t, result, "should return empty slice, not nil")
}

func TestJSONToStrings_InvalidJSON_ReturnsEmptySlice(t *testing.T) {
	result := JSONToStrings("not json at all")
	assert.Empty(t, result)
	assert.NotNil(t, result, "should return empty slice, not nil")
}

func TestJSONToStrings_EmptyString_ReturnsEmptySlice(t *testing.T) {
	result := JSONToStrings("")
	assert.Empty(t, result)
	assert.NotNil(t, result, "should return empty slice, not nil")
}
