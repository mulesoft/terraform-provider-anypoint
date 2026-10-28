package exchange

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Shapes captured from the live asset GET (prod, 2026-10-26).
func TestCustomFieldUnmarshal_TypedValueKeys(t *testing.T) {
	cases := map[string]interface{}{
		`{"key":"k","dataType":"text","textValue":"v1"}`:                       "v1",
		`{"key":"k","dataType":"number","numberValue":5}`:                      "5",
		`{"key":"k","dataType":"number","numberValue":5.25}`:                   "5.25",
		`{"key":"k","dataType":"date","dateValue":"2026-10-26T00:00:00.000Z"}`: "2026-10-26T00:00:00.000Z",
		`{"key":"k","dataType":"enum","enumValue":["red"]}`:                    []string{"red"},
		`{"key":"k","dataType":"text-list","textListValue":["a","b"]}`:         []string{"a", "b"},
		`{"key":"k","dataType":"number-list","numberListValue":[1,2]}`:         []string{"1", "2"},
		`{"key":"k","value":["x","y"]}`:                                        []string{"x", "y"},
		`{"key":"k","value":"x"}`:                                              "x",
	}
	for in, want := range cases {
		var f CustomField
		if err := json.Unmarshal([]byte(in), &f); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if f.Key != "k" || !reflect.DeepEqual(f.Value, want) {
			t.Errorf("%s: got key=%q value=%#v, want %#v", in, f.Key, f.Value, want)
		}
	}
	var none CustomField
	if err := json.Unmarshal([]byte(`{"key":"k","dataType":"text"}`), &none); err != nil || none.Value != nil {
		t.Errorf("no value: got %#v, %v", none.Value, err)
	}
}

func tagServer(t *testing.T, handle func(body map[string]json.RawMessage) (int, string)) (*AssetClient, *[]string) {
	t.Helper()
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":400,"message":"request/body must be object"}`))
			return
		}
		st, resp := handle(m)
		w.WriteHeader(st)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return &AssetClient{BaseURL: srv.URL, Token: "t", HTTPClient: &http.Client{}}, &bodies
}

func TestSetCategory_ObjectBodyAnd201(t *testing.T) {
	c, bodies := tagServer(t, func(m map[string]json.RawMessage) (int, string) {
		if !strings.HasPrefix(string(m["tagValue"]), "[") {
			return http.StatusBadRequest, `{"message":"request/body/tagValue must be array"}`
		}
		return http.StatusCreated, `{}`
	})
	if err := c.SetCategory(context.Background(), "g", "a", "1.0.0", "cat", []string{"Alpha", "Beta"}); err != nil {
		t.Fatal(err)
	}
	if (*bodies)[0] != `{"tagValue":["Alpha","Beta"]}` {
		t.Errorf("body = %s", (*bodies)[0])
	}
}

func TestSetCustomField_ScalarThenArrayFallback(t *testing.T) {
	// enum-like field: scalar rejected, array accepted.
	c, bodies := tagServer(t, func(m map[string]json.RawMessage) (int, string) {
		if strings.HasPrefix(string(m["tagValue"]), "[") {
			return http.StatusCreated, `{}`
		}
		return http.StatusBadRequest, `{"status":400,"message":"Invalid value for dataType: enum."}`
	})
	if err := c.SetCustomField(context.Background(), "g", "a", "1.0.0", "f", []string{"red"}); err != nil {
		t.Fatal(err)
	}
	if got := *bodies; len(got) != 2 || got[0] != `{"tagValue":"red"}` || got[1] != `{"tagValue":["red"]}` {
		t.Errorf("requests = %v", got)
	}

	// text-like field: scalar accepted first time.
	c2, bodies2 := tagServer(t, func(m map[string]json.RawMessage) (int, string) {
		if strings.HasPrefix(string(m["tagValue"]), "[") {
			return http.StatusBadRequest, `{"status":400,"message":"Invalid value for dataType: text."}`
		}
		return http.StatusCreated, `{}`
	})
	if err := c2.SetCustomField(context.Background(), "g", "a", "1.0.0", "f", []string{"v1"}); err != nil || len(*bodies2) != 1 {
		t.Fatalf("err=%v requests=%v", err, *bodies2)
	}

	// several values: array only, and a non-dataType 400 is returned without a retry.
	c3, bodies3 := tagServer(t, func(m map[string]json.RawMessage) (int, string) {
		return http.StatusBadRequest, `{"status":400,"message":"Invalid value for dataType: text."}`
	})
	if err := c3.SetCustomField(context.Background(), "g", "a", "1.0.0", "f", []string{"a", "b"}); err == nil || len(*bodies3) != 1 || (*bodies3)[0] != `{"tagValue":["a","b"]}` {
		t.Fatalf("err=%v requests=%v", err, *bodies3)
	}
	c4, bodies4 := tagServer(t, func(m map[string]json.RawMessage) (int, string) {
		return http.StatusBadRequest, `{"status":400,"message":"tag not configured"}`
	})
	if err := c4.SetCustomField(context.Background(), "g", "a", "1.0.0", "f", []string{"a"}); err == nil || len(*bodies4) != 1 {
		t.Fatalf("err=%v requests=%v", err, *bodies4)
	}
}
