package provider

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
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

type MpesaProvider struct {
	client *http.Client
}

func NewMpesaProvider() *MpesaProvider {
	return &MpesaProvider{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (p *MpesaProvider) ProviderKey() string {
	return "mpesa"
}

func (p *MpesaProvider) DisplayName() string {
	return "M-Pesa (Safaricom Ethiopia)"
}

type mpesaAPIResponse struct {
	ResponseCode        string `json:"responseCode"`
	ResponseDescription string `json:"responseDescription"`
	Base64Data          string `json:"base64Data,omitempty"`
}

var (
	mpesaAmountRegex   = regexp.MustCompile(`(?i)(?:TOTAL|Amount|Total Paid)[:\s]*([0-9,]+(?:\.[0-9]{1,2})?)`)
	mpesaPayerRegex    = regexp.MustCompile(`(?i)PAYER\s*NAME\s+([A-Za-z0-9\s]+?)(?:PAYER\s*PHONE|\+251|00\d+|Addis|$)`)
	mpesaPayerPhRegex  = regexp.MustCompile(`(?i)PAYER\s*PHONE\s*(?:NUMBER)?\s*([0-9\+]+)`)
	mpesaReceiverRegex = regexp.MustCompile(`(?i)RECEIVER\s*NAME\s+([A-Za-z0-9\s]+?)(?:RECEIVER\s*NUMBER|\/|$)`)
	mpesaDateRegex     = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})`)
	mpesaTxIDRegex     = regexp.MustCompile(`(?i)TRANSACTION\s*ID\s+([A-Z0-9]+)`)
)

func (p *MpesaProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)
	primaryURL := fmt.Sprintf("https://m-pesabusiness.safaricom.et/api/receipt/getReceipt?trxNo=%s", cleanID)

	var apiResp mpesaAPIResponse
	var fetchErr error

	skipPrimary := os.Getenv("SKIP_PRIMARY_VERIFICATION") == "true"
	if !skipPrimary {
		fetchErr = p.fetchJSON(ctx, primaryURL, &apiResp)
	}

	// Try fallback proxy if primary failed or skipped
	if skipPrimary || fetchErr != nil || (apiResp.ResponseCode != "0" && apiResp.Base64Data == "") {
		proxyURL := os.Getenv("MPESA_PROXY_URL")
		if proxyURL == "" {
			proxyKey := os.Getenv("MPESA_PROXY_KEY")
			proxyBase := os.Getenv("MPESA_FALLBACK_PROXY")
			if proxyBase != "" {
				proxyURL = fmt.Sprintf("%s?reference=%s&key=%s", proxyBase, cleanID, proxyKey)
			}
		} else {
			if strings.Contains(proxyURL, "%s") {
				proxyURL = fmt.Sprintf(proxyURL, cleanID)
			} else if strings.HasSuffix(proxyURL, "=") {
				proxyURL = proxyURL + cleanID
			}
		}

		if proxyURL != "" {
			var proxyResp mpesaAPIResponse
			pErr := p.fetchJSON(ctx, proxyURL, &proxyResp)
			if pErr == nil && (proxyResp.ResponseCode == "0" || proxyResp.Base64Data != "") {
				apiResp = proxyResp
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
			ErrorMessage:  fmt.Sprintf("Failed to contact Safaricom M-Pesa verification service: %v", fetchErr),
		}, nil
	}

	// If transaction does not exist or API returned an error code
	if apiResp.ResponseCode != "0" || apiResp.Base64Data == "" {
		desc := apiResp.ResponseDescription
		if desc == "" {
			desc = "The transaction receipt was not found or is invalid."
		}
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "TRANSACTION_NOT_FOUND",
			ErrorMessage:  fmt.Sprintf("M-Pesa transaction '%s': %s", cleanID, desc),
		}, nil
	}

	// Decode base64 PDF receipt
	pdfBytes, err := base64.StdEncoding.DecodeString(apiResp.Base64Data)
	if err != nil {
		return &domain.PaymentVerificationResult{
			Verified:      false,
			ProviderKey:   p.ProviderKey(),
			TransactionID: cleanID,
			Currency:      "ETB",
			Status:        "FAILED",
			ErrorCode:     "RECEIPT_DECODE_ERROR",
			ErrorMessage:  fmt.Sprintf("Failed to decode M-Pesa receipt PDF: %v", err),
		}, nil
	}

	extractedText := extractTextFromPDF(pdfBytes)

	var parsedAmount float64
	if match := mpesaAmountRegex.FindStringSubmatch(extractedText); len(match) > 1 {
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
			ErrorMessage:  fmt.Sprintf("M-Pesa transaction '%s' could not extract amount from receipt.", cleanID),
		}, nil
	}

	payer := ""
	if match := mpesaPayerRegex.FindStringSubmatch(extractedText); len(match) > 1 {
		payer = strings.TrimSpace(match[1])
	}
	if payerPhone := mpesaPayerPhRegex.FindStringSubmatch(extractedText); len(payerPhone) > 1 {
		if payer != "" {
			payer = fmt.Sprintf("%s (%s)", payer, strings.TrimSpace(payerPhone[1]))
		} else {
			payer = strings.TrimSpace(payerPhone[1])
		}
	}

	receiver := ""
	if match := mpesaReceiverRegex.FindStringSubmatch(extractedText); len(match) > 1 {
		receiver = strings.TrimSpace(match[1])
	}

	dateStr := time.Now().Format(time.RFC3339)
	if match := mpesaDateRegex.FindStringSubmatch(extractedText); len(match) > 1 {
		dateStr = strings.TrimSpace(match[1])
	}

	actualTxID := cleanID
	if match := mpesaTxIDRegex.FindStringSubmatch(extractedText); len(match) > 1 {
		actualTxID = strings.TrimSpace(match[1])
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        p.ProviderKey(),
		TransactionID:      actualTxID,
		Amount:             parsedAmount,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    payer,
		ReceiverIdentifier: receiver,
		Timestamp:          dateStr,
	}, nil
}

func (p *MpesaProvider) fetchJSON(ctx context.Context, url string, target *mpesaAPIResponse) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://m-pesabusiness.safaricom.et/")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(bodyBytes, target)
}

// extractTextFromPDF scans a PDF buffer, uncompresses Flate/zlib streams, and extracts text tokens.
func extractTextFromPDF(pdfBytes []byte) string {
	var sb strings.Builder

	streamRe := regexp.MustCompile(`(?s)stream\r?\n(.*?)[\r\n]+endstream`)
	matches := streamRe.FindAllSubmatch(pdfBytes, -1)

	for _, m := range matches {
		rawStream := m[1]
		var textBytes []byte

		// Try zlib decompression
		zr, err := zlib.NewReader(bytes.NewReader(rawStream))
		if err == nil {
			decompressed, readErr := io.ReadAll(zr)
			zr.Close()
			if readErr == nil && len(decompressed) > 0 {
				textBytes = decompressed
			}
		}

		if len(textBytes) == 0 {
			textBytes = rawStream
		}

		// Extract string literals enclosed in parentheses: (Sample text)
		parenRe := regexp.MustCompile(`\(([^)]+)\)`)
		pMatches := parenRe.FindAllSubmatch(textBytes, -1)
		for _, pm := range pMatches {
			sb.Write(pm[1])
			sb.WriteString(" ")
		}
		sb.WriteString("\n")
	}

	// Also extract any readable plain ASCII sequences outside streams
	if sb.Len() == 0 {
		cleanAscii := regexp.MustCompile(`[A-Za-z0-9\s:,\-\.\/]{4,}`)
		for _, match := range cleanAscii.FindAll(pdfBytes, -1) {
			sb.Write(match)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
