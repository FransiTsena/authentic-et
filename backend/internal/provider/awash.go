package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

type AwashProvider struct {
	client *http.Client
}

func NewAwashProvider() *AwashProvider {
	return &AwashProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *AwashProvider) ProviderKey() string {
	return "awash"
}

func (p *AwashProvider) DisplayName() string {
	return "Awash Bank (Awash Birr)"
}

var (
	awashAmountRegex    = regexp.MustCompile(`(?i)Amount[\s\S]*?<td>:\s*<\/td>[\s\S]*?<td>\s*<tt><span>\s*([0-9,]+(?:\.[0-9]{1,2})?)\s*(?:ETB|Birr)?`)
	awashTimeRegex      = regexp.MustCompile(`(?i)Transaction\s*Time[\s\S]*?<td>:\s*<\/td>[\s\S]*?<td>\s*<tt><span>\s*([0-9\-\:\sAPMapm]+?)\s*<\/span>`)
	awashSenderRegex    = regexp.MustCompile(`(?i)Sender\s*Name[\s\S]*?<td>:\s*<\/td>[\s\S]*?<td>\s*<tt><span>\s*([A-Za-z0-9\s]+?)\s*<\/span>`)
	awashReceiverRegex  = regexp.MustCompile(`(?i)Beneficiary\s*name[\s\S]*?<td>:\s*<\/td>[\s\S]*?<td>\s*<tt><span>\s*([A-Za-z0-9\s]+?)\s*<\/span>`)
	awashTxIDRegex      = regexp.MustCompile(`(?i)Transaction\s*ID[\s\S]*?<td>:\s*<\/td>[\s\S]*?<td>\s*<tt><span>\s*([A-Za-z0-9\s]+?)\s*<\/span>`)
	awashAltAmountRegex = regexp.MustCompile(`(?i)(?:Amount|Total)[:\s]*([0-9,]+(?:\.[0-9]{1,2})?)\s*(?:ETB|Birr)`)
)

func (p *AwashProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)
	cleanID = strings.TrimPrefix(cleanID, "-")

	// Awash receipts URLs start with a dash before the ID (e.g. /-E41AE0D86FFA-21XYYW or /-E41AE0D86FFA)
	url := fmt.Sprintf("https://awashpay.awashbank.com:8225/-%s", cleanID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to contact Awash Bank receipt portal: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(bodyBytes)

	// Check if Awash returned "Receipt not found"
	if strings.Contains(html, "Receipt not found") || strings.Contains(html, "not found") || len(html) < 50 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Awash Bank transaction '%s' was not found or has expired.", cleanID),
		}, nil
	}

	var parsedAmount float64
	if match := awashAmountRegex.FindStringSubmatch(html); len(match) > 1 {
		clean := strings.ReplaceAll(match[1], ",", "")
		parsedAmount, _ = strconv.ParseFloat(clean, 64)
	} else if match := awashAltAmountRegex.FindStringSubmatch(html); len(match) > 1 {
		clean := strings.ReplaceAll(match[1], ",", "")
		parsedAmount, _ = strconv.ParseFloat(clean, 64)
	}

	if parsedAmount <= 0 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Awash Bank transaction '%s' has missing or unparseable amount.", cleanID),
		}, nil
	}

	sender := ""
	if match := awashSenderRegex.FindStringSubmatch(html); len(match) > 1 {
		sender = strings.TrimSpace(match[1])
	}

	receiver := ""
	if match := awashReceiverRegex.FindStringSubmatch(html); len(match) > 1 {
		receiver = strings.TrimSpace(match[1])
	}

	dateStr := time.Now().Format(time.RFC3339)
	if match := awashTimeRegex.FindStringSubmatch(html); len(match) > 1 {
		dateStr = strings.TrimSpace(match[1])
	}

	actualTxID := cleanID
	if match := awashTxIDRegex.FindStringSubmatch(html); len(match) > 1 {
		actualTxID = strings.TrimSpace(match[1])
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      actualTxID,
		Amount:             parsedAmount,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    sender,
		ReceiverIdentifier: receiver,
		Timestamp:          dateStr,
	}, nil
}
