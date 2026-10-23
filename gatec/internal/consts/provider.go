package consts

import "strings"

type Provider string

const (
	ProviderNobitex Provider = "NOBITEX"
	ProviderTSE     Provider = "TSE"
)

var providerString = map[string]Provider{
	"NOBITEX": ProviderNobitex,
	"TSE":     ProviderTSE,
}

var providerPersianLabel = map[Provider]string{
	ProviderNobitex: "نوبیتکس",
	ProviderTSE:     "بورس تهران",
}

type ProviderStatus string

const (
	ProviderStatusEnable  ProviderStatus = "ENABLE"
	ProviderStatusDISABLE ProviderStatus = "DISABLE"
)

var providerStatusPersianLabel = map[ProviderStatus]string{
	ProviderStatusEnable:  "فعال",
	ProviderStatusDISABLE: "غیرفعال",
}

func ToProvider(providerStr string) (Provider, bool) {
	providerStr = strings.ToUpper(providerStr)

	provider, ok := providerString[providerStr]
	if !ok {
		return "", false
	}

	return provider, true
}

func (p Provider) PersianLabel() string {
	if label, exists := providerPersianLabel[p]; exists {
		return label
	}
	return ""
}

func (ps ProviderStatus) PersianLabel() string {
	if label, exists := providerStatusPersianLabel[ps]; exists {
		return label
	}
	return ""
}
