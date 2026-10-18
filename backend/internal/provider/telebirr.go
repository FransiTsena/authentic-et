package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

type TelebirrProvider struct {
	client *http.Client
}

func NewTelebirrProvider() *TelebirrProvider {
	return &TelebirrProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *TelebirrProvider) ProviderKey() string {
	return "telebirr"
}

func (p *TelebirrProvider) DisplayName() string {
	return "Telebirr (Ethio Telecom)"
}

var (
	tbAmountRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:የተከፈለው\s+መጠን\/)?Settled\s+Amount[\s\S]*?<td[^>]*>\s*([0-9,]+(?:\.[0-9]{1,2})?)\s*(?:Birr|ETB)?`),
		regexp.MustCompile(`(?i)(?:ጠቅላላ\s+የተከፈለ\/)?Total\s+Paid\s+Amount[\s\S]*?<td[^>]*>\s*([0-9,]+(?:\.[0-9]{1,2})?)\s*(?:Birr|ETB)?`),
		regexp.MustCompile(`(?i)(?:ETB|Amount|Birr)[:\s]*([0-9,]+(?:\.[0-9]{1,2})?)`),
	}

	tbPayerRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:የከፋይ\s+ስም\/)?Payer\s+Name[\s\S]*?<td[^>]*>\s*([A-Za-z0-9\s]+?)\s*<\/`),
		regexp.MustCompile(`(?i)(?:Payer\s+telebirr\s+no\.?|Customer\s+Name|የከፋይ\s+ስም)[\s\S]*?<td[^>]*>\s*([A-Za-z0-9\s]+?)\s*<\/`),
	}

	tbReceiverRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:የገንዘብ\s+ተቀባይ\s+ስም\/)?Credited\s+Party\s+name[\s\S]*?<td[^>]*>\s*([A-Za-z0-9\s]+?)\s*<\/`),
		regexp.MustCompile(`(?i)(?:Receiver\s+Name|የገንዘብ\s+ተቀባይ\s+ስም)[\s\S]*?<td[^>]*>\s*([A-Za-z0-9\s]+?)\s*<\/`),
	}

	tbDateRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(\d{2}-\d{2}-\d{4}\s+\d{2}:\d{2}:\d{2})`),
		regexp.MustCompile(`(?i)(?:Payment\s+Date|Date\s*&\s*Time|Transaction\s+Date|Date|Time)[\s\S]*?<td[^>]*>\s*([0-9\-\:\sAPMapm\/]+?)\s*<\/`),
	}
)

func (p *TelebirrProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)
	receiptURL := fmt.Sprintf("https://transactioninfo.ethiotelecom.et/receipt/%s", cleanID)

	skipPrimary := os.Getenv("SKIP_PRIMARY_VERIFICATION") == "true"
	var html string
	var fetchErr error

	if !skipPrimary {
		html, fetchErr = p.fetchHTML(ctx, receiptURL)
	}

	// If primary failed or timed out (due to Ethiopian regional IP geoblock), check for fallback proxy
	if (skipPrimary || fetchErr != nil || len(html) < 20) {
		proxyURL := os.Getenv("TELEBIRR_PROXY_URL")
		if proxyURL == "" {
			proxyURL = os.Getenv("FALLBACK_PROXIES")
		}
		if proxyURL != "" {
			targetURL := proxyURL
			if !strings.Contains(targetURL, "%s") && !strings.HasSuffix(targetURL, "=") {
				targetURL = targetURL + cleanID
			} else if strings.Contains(targetURL, "%s") {
				targetURL = fmt.Sprintf(targetURL, cleanID)
			} else {
				targetURL = targetURL + cleanID
			}

			proxyHTML, pErr := p.fetchHTML(ctx, targetURL)
			if pErr == nil && len(proxyHTML) >= 20 {
				html = proxyHTML
				fetchErr = nil
			}
		}
	}

	if fetchErr != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to fetch Telebirr receipt: %v", fetchErr),
		}, nil
	}

	// Check for JSON response (from proxy or API)
	if strings.HasPrefix(strings.TrimSpace(html), "{") {
		var jsonResp struct {
			Success bool `json:"success"`
			Data    struct {
				PayerName              string `json:"payerName"`
				PayerTelebirrNo        string `json:"payerTelebirrNo"`
				CreditedPartyName      string `json:"creditedPartyName"`
				CreditedPartyAccountNo string `json:"creditedPartyAccountNo"`
				TransactionStatus      string `json:"transactionStatus"`
				ReceiptNo              string `json:"receiptNo"`
				PaymentDate            string `json:"paymentDate"`
				SettledAmount          string `json:"settledAmount"`
				ServiceFee             string `json:"serviceFee"`
				TotalPaidAmount        string `json:"totalPaidAmount"`
			} `json:"data"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(html), &jsonResp); err == nil && jsonResp.Data.ReceiptNo != "" {
			amountStr := jsonResp.Data.SettledAmount
			if amountStr == "" {
				amountStr = jsonResp.Data.TotalPaidAmount
			}
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			amountStr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(amountStr), "Birr"))
			amountVal, _ := strconv.ParseFloat(amountStr, 64)

			if amountVal > 0 {
				return &domain.PaymentVerificationResult{
					Verified:           true,
					ProviderKey:        p.ProviderKey(),
					TransactionID:      cleanID,
					Amount:             amountVal,
					Currency:           "ETB",
					Status:             "COMPLETED",
					PayerIdentifier:    jsonResp.Data.PayerName,
					ReceiverIdentifier: jsonResp.Data.CreditedPartyName,
					Timestamp:          jsonResp.Data.PaymentDate,
				}, nil
			}
		}
	}

	if len(html) < 20 || strings.Contains(html, "404") || strings.Contains(html, "Not Found") || strings.Contains(html, "Invalid") {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Telebirr transaction %s not found or invalid receipt page.", cleanID),
		}, nil
	}

	var parsedAmount float64
	for _, regex := range tbAmountRegexes {
		if match := regex.FindStringSubmatch(html); len(match) > 1 {
			cleaned := strings.ReplaceAll(match[1], ",", "")
			if val, err := strconv.ParseFloat(cleaned, 64); err == nil && val > 0 {
				parsedAmount = val
				break
			}
		}
	}

	if parsedAmount <= 0 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Telebirr transaction %s invalid or missing amount.", cleanID),
		}, nil
	}

	payer := ""
	for _, regex := range tbPayerRegexes {
		if match := regex.FindStringSubmatch(html); len(match) > 1 {
			payer = strings.TrimSpace(match[1])
			if payer != "" {
				break
			}
		}
	}

	receiver := ""
	for _, regex := range tbReceiverRegexes {
		if match := regex.FindStringSubmatch(html); len(match) > 1 {
			receiver = strings.TrimSpace(match[1])
			if receiver != "" {
				break
			}
		}
	}

	timestamp := time.Now().Format(time.RFC3339)
	for _, regex := range tbDateRegexes {
		if match := regex.FindStringSubmatch(html); len(match) > 1 {
			timestamp = strings.TrimSpace(match[1])
			if timestamp != "" {
				break
			}
		}
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      cleanID,
		Amount:             parsedAmount,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    payer,
		ReceiverIdentifier: receiver,
		Timestamp:          timestamp,
	}, nil
}

func (p *TelebirrProvider) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json,*/*;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}
