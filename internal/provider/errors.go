package provider

import (
	"fmt"

	"github.com/radu103/git-enterprise-hooks/internal/errs"
)

type unsupportedProviderError struct {
	provider string
}

func (u unsupportedProviderError) Error() string {
	return fmt.Sprintf(errs.FmtUnsupportedProviderTypeSimple, u.provider)
}

func ErrUnsupportedProvider(provider string) error {
	return unsupportedProviderError{provider: provider}
}
