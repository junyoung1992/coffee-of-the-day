package domain

import "encoding/json"

// StringsToJSON converts a string slice into a JSON array string.
func StringsToJSON(ss []string) string {
	if ss == nil {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

// JSONToStrings converts a JSON array string into a string slice.
func JSONToStrings(s string) []string {
	var out []string
	json.Unmarshal([]byte(s), &out)
	if out == nil {
		out = []string{}
	}
	return out
}
