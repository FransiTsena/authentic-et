package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type BOAProvider struct {
	client *http.Client
}

func NewBOAProvider() *BOAProvider {
	return &BOAProvider{
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

func (p *BOAProvider) ProviderKey() string {
	return "boa"
}

func (p *BOAProvider) DisplayName() string {
	return "Bank of Abyssinia (BOA)"
}

func (p *BOAProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)
	apiURL := fmt.Sprintf("https://cs.bankofabyssinia.com/api/onlineSlip/getDetails/?id=%s", cleanID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "PROVIDER_UNAVAILABLE",
			ErrorMessage:  fmt.Sprintf("Failed to fetch Bank of Abyssinia receipt: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var resData struct {
		Header struct {
			Status string `json:"status"`
		} `json:"header"`
		Body []map[string]any `json:"body"`
	}

	if err := json.Unmarshal(bodyBytes, &resData); err != nil || resData.Header.Status != "success" || len(resData.Body) == 0 {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("Bank of Abyssinia transaction '%s' not found or invalid.", cleanID),
		}, nil
	}

	slip := resData.Body[0]

	// Check if any slip fields indicate error
	for _, v := range slip {
		strVal := strings.ToLower(fmt.Sprintf("%v", v))
		if strings.Contains(strVal, "invalid reference") || strings.Contains(strVal, "not found") || strings.Contains(strVal, "error") {
			return &domain.PaymentVerificationResult{
				Verified:      false,
				ProviderKey:   p.ProviderKey(),
				TransactionID: cleanID,
				Currency:      "ETB",
				Status:        "FAILED",
				ErrorCode:     "TRANSACTION_NOT_FOUND",
				ErrorMessage:  fmt.Sprintf("Bank of Abyssinia transaction '%s' not found or invalid reference number.", cleanID),
			}, nil
		}
	}

	var parsedAmount float64
	for _, key := range []string{"Transferred Amount", "Total Amount including VAT", "amount"} {
		if raw, ok := slip[key]; ok {
			cleanStr := strings.ReplaceAll(fmt.Sprintf("%v", raw), ",", "")
			if val, err := strconv.ParseFloat(cleanStr, 64); err == nil && val > 0 {
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
			ErrorMessage:  fmt.Sprintf("Bank of Abyssinia transaction '%s' invalid or missing amount.", cleanID),
		}, nil
	}

	payer := ""
	for _, key := range []string{"Payer's Name", "Source Account Name", "Sender's Name"} {
		if v, ok := slip[key]; ok && v != nil && fmt.Sprintf("%v", v) != "<nil>" {
			payer = strings.TrimSpace(fmt.Sprintf("%v", v))
			if payer != "" {
				break
			}
		}
	}

	receiver := ""
	for _, key := range []string{"Receiver's Name", "Beneficiary Name", "Credited Party name"} {
		if v, ok := slip[key]; ok && v != nil && fmt.Sprintf("%v", v) != "<nil>" {
			receiver = strings.TrimSpace(fmt.Sprintf("%v", v))
			if receiver != "" {
				break
			}
		}
	}

	dateStr := fmt.Sprintf("%v", slip["Transaction Date"])
	if dateStr == "<nil>" || dateStr == "" {
		dateStr = time.Now().Format(time.RFC3339)
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
		Timestamp:          dateStr,
		RawDetails:         slip,
	}, nil
}
