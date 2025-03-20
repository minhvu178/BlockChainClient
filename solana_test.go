package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock RPC client for testing
type mockRPCClient struct {
	latestSlot   uint64
	blockDetails json.RawMessage
	shouldFail   bool
	errorMessage string
}

func (m *mockRPCClient) getLatestSlot() (uint64, error) {
	if m.shouldFail {
		return 0, fmt.Errorf(m.errorMessage)
	}
	return m.latestSlot, nil
}

func (m *mockRPCClient) getBlockDetails(slot uint64) (json.RawMessage, error) {
	if m.shouldFail {
		return nil, fmt.Errorf(m.errorMessage)
	}
	return m.blockDetails, nil
}

func TestHandleGetLatestSlot(t *testing.T) {
	tests := []struct {
		name           string
		mockClient     mockRPCClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			mockClient:     mockRPCClient{latestSlot: 12345678, shouldFail: false},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"latest_block":12345678}`,
		},
		{
			name:           "RPC Error",
			mockClient:     mockRPCClient{shouldFail: true, errorMessage: "RPC connection failed"},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "RPC connection failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request to pass to our handler
			req := httptest.NewRequest("GET", "/latest-block", nil)
			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler
			handler := handleGetLatestSlot(&tt.mockClient)
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Check the response body
			if rr.Body.String() != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleGetBlockDetails(t *testing.T) {
	mockBlockDetails := json.RawMessage(`{"parentSlot":12345677,"transactions":[{"meta":{"fee":5000},"transaction":{"message":{"accountKeys":["abc123"]}}}]}`)

	tests := []struct {
		name           string
		mockClient     mockRPCClient
		queryParam     string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "Success",
			mockClient:     mockRPCClient{blockDetails: mockBlockDetails, shouldFail: false},
			queryParam:     "?block=12345678",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if body != string(mockBlockDetails) {
					t.Errorf("handler returned unexpected body: got %v want %v", body, string(mockBlockDetails))
				}
			},
		},
		{
			name:           "Missing Block Parameter",
			mockClient:     mockRPCClient{},
			queryParam:     "",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "block parameter is required\n" {
					t.Errorf("handler returned unexpected body: got %v want %v", body, "block parameter is required\n")
				}
			},
		},
		{
			name:           "Invalid Block Number",
			mockClient:     mockRPCClient{},
			queryParam:     "?block=invalid",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "invalid block number\n" {
					t.Errorf("handler returned unexpected body: got %v want %v", body, "invalid block number\n")
				}
			},
		},
		{
			name:           "RPC Error",
			mockClient:     mockRPCClient{shouldFail: true, errorMessage: "RPC connection failed"},
			queryParam:     "?block=12345678",
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if body != "RPC connection failed\n" {
					t.Errorf("handler returned unexpected body: got %v want %v", body, "RPC connection failed\n")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request to pass to our handler
			req := httptest.NewRequest("GET", "/block-details"+tt.queryParam, nil)
			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler
			handler := handleGetBlockDetails(&tt.mockClient)
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Check the response body
			tt.checkBody(t, rr.Body.String())
		})
	}
}

// TestNewRPCClient tests the creation of a new RPC client
func TestNewRPCClient(t *testing.T) {
	client := newRPCClient("https://test-endpoint.com")

	if client.endpoint != "https://test-endpoint.com" {
		t.Errorf("client has wrong endpoint: got %v want %v", client.endpoint, "https://test-endpoint.com")
	}

	if client.client.Timeout != httpTimeout {
		t.Errorf("client has wrong timeout: got %v want %v", client.client.Timeout, httpTimeout)
	}
}

// This test requires a mock HTTP server to test the actual RPC client
func TestSendRequest(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Test content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Read request body
		var req RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Test request structure
		if req.Jsonrpc != "2.0" {
			t.Errorf("Expected jsonrpc: 2.0, got %s", req.Jsonrpc)
		}

		if req.Method != "testMethod" {
			t.Errorf("Expected method: testMethod, got %s", req.Method)
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","result":42,"id":1}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := newRPCClient(server.URL)

	// Send request
	response, err := client.sendRequest("testMethod", []interface{}{1, "test"})

	// Check for errors
	if err != nil {
		t.Errorf("sendRequest returned error: %v", err)
	}

	// Check response
	if response == nil {
		t.Error("sendRequest returned nil response")
	} else {
		var result int
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
		}

		if result != 42 {
			t.Errorf("Expected result: 42, got %d", result)
		}
	}
}
