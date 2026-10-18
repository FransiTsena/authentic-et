package provider

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/beecloud/paygate/internal/domain"
)

type PaymentVerifierProvider interface {
	ProviderKey() string
	DisplayName() string
	Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error)
}

var (
	queryParamRegex = regexp.MustCompile(`(?i)(?:trx|id|ref)=([A-Za-z0-9_\-]+)`)
	v2TokenRegex    = regexp.MustCompile(`(v2-[A-Za-z0-9_\-]+)`)
	codeRegex       = regexp.MustCompile(`[A-Za-z0-9_\-]{6,40}`)
)

func CleanTransactionID(input string) string {
	str := strings.TrimSpace(input)
	if str == "" {
		return ""
	}

	lower := strings.ToLower(str)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.Contains(lower, "cbe.com.et") || strings.Contains(lower, "bankofabyssinia.com") || strings.Contains(lower, "ethiotelecom.et") {
		normURL := str
		if !strings.HasPrefix(normURL, "http") {
			normURL = "https://" + str
		}
		if u, err := url.Parse(normURL); err == nil {
			q := u.Query()
			for _, param := range []string{"trx", "id", "ref"} {
				if val := q.Get(param); val != "" {
					clean := strings.TrimSpace(val)
					if strings.HasPrefix(clean, "v2-") {
						return clean
					}
					return strings.ToUpper(clean)
				}
			}
			segments := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(segments) > 0 {
				last := strings.TrimSpace(segments[len(segments)-1])
				if len(last) >= 6 {
					if strings.Contains(lower, "cbe.com.et") || strings.HasPrefix(last, "v2-") {
						return last
					}
					return strings.ToUpper(last)
				}
			}
		}
	}

	if match := queryParamRegex.FindStringSubmatch(str); len(match) > 1 {
		val := strings.TrimSpace(match[1])
		if strings.HasPrefix(val, "v2-") {
			return val
		}
		return strings.ToUpper(val)
	}

	if match := v2TokenRegex.FindStringSubmatch(str); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	if match := codeRegex.FindString(str); match != "" {
		if strings.HasPrefix(match, "v2-") {
			return match
		}
		return strings.ToUpper(match)
	}

	return strings.ToUpper(str)
}

func IsReceiverMatchingMerchant(receiverIdentifier string, merchantNames, merchantAccounts []string) bool {
	clean := strings.ToLower(strings.TrimSpace(receiverIdentifier))
	if clean == "" {
		return false
	}

	// Normalize spaces
	cleanNormalized := strings.Join(strings.Fields(clean), " ")

	// Check direct accounts/numbers
	for _, acc := range merchantAccounts {
		cleanAcc := strings.ToLower(strings.TrimSpace(acc))
		if cleanAcc != "" && strings.Contains(clean, cleanAcc) {
			return true
		}
	}

	// Check compound names (e.g. "fransi tsena" requires BOTH "fransi" AND "tsena")
	for _, merchant := range merchantNames {
		merchantWords := strings.Fields(strings.ToLower(strings.TrimSpace(merchant)))
		if len(merchantWords) == 0 {
			continue
		}

		allMatch := true
		for _, word := range merchantWords {
			if !strings.Contains(cleanNormalized, word) {
				allMatch = false
				break
			}
		}

		if allMatch {
			return true
		}
	}

	return false
}
