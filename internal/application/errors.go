package application

import "timber-release-gate/internal/domain"

func StateError(e error) *domain.Error {
	if x, ok := e.(*domain.Error); ok {
		return x
	}
	return &domain.Error{Code: "BUSINESS_ERROR", Message: e.Error()}
}
