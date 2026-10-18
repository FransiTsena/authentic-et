package provider_test

import (
	"testing"

	"github.com/beecloud/paygate/internal/provider"
	"github.com/stretchr/testify/assert"
)

func TestCleanTransactionID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean alphanumeric",
			input:    "FT262036S1GH10104",
			expected: "FT262036S1GH10104",
		},
		{
			name:     "CBE v2 receipt URL",
			input:    "https://mbreciept.cbe.com.et/v2-hfHCxGvfZKPFdpKDDDwH",
			expected: "v2-hfHCxGvfZKPFdpKDDDwH",
		},
		{
			name:     "BOA query parameter URL",
			input:    "https://cs.bankofabyssinia.com/slip/?trx=FT262036S1GH10104",
			expected: "FT262036S1GH10104",
		},
		{
			name:     "plain query string",
			input:    "trx=FT9988776655",
			expected: "FT9988776655",
		},
		{
			name:     "spaces and lowercase",
			input:    "  ft12345678  ",
			expected: "FT12345678",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := provider.CleanTransactionID(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsReceiverMatchingMerchant(t *testing.T) {
	merchantNames := []string{"fransi tsena", "beecloud"}
	merchantAccounts := []string{"1000646339972", "929945151"}

	// Valid matches
	assert.True(t, provider.IsReceiverMatchingMerchant("FRANSI TSENA", merchantNames, merchantAccounts))
	assert.True(t, provider.IsReceiverMatchingMerchant("Fransi Tsena G.", merchantNames, merchantAccounts))
	assert.True(t, provider.IsReceiverMatchingMerchant("BeeCloud Technologies", merchantNames, merchantAccounts))
	assert.True(t, provider.IsReceiverMatchingMerchant("1000646339972", merchantNames, merchantAccounts))

	// Invalid matches: only first name "Fransi" but missing "Tsena"
	assert.False(t, provider.IsReceiverMatchingMerchant("Fransi Kebede", merchantNames, merchantAccounts))
	// Missing first name
	assert.False(t, provider.IsReceiverMatchingMerchant("Abebe Tsena", merchantNames, merchantAccounts))
	// Unrelated
	assert.False(t, provider.IsReceiverMatchingMerchant("John Doe", merchantNames, merchantAccounts))
	assert.False(t, provider.IsReceiverMatchingMerchant("", merchantNames, merchantAccounts))
}

func TestRegistry_SupportedProviders(t *testing.T) {
	reg := provider.NewRegistry()
	list := reg.List()

	expectedKeys := map[string]bool{
		"cbe":      false,
		"telebirr": false,
		"boa":      false,
		"dashen":   false,
		"awash":    false,
		"mpesa":    false,
		"mock":     false,
	}

	for _, p := range list {
		if _, ok := expectedKeys[p.Key]; ok {
			expectedKeys[p.Key] = true
		}
	}

	for k, found := range expectedKeys {
		assert.Truef(t, found, "Expected provider %s to be registered in Registry", k)
	}

	// Ensure each can be retrieved
	for k := range expectedKeys {
		p, err := reg.Get(k)
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, k, p.ProviderKey())
	}
}

func TestDashenProvider_NotFound(t *testing.T) {
	dp := provider.NewDashenProvider()
	ctx := t.Context()
	res, err := dp.Verify(ctx, "NONEXISTENT_DASHEN_99999999")
	assert.NoError(t, err)
	assert.False(t, res.Verified)
	assert.Contains(t, []string{"TRANSACTION_NOT_FOUND", "PROVIDER_UNAVAILABLE", "PROVIDER_ERROR"}, res.ErrorCode)
}

func TestAwashProvider_NotFound(t *testing.T) {
	ap := provider.NewAwashProvider()
	ctx := t.Context()
	res, err := ap.Verify(ctx, "NONEXISTENT_AWASH_99999999")
	assert.NoError(t, err)
	assert.False(t, res.Verified)
	assert.Contains(t, []string{"TRANSACTION_NOT_FOUND", "PROVIDER_UNAVAILABLE", "PROVIDER_ERROR"}, res.ErrorCode)
}

func TestMpesaProvider_NotFound(t *testing.T) {
	mp := provider.NewMpesaProvider()
	ctx := t.Context()
	res, err := mp.Verify(ctx, "NONEXISTENT_MPESA_99999999")
	assert.NoError(t, err)
	assert.False(t, res.Verified)
	assert.Contains(t, []string{"TRANSACTION_NOT_FOUND", "PROVIDER_UNAVAILABLE", "PROVIDER_ERROR"}, res.ErrorCode)
}
