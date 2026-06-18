package ttclient

import "encoding/json"

// Pagination mirrors the tastytrade paged-list envelope.
type Pagination struct {
	PerPage          int `json:"per-page"`
	PageOffset       int `json:"page-offset"`
	TotalItems       int `json:"total-items"`
	TotalPages       int `json:"total-pages"`
	CurrentItemCount int `json:"current-item-count"`
}

// DecodeData unwraps {"data": <out>} from raw JSON.
func DecodeData(raw []byte, out any) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	return json.Unmarshal(env.Data, out)
}

// DecodeItems unwraps {"data": {"items": <out>}, "pagination": ...}.
func DecodeItems(raw []byte, out any) (Pagination, error) {
	var env struct {
		Data struct {
			Items json.RawMessage `json:"items"`
		} `json:"data"`
		Pagination Pagination `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return Pagination{}, err
	}
	if len(env.Data.Items) == 0 {
		return env.Pagination, nil
	}
	return env.Pagination, json.Unmarshal(env.Data.Items, out)
}
