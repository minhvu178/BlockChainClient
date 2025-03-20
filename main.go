package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Configuration
const (
	solanaRPC      = "https://api.mainnet-beta.solana.com"
	httpServerAddr = ":8080"
	httpTimeout    = 10 * time.Second
)

// SolanaRPCClient defines the interface for Solana RPC operations
type SolanaRPCClient interface {
	getLatestSlot() (uint64, error)
	getBlockDetails(slot uint64) (json.RawMessage, error)
}

// JSON-RPC request struct
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
	ID      int           `json:"id"`
}

// JSON-RPC response struct
type RPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError represents an error returned from the RPC server
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcClient is a client for making RPC requests
type rpcClient struct {
	endpoint string
	client   *http.Client
}

// newRPCClient creates a new RPC client
func newRPCClient(endpoint string) *rpcClient {
	return &rpcClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// sendRequest sends an RPC request to Solana
func (c *rpcClient) sendRequest(method string, params []interface{}) (*RPCResponse, error) {
	reqBody := RPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response RPCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("RPC error: %d - %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

// getLatestSlot gets the latest block (slot number)
func (c *rpcClient) getLatestSlot() (uint64, error) {
	response, err := c.sendRequest("getSlot", nil)
	if err != nil {
		return 0, err
	}

	var slot uint64
	if err := json.Unmarshal(response.Result, &slot); err != nil {
		return 0, fmt.Errorf("failed to parse slot number: %w", err)
	}

	return slot, nil
}

// getBlockDetails gets details of a specific block
func (c *rpcClient) getBlockDetails(slot uint64) (json.RawMessage, error) {
	response, err := c.sendRequest("getBlock", []interface{}{slot})
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

// API handlers
func handleGetLatestSlot(client SolanaRPCClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slot, err := client.getLatestSlot()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// json.NewEncoder(w).Encode(map[string]uint64{"latest_block": slot})
		jsonData, _ := json.Marshal(map[string]uint64{"latest_block": slot})
		w.Write(jsonData)
	}
}

func handleGetBlockDetails(client SolanaRPCClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slotStr := r.URL.Query().Get("block")
		if slotStr == "" {
			http.Error(w, "block parameter is required", http.StatusBadRequest)
			return
		}

		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid block number", http.StatusBadRequest)
			return
		}

		blockDetails, err := client.getBlockDetails(slot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(blockDetails)
	}
}

func main() {
	client := newRPCClient(solanaRPC)

	// Setup HTTP API routes
	mux := http.NewServeMux()
	mux.HandleFunc("/latest-block", handleGetLatestSlot(client))
	mux.HandleFunc("/block-details", handleGetBlockDetails(client))

	// Start server
	log.Printf("Starting Solana Blockchain Client API server on %s...", httpServerAddr)
	log.Fatal(http.ListenAndServe(httpServerAddr, mux))
}
