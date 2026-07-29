package provider

import "fmt"

type unsupportedProviderError struct {
	provider string
}

func (u unsupportedProviderError) Error() string {
	return fmt.Sprintf("unsupported provider type: %s", u.provider)
}

func ErrUnsupportedProvider(provider string) error {
	return unsupportedProviderError{provider: provider}
}
