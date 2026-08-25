package application

import (
	"fmt"
	"timber-release-gate/internal/domain"
)

func auditIntegrityError(err error) error {
	return fmt.Errorf("审计链验证失败: %w", &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()})
}

func StateError(e error) *domain.Error {
	if x, ok := e.(*domain.Error); ok {
		return x
	}
	return &domain.Error{Code: "BUSINESS_ERROR", Message: e.Error()}
}
