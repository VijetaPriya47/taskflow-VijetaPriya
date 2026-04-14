package validator

import (
	"strings"
	"time"
)

// V is a small request validator that accumulates field errors.
// It is intentionally framework-agnostic and supports cross-field checks.
type V struct {
	Fields map[string]string
}

func New() *V {
	return &V{Fields: map[string]string{}}
}

func (v *V) Ok() bool { return len(v.Fields) == 0 }

func (v *V) add(field, msg string) {
	if field == "" {
		field = "_"
	}
	if _, exists := v.Fields[field]; exists {
		return
	}
	v.Fields[field] = msg
}

func (v *V) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(field, "is required")
	}
}

func (v *V) OneOf(field, value string, allowed ...string) {
	val := strings.TrimSpace(value)
	if val == "" {
		return
	}
	for _, a := range allowed {
		if val == a {
			return
		}
	}
	v.add(field, "is invalid")
}

func (v *V) DateYYYYMMDD(field, value string) *time.Time {
	val := strings.TrimSpace(value)
	if val == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		v.add(field, "must be YYYY-MM-DD")
		return nil
	}
	return &t
}

// RequireAllOrNone enforces that either all fields are present/non-empty or all are empty.
func (v *V) RequireAllOrNone(msg string, pairs ...struct{ Field, Value string }) {
	nonEmpty := 0
	for _, p := range pairs {
		if strings.TrimSpace(p.Value) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 0 && nonEmpty != len(pairs) {
		v.add("_", msg)
	}
}

