package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

type DashenProvider struct {
	client *http.Client
}

func NewDashenProvider() *DashenProvider {
	return &DashenProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *DashenProvider) ProviderKey() string {
	return "dashen"
}

func (p *DashenProvider) DisplayName() string {
	return "Dashen Bank"
}

var (
	dashenAmountRegex   = regexp.MustCompile(`(?i)(?:Transaction\s*Amount|Amount|Total)[:\s]*(?:ETB|Birr)?\s*([0-9,]+(?:\.[0-9]{1,2})?)`)
	dashenSenderRegex   = regexp.MustCompile(`(?i)(?:Sender\s*Name|Account\s*Holder\s*Name)[:\s]*([A-Za-z0-9\s]+?)(?:\n|Sender|Account|<|$)`)
	dashenReceiverRegex = regexp.MustCompile(`(?i)(?:Receiver\s*Name|Beneficiary\s*Name|Institution\s*Name)[:\s]*([A-Za-z0-9\s]+?)(?:\n|Phone|Institution|<|$)`)
	dashenDateRegex     = regexp.MustCompile(`(?i)(?:Transaction\s*Date|Date)[:\s]*([0-9A-Za-z,\-\:\sAPMapm\/]+?)(?:\n|<|$)`)
)

func (p *DashenProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)

	// Primary official receipt portal
	primaryURL := fmt.Sprintf("https://receipts.dashenbanksc.com/receipt/%s", cleanID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, primaryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/html, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to contact Dashen Bank verification service: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)

	// 1. Check for official JSON response (e.g. 400 Bad Request or not found)
	if resp.StatusCode == http.StatusBadRequest || strings.Contains(bodyStr, "Transaction not found") || strings.Contains(bodyStr, "CLNT_TXNALLT") {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Dashen Bank transaction '%s' was not found or is invalid.", cleanID),
		}, nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_ERROR",
			ErrorMessage:  fmt.Sprintf("Dashen Bank returned unexpected HTTP status %d", resp.StatusCode),
		}, nil
	}

	// 2. Try JSON parsing
	var jsonData map[string]any
	if err := json.Unmarshal(bodyBytes, &jsonData); err == nil && len(jsonData) > 0 {
		var amount float64
		for _, key := range []string{"transactionAmount", "amount", "total", "Transaction Amount", "Total"} {
			if val, ok := jsonData[key]; ok {
				cleanVal := strings.ReplaceAll(fmt.Sprintf("%v", val), ",", "")
				if f, err := strconv.ParseFloat(cleanVal, 64); err == nil && f > 0 {
					amount = f
					break
				}
			}
		}

		sender := ""
		for _, key := range []string{"senderName", "sender_name", "accountHolderName", "Account Holder Name", "payer"} {
			if val, ok := jsonData[key]; ok && val != nil {
				sender = strings.TrimSpace(fmt.Sprintf("%v", val))
				if sender != "" && sender != "<nil>" {
					break
				}
			}
		}

		receiver := ""
		for _, key := range []string{"receiverName", "receiver_name", "beneficiary_name", "institutionName", "payee"} {
			if val, ok := jsonData[key]; ok && val != nil {
				receiver = strings.TrimSpace(fmt.Sprintf("%v", val))
				if receiver != "" && receiver != "<nil>" {
					break
				}
			}
		}

		dateStr := ""
		for _, key := range []string{"transactionDate", "date", "timestamp", "Transaction Date"} {
			if val, ok := jsonData[key]; ok && val != nil {
				dateStr = strings.TrimSpace(fmt.Sprintf("%v", val))
				if dateStr != "" && dateStr != "<nil>" {
					break
				}
			}
		}
		if dateStr == "" {
			dateStr = time.Now().Format(time.RFC3339)
		}

		if amount > 0 {
			return &domain.PaymentVerificationResult{
				Verified:           true,
				ProviderKey:        p.ProviderKey(),
				TransactionID:      cleanID,
				Amount:             amount,
				Currency:           "ETB",
				Status:             "COMPLETED",
				PayerIdentifier:    sender,
				ReceiverIdentifier: receiver,
				Timestamp:          dateStr,
				RawDetails:         jsonData,
			}, nil
		}
	}

	// 3. Fallback text/HTML regex parsing
	var parsedAmount float64
	if match := dashenAmountRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
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
			ErrorMessage:  fmt.Sprintf("Dashen Bank transaction '%s' could not be verified or missing amount.", cleanID),
		}, nil
	}

	sender := ""
	if match := dashenSenderRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
		sender = strings.TrimSpace(match[1])
	}

	receiver := ""
	if match := dashenReceiverRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
		receiver = strings.TrimSpace(match[1])
	}

	dateStr := time.Now().Format(time.RFC3339)
	if match := dashenDateRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
		dateStr = strings.TrimSpace(match[1])
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      cleanID,
		Amount:             parsedAmount,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    sender,
		ReceiverIdentifier: receiver,
		Timestamp:          dateStr,
	}, nil
}
