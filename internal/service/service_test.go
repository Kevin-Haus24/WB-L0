package service

import (
	_ "embed"
	"encoding/json"
	"testing"

	"L0/internal/dto"
)

//go:embed testdata/model.json
var sampleOrder []byte

//go:embed testdata/model2.json
var sampleInvalidType []byte

//go:embed testdata/model3.json
var sampleMinimal []byte

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

	var obj map[string]any
	if err := json.Unmarshal(sampleMinimal, &obj); err != nil {
		t.Fatalf("unmarshal sample: %v", err)
	}
	obj["order_uid"] = ""
	payload, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}

	_, _, err = decode(payload)
	if err == nil {
		t.Fatal("expected error for missing order_uid")
	}
}

func TestDecodeInvalidType(t *testing.T) {
	if _, _, err := decode(sampleInvalidType); err == nil {
		t.Fatal("expected error for invalid field types")
	}
}

func TestDecodeMinimalOrder(t *testing.T) {
	order, normalized, err := decode(sampleMinimal)
	if err != nil {
		t.Fatalf("decode minimal failed: %v", err)
	}

	if order.OrderUID == "" {
		t.Fatal("expected order uid to be preserved")
	}

	var dtoOrder dto.Order
	if err := json.Unmarshal(normalized, &dtoOrder); err != nil {
		t.Fatalf("normalized minimal invalid: %v", err)
	}
	if dtoOrder.OrderUID != order.OrderUID {
		t.Fatalf("expected matching ids, got %s vs %s", dtoOrder.OrderUID, order.OrderUID)
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
