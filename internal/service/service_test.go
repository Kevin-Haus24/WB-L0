package service

import (
	_ "embed"
	"encoding/json"
	"testing"

	"L0/internal/dto"
)

//go:embed testdata/order.json
var sampleOrder []byte

func TestDecodeValid(t *testing.T) {
	t.Helper()

	order, normalized, err := decode(sampleOrder)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if order.OrderUID == "" {
		t.Fatal("expected order uid to be populated")
	}

	var dtoOrder dto.Order
	if err := json.Unmarshal(normalized, &dtoOrder); err != nil {
		t.Fatalf("normalized payload invalid: %v", err)
	}

	if dtoOrder.OrderUID != order.OrderUID {
		t.Fatalf("expected matching ids, got %s vs %s", dtoOrder.OrderUID, order.OrderUID)
	}
}

func TestDecodeMissingUID(t *testing.T) {
	t.Helper()

	_, _, err := decode([]byte(`{"track_number":"123"}`))
	if err == nil {
		t.Fatal("expected error for missing order_uid")
	}
}

func TestNormalizeMatchesDecode(t *testing.T) {
	t.Helper()

	_, normalized, err := decode(sampleOrder)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	fromRaw, err := normalize(sampleOrder)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}

	if string(normalized) != string(fromRaw) {
		t.Fatalf("expected normalize consistency")
	}
}
