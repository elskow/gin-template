package response

import (
	"encoding/json"
	"testing"
)

func TestSuccessResponse(t *testing.T) {
	type TestOutput struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	output := TestOutput{
		ID:   "123",
		Name: "Test",
	}

	resp := Success(output)

	if resp.Error != nil {
		t.Error("Success response should not have error")
	}

	if resp.Output == nil {
		t.Error("Success response should have output")
	}

	if resp.Output.ID != "123" || resp.Output.Name != "Test" {
		t.Error("Output data mismatch")
	}
}

func TestErrorResponse(t *testing.T) {
	type TestOutput struct {
		ID string `json:"id"`
	}

	resp := Error[TestOutput]("TEST_ERROR", "This is a test error")

	if resp.Error == nil {
		t.Error("Error response should have error")
	}

	if resp.Output != nil {
		t.Error("Error response should not have output")
	}

	if resp.Error.ErrorCode != "TEST_ERROR" {
		t.Error("Error code mismatch")
	}

	if resp.Error.ErrorMessage != "This is a test error" {
		t.Error("Error message mismatch")
	}
}

func TestResponseJSONSerialization(t *testing.T) {
	type TestOutput struct {
		CustomerID int    `json:"customer_id"`
		Name       string `json:"name"`
	}

	output := TestOutput{
		CustomerID: 123,
		Name:       "John Doe",
	}

	resp := Success(output)
	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal response: %v", err)
	}

	var unmarshaled Response[TestOutput]
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.Output.CustomerID != 123 {
		t.Error("CustomerID mismatch after unmarshaling")
	}
}

func TestErrorResponseJSONSerialization(t *testing.T) {
	type TestOutput struct {
		ID string `json:"id"`
	}

	resp := Error[TestOutput]("INVALID_INPUT", "Invalid input provided")
	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal error response: %v", err)
	}

	var unmarshaled Response[TestOutput]
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal error response: %v", err)
	}

	if unmarshaled.Error.ErrorCode != "INVALID_INPUT" {
		t.Error("Error code mismatch after unmarshaling")
	}

	if unmarshaled.Error.ErrorMessage != "Invalid input provided" {
		t.Error("Error message mismatch after unmarshaling")
	}

	if unmarshaled.Output != nil {
		t.Error("Error response should not have output after unmarshaling")
	}
}
