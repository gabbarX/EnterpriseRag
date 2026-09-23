package vlm

import (
	"fmt"

	secutils "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/utils"
)

func validateVLMBaseURL(baseURL string) error {
	if baseURL == "" {
		return nil
	}
	if err := secutils.ValidateURLForSSRF(baseURL); err != nil {
		return fmt.Errorf("base URL SSRF check failed: %w", err)
	}
	return nil
}
