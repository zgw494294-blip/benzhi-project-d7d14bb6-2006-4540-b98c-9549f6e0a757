package application

import "timber-release-gate/internal/domain"

type ReviewInput struct {
	Decision                     domain.ReviewDecision
	Comments, ResolvedFindingIDs []string
}
