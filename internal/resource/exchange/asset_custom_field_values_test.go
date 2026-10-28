package exchange

import (
	"reflect"
	"testing"
)

func TestKeepEquivalentPriorValues(t *testing.T) {
	cases := []struct {
		api, prior, want []string
	}{
		{[]string{"2026-10-26T00:00:00.000Z"}, []string{"2026-10-26"}, []string{"2026-10-26"}},
		{[]string{"5"}, []string{"5.0"}, []string{"5.0"}},
		{[]string{"1", "2"}, []string{"2", "1"}, []string{"1", "2"}}, // order handled by reorderByKey
		{[]string{"v2"}, []string{"v1"}, []string{"v2"}},             // a real change stays visible
		{[]string{"2026-10-27T00:00:00.000Z"}, []string{"2026-10-26"}, []string{"2026-10-27T00:00:00.000Z"}},
		{[]string{"a"}, nil, []string{"a"}},
	}
	for _, c := range cases {
		if got := keepEquivalentPriorValues(c.api, c.prior); !reflect.DeepEqual(got, c.want) {
			t.Errorf("api=%v prior=%v: got %v, want %v", c.api, c.prior, got, c.want)
		}
	}
}
