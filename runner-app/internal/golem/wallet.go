package golem

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// WalletInfo represents wallet status and balance information
type WalletInfo struct {
	Address    string  `json:"address"`
	Network    string  `json:"network"`
	Balance    float64 `json:"balance"`
	Currency   string  `json:"currency"`
	IsUnlocked bool    `json:"is_unlocked"`
}

// PaymentPlatform represents a payment platform configuration
type PaymentPlatform struct {
	Platform string `json:"platform"`
	Driver   string `json:"driver"`
	Network  string `json:"network"`
	Token    string `json:"token"`
	Address  string `json:"address"`
}

// extractBalance tries to extract balance from various possible fields
func extractBalance(account map[string]interface{}) float64 {
	// Try different balance field names
	balanceFields := []string{"amount", "balance", "total_amount", "confirmed", "available"}
	
	for _, field := range balanceFields {
		if val, ok := account[field]; ok {
			// Try as float64 first
			if balance, ok := val.(float64); ok && balance > 0 {
				return balance
			}
			// Try as string (common in CLI JSON output)
			if balanceStr, ok := val.(string); ok {
				if balance, err := parseFloat(balanceStr); err == nil && balance > 0 {
					return balance
				}
			}
			// Try as int/int64
			if balanceInt, ok := val.(int); ok && balanceInt > 0 {
				return float64(balanceInt)
			}
			if balanceInt64, ok := val.(int64); ok && balanceInt64 > 0 {
				return float64(balanceInt64)
			}
		}
	}
	
	return 0.0
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	// Remove common currency suffixes
	s = strings.TrimSuffix(s, " tGLM")
	s = strings.TrimSuffix(s, " GLM")
	s = strings.TrimSpace(s)
	
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return 0.0, fmt.Errorf("invalid float: %s", s)
}

// InitializeWallet ensures wallet is properly configured for payments
func (s *Service) InitializeWallet(ctx context.Context) error {
	if s.backend != "sdk" {
		return nil // Skip wallet setup for mock backend
	}

	// Use CLI directly since REST API balance endpoint isn't working
	wallet, err := s.getWalletInfoDirectly(ctx)
	if err != nil {
		return fmt.Errorf("failed to get wallet info: %w", err)
	}

	if wallet.Address == "" {
		return fmt.Errorf("no wallet address configured")
	}

	// Minimum balance requirement for Golem execution
	minBalance := 0.01
	if wallet.Balance < minBalance {
		return fmt.Errorf("insufficient wallet balance: %f %s (minimum: %f)", 
			wallet.Balance, wallet.Currency, minBalance)
	}

	// Store wallet info for later use
	s.walletInfo = wallet

	return nil
}

// getWalletInfoDirectly gets wallet info directly from CLI, simplified version
func (s *Service) getWalletInfoDirectly(ctx context.Context) (*WalletInfo, error) {
	// Execute yagna payment status command with full path
	cmd := exec.CommandContext(ctx, "/Users/Jammie/.local/bin/yagna", "payment", "status", "--network", "holesky", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("CLI command failed: %w", err)
	}

	// Parse JSON output
	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	// Extract balance and other info
	balance := extractBalance(statusData)
	network := getString(statusData, "network")
	token := getString(statusData, "token")

	if balance <= 0 {
		return nil, fmt.Errorf("no balance found in CLI output")
	}

	return &WalletInfo{
		Address:    "0xc5420dcb1d0be3813505eee407fb50e627a329f4", // Known address
		Network:    network,
		Balance:    balance,
		Currency:   token,
		IsUnlocked: true,
	}, nil
}


//lint:ignore U1000 kept for future CLI fallback parsing when JSON is unavailable
// parseTextPaymentStatus parses the text output of yagna payment status
func (s *Service) parseTextPaymentStatus(ctx context.Context) *WalletInfo {
	cmd := exec.CommandContext(ctx, "/Users/Jammie/.local/bin/yagna", "payment", "status", "--network", "holesky")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	var address string
	var balance float64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Extract address from "Status for account: 0x..."
		if strings.HasPrefix(line, "Status for account:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				address = strings.TrimSpace(parts[1])
			}
		}
		
		// Extract balance from table row containing "total amount"
		if strings.Contains(line, "tGLM") && strings.Contains(line, "driver: erc20") {
			// Look for the pattern like "1000.0000 tGLM"
			fields := strings.Fields(line)
			for i, field := range fields {
				if strings.HasSuffix(field, "tGLM") && i > 0 {
					balanceStr := strings.TrimSuffix(field, "tGLM")
					if bal, err := strconv.ParseFloat(balanceStr, 64); err == nil {
						balance = bal
						break
					}
				}
			}
		}
	}

	if address != "" && balance > 0 {
		return &WalletInfo{
			Address:    address,
			Network:    "holesky",
			Balance:    balance,
			Currency:   "tGLM",
			IsUnlocked: true,
		}
	}

	return nil
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// EnsureWalletReady checks wallet status and provides helpful error messages
func (s *Service) EnsureWalletReady(ctx context.Context) error {
	if s.backend != "sdk" {
		return nil // Skip for mock backend
	}

	if s.walletInfo == nil {
		if err := s.InitializeWallet(ctx); err != nil {
			return fmt.Errorf("wallet initialization failed: %w", err)
		}
	}

	// Use the already-initialized wallet info (which includes CLI fallback if needed)
	if s.walletInfo.Balance < 0.01 {
		return fmt.Errorf("insufficient funds for task execution: %f %s", s.walletInfo.Balance, s.walletInfo.Currency)
	}

	return nil
}
