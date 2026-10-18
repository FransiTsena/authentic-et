package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type CBEProvider struct {
	client *http.Client
}

func NewCBEProvider() *CBEProvider {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // CBE apps portal uses internal cert
	}
	return &CBEProvider{
		client: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}
}

func (p *CBEProvider) ProviderKey() string {
	return "cbe"
}

func (p *CBEProvider) DisplayName() string {
	return "Commercial Bank of Ethiopia (CBE)"
}

func (p *CBEProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)

	// 1. If v2 token, verify via official Mobile Banking v2 API
	if strings.HasPrefix(cleanID, "v2-") || strings.Contains(cleanID, "v2-") {
		res, err := p.verifyViaV2API(ctx, cleanID)
		if err == nil && res.Verified {
			return res, nil
		}
	}

	// 2. Fallback to CBE receipt endpoint
	return p.verifyViaReceiptEndpoint(ctx, cleanID, transactionID)
}

func (p *CBEProvider) verifyViaV2API(ctx context.Context, token string) (*domain.PaymentVerificationResult, error) {
	apiURL := fmt.Sprintf("https://Mb.cbe.com.et/api/v1/transactions/public/transaction-detail/%s", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-App-ID", "d1292e42-7400-49de-a2d3-9731caa4c819")
	req.Header.Set("X-App-Version", "0a01980b-9859-1369-8198-59f403820000")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: token,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to reach CBE v2 API: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode >= 500 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: token,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("CBE transaction '%s' not found or invalid receipt token.", token),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	resolvedRef := token
	if idVal, ok := data["id"].(string); ok && idVal != "" {
		resolvedRef = strings.ToUpper(strings.TrimSpace(idVal))
	}

	var amount float64
	for _, key := range []string{"amountCredited", "amountDebited", "debitAmount"} {
		if val, exists := data[key]; exists {
			switch v := val.(type) {
			case float64:
				amount = v
			case string:
				if parsed, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64); err == nil {
					amount = parsed
				}
			}
			if amount > 0 {
				break
			}
		}
	}

	if amount <= 0 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: resolvedRef,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("CBE transaction %s missing valid credited amount.", resolvedRef),
		}, nil
	}

	payer, _ := data["debitAccountHolder"].(string)
	receiver, _ := data["creditAccountHolder"].(string)
	currency, _ := data["creditCurrency"].(string)
	if currency == "" {
		currency = "ETB"
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      resolvedRef,
		Amount:             amount,
		Currency:           currency,
		Status:             "COMPLETED",
		PayerIdentifier:    strings.TrimSpace(payer),
		ReceiverIdentifier: strings.TrimSpace(receiver),
		Timestamp:          time.Now().Format(time.RFC3339),
		RawDetails:         data,
	}, nil
}

var (
	cbePayerRegex    = regexp.MustCompile(`(?i)(?:Payer|Debit Account Holder|Sender)\s*[:\s]+([^\n\r<]+)`)
	cbeReceiverRegex = regexp.MustCompile(`(?i)(?:Receiver|Credit Account Holder|Payee)\s*[:\s]+([^\n\r<]+)`)
	cbeAmountRegex   = regexp.MustCompile(`(?i)(?:Transferred Amount|Amount Credited|Amount|Total amount debited)[:\s]+([\d,]+\.?\d*)`)
	cbeDateRegex     = regexp.MustCompile(`(?i)(?:Payment Date\s*&\s*Time|Date\s*&\s*Time|Date)[:\s]*([^\n\r<]+)`)
	cbeRefRegex      = regexp.MustCompile(`(?i)\b(FT[A-Z0-9]{8,})\b`)
)

func (p *CBEProvider) verifyViaReceiptEndpoint(ctx context.Context, cleanID, rawInput string) (*domain.PaymentVerificationResult, error) {
	queryID := cleanID
	cbeAccountSuffix := "646339972"
	if strings.HasPrefix(cleanID, "FT") && !strings.HasSuffix(cleanID, cbeAccountSuffix) && len(cleanID) <= 14 {
		queryID = cleanID + cbeAccountSuffix
	}

	// Strictly pin to official CBE portal endpoint with alphanumeric query ID to prevent SSRF
	sanitizedQueryID := regexp.MustCompile(`[^A-Za-z0-9_\-]`).ReplaceAllString(queryID, "")
	if sanitizedQueryID == "" {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "INVALID_TRANSACTION_ID",
			ErrorMessage:  "Invalid CBE transaction ID format",
		}, nil
	}

	receiptURL := fmt.Sprintf("https://apps.cbe.com.et:100/?id=%s", sanitizedQueryID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, receiptURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,*/*")

	resp, err := p.client.Do(req)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to fetch CBE receipt: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(bodyBytes) < 50 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("CBE transaction %s not found or invalid receipt response.", cleanID),
		}, nil
	}

	bodyText := string(bodyBytes)

	// Check if page contains embedded v2 token
	if v2Match := v2TokenRegex.FindString(bodyText); v2Match != "" && v2Match != cleanID {
		cascaded, err := p.verifyViaV2API(ctx, v2Match)
		if err == nil && cascaded.Verified {
			return cascaded, nil
		}
	}

	lowerText := strings.ToLower(bodyText)
	if strings.Contains(lowerText, "not found") || strings.Contains(lowerText, "invalid reference") {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("CBE transaction %s not found or invalid reference.", cleanID),
		}, nil
	}

	var parsedAmount float64
	if match := cbeAmountRegex.FindStringSubmatch(bodyText); len(match) > 1 {
		cleaned := strings.ReplaceAll(match[1], ",", "")
		parsedAmount, _ = strconv.ParseFloat(cleaned, 64)
	}

	if parsedAmount <= 0 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("CBE transaction %s invalid or missing amount.", cleanID),
		}, nil
	}

	payer := ""
	if match := cbePayerRegex.FindStringSubmatch(bodyText); len(match) > 1 {
		payer = strings.TrimSpace(match[1])
	}

	receiver := ""
	if match := cbeReceiverRegex.FindStringSubmatch(bodyText); len(match) > 1 {
		receiver = strings.TrimSpace(match[1])
	}

	resolvedRef := cleanID
	if match := cbeRefRegex.FindStringSubmatch(bodyText); len(match) > 1 {
		resolvedRef = strings.TrimSpace(match[1])
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      resolvedRef,
		Amount:             parsedAmount,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    payer,
		ReceiverIdentifier: receiver,
		Timestamp:          time.Now().Format(time.RFC3339),
	}, nil
}
