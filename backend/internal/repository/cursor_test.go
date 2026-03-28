package repository

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// EncodeCursor / DecodeCursor round-trip
// ---------------------------------------------------------------------------

func TestEncodeDecode_RoundTrip_AllFieldsPreserved(t *testing.T) {
	original := Cursor{
		SortBy:    "recorded_at",
		Order:     "desc",
		SortValue: "2026-03-28T10:00:00Z",
		ID:        "abc-123",
	}

	encoded := EncodeCursor(original)
	require.NotEmpty(t, encoded, "encoded cursor must not be empty")

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestEncodeDecode_RoundTrip_SpecialCharacters(t *testing.T) {
	original := Cursor{
		SortBy:    "recorded_at",
		Order:     "asc",
		SortValue: `café "special" ☕ back\slash`,
		ID:        "id/with+special=chars",
	}

	encoded := EncodeCursor(original)
	require.NotEmpty(t, encoded, "encoded cursor must not be empty")

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

// ---------------------------------------------------------------------------
// EncodeCursor output format
// ---------------------------------------------------------------------------

func TestEncodeCursor_OutputIsValidBase64(t *testing.T) {
	c := Cursor{
		SortBy:    "recorded_at",
		Order:     "desc",
		SortValue: "2026-03-28T10:00:00Z",
		ID:        "abc-123",
	}

	encoded := EncodeCursor(c)
	require.NotEmpty(t, encoded)

	// Must be valid base64.
	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err, "encoded cursor must be valid base64")

	// Underlying payload must be valid JSON containing all fields.
	var payload map[string]interface{}
	err = json.Unmarshal(raw, &payload)
	require.NoError(t, err, "base64 payload must be valid JSON")

	assert.Equal(t, "recorded_at", payload["sort_by"])
	assert.Equal(t, "desc", payload["order"])
	assert.Equal(t, "2026-03-28T10:00:00Z", payload["sort_value"])
	assert.Equal(t, "abc-123", payload["id"])
}

// ---------------------------------------------------------------------------
// DecodeCursor error cases
// ---------------------------------------------------------------------------

func TestDecodeCursor_EmptyString_ReturnsError(t *testing.T) {
	_, err := DecodeCursor("")
	assert.Error(t, err, "empty string must produce an error")
}

func TestDecodeCursor_InvalidBase64_ReturnsError(t *testing.T) {
	_, err := DecodeCursor("not!valid!base64!!!")
	assert.Error(t, err, "invalid base64 must produce an error")
}

func TestDecodeCursor_ValidBase64ButInvalidJSON_ReturnsError(t *testing.T) {
	notJSON := base64.URLEncoding.EncodeToString([]byte("this is not json"))
	_, err := DecodeCursor(notJSON)
	assert.Error(t, err, "valid base64 with non-JSON payload must produce an error")
}

func TestDecodeCursor_MissingFields_ReturnsError(t *testing.T) {
	// JSON with only some fields — missing sort_by and order.
	incomplete := map[string]string{"id": "abc-123"}
	raw, _ := json.Marshal(incomplete)
	encoded := base64.URLEncoding.EncodeToString(raw)

	_, err := DecodeCursor(encoded)
	assert.Error(t, err, "cursor with missing required fields must produce an error")
}
