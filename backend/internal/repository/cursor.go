package repository

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// Cursor holds the fields encoded into the opaque pagination cursor.
// Fields: sort_by, order, sort_value, id — base64-encoded JSON.
type Cursor struct {
	SortBy    string `json:"sort_by"`
	Order     string `json:"order"`
	SortValue string `json:"sort_value"`
	ID        string `json:"id"`
}

// EncodeCursor serialises a Cursor into an opaque base64 string.
func EncodeCursor(c Cursor) string {
	raw, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(raw)
}

// DecodeCursor parses an opaque base64 string back into a Cursor.
func DecodeCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, errors.New("empty cursor string")
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, err
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return Cursor{}, err
	}
	if c.SortBy == "" || c.Order == "" || c.ID == "" {
		return Cursor{}, errors.New("cursor missing required fields")
	}
	return c, nil
}
