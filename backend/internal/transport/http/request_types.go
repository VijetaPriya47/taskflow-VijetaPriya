package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func parsePageLimit(r *http.Request) (limit, offset int, fields map[string]string) {
	fields = map[string]string{}
	page := 1
	limit = 50

	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			fields["page"] = "must be a positive integer"
		} else {
			page = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			fields["limit"] = "must be a positive integer"
		} else if n > 200 {
			fields["limit"] = "must be <= 200"
		} else {
			limit = n
		}
	}

	offset = (page - 1) * limit
	if len(fields) > 0 {
		return 0, 0, fields
	}
	return limit, offset, nil
}
