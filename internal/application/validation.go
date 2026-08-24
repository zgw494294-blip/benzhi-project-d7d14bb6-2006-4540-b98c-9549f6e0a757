package application

import (
	"strings"
	"timber-release-gate/internal/domain"
)

func ValidateActor(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return domain.Invalid("INVALID_ACTOR", "操作者不能为空")
	}
	return nil
}
func ValidateExpected(v uint64) error {
	if v == 0 {
		return domain.Invalid("EXPECTED_VERSION_REQUIRED", "必须提供expectedVersion")
	}
	return nil
}
