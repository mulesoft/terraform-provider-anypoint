package exchange

import (
	"strconv"
	"time"
)

// keepEquivalentPriorValues returns the API's custom-field values, substituting the
// prior (configured) spelling wherever it denotes the same value. Exchange normalises
// on write: dates come back as ISO-8601 instants ("2026-10-26" -> "2026-10-26T00:00:00.000Z")
// and numbers as JSON numbers ("5.0" -> "5"). Without this, Create would fail with
// "Provider produced inconsistent result after apply" and Read would show a diff forever.
func keepEquivalentPriorValues(api, prior []string) []string {
	if len(prior) == 0 {
		return api
	}
	used := make([]bool, len(prior))
	out := make([]string, len(api))
	for i, v := range api {
		out[i] = v
		for j, p := range prior {
			if !used[j] && sameCustomFieldValue(p, v) {
				out[i], used[j] = p, true
				break
			}
		}
	}
	return out
}

func sameCustomFieldValue(configured, api string) bool {
	if configured == api {
		return true
	}
	if a, err := strconv.ParseFloat(configured, 64); err == nil {
		if b, err := strconv.ParseFloat(api, 64); err == nil {
			return a == b
		}
	}
	ta, okA := parseCustomFieldDate(configured)
	tb, okB := parseCustomFieldDate(api)
	return okA && okB && ta.Equal(tb)
}

func parseCustomFieldDate(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
