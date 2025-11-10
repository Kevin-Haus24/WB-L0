package cache

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCacheSetGet(t *testing.T) {
	c := New()
	payload := json.RawMessage(`{"id":1}`)

	c.Set("order-1", payload)

	got, ok := c.Get("order-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload: %s", string(got))
	}
}

func TestCacheLoadAll(t *testing.T) {
	c := New()
	bulk := map[string]json.RawMessage{
		"one":   json.RawMessage(`{"val":1}`),
		"two":   json.RawMessage(`{"val":2}`),
		"three": json.RawMessage(`{"val":3}`),
	}

	c.LoadAll(bulk)

	for key, expected := range bulk {
		got, ok := c.Get(key)
		if !ok {
			t.Fatalf("expected %s in cache", key)
		}
		if !bytes.Equal(got, expected) {
			t.Fatalf("unexpected value for %s", key)
		}
	}
}
