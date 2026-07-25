package ttclient

import (
	"encoding/json"
	"fmt"
)

// apiError mirrors tastytrade's error envelope:
// {"error":{"code":"...","message":"...","errors":[...]}}
type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// HTTPError builds an error for a non-2xx response. status is the HTTP code,
// body is the raw response body (may be empty or non-JSON).
func HTTPError(status int, body []byte) error {
	var ae apiError
	if json.Unmarshal(body, &ae) == nil && ae.Error.Message != "" {
		return fmt.Errorf("tastytrade %d: %s (%s)", status, ae.Error.Message, ae.Error.Code)
	}
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return fmt.Errorf("tastytrade %d: %s", status, snippet)
}

// IsSuccess reports whether status is 2xx.
func IsSuccess(status int) bool { return status >= 200 && status < 300 }
