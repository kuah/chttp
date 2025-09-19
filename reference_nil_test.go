package chttp

import (
	"bytes"
	"net/http"
	"testing"
)

// Minimal types to reproduce the issue
type messageReference struct {
	Id *string `json:"id,omitempty"`
}

type postMessageReq struct {
	Type      *string           `json:"type,omitempty" v:"required"`
	Reference *messageReference `json:"reference,omitempty"`
}

// The expectation: when the JSON body omits "reference", Reference should remain nil
func TestPointerStructOmittedFieldRemainsNil(t *testing.T) {
	body := `{"type":"text"}`
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	result, parserResult, err := Valid[postMessageReq](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parserResult != ParserResultSuccess {
		t.Fatalf("expected ParserResultSuccess, got %v", parserResult)
	}

	if result.Reference != nil {
		t.Fatalf("expected Reference to be nil when omitted in JSON, got non-nil: %+v", result.Reference)
	}
}
