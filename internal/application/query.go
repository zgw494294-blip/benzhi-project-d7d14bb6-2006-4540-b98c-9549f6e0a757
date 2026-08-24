package application

import "timber-release-gate/internal/domain"

func ValidateCertificate(c *domain.WorkReleaseCertificate) bool {
	return c != nil && c.Digest() == c.CertificateDigest
}
