package attestation

import (
	"context"

	"github.com/pycify/phylax/internal/phylax"
)

type Service struct {
	phylax *phylax.Service
}

func NewService(phylaxService *phylax.Service) *Service {
	return &Service{phylax: phylaxService}
}

func (s *Service) CreateChallenge(ctx context.Context, req phylax.CreateChallengeRequest) (phylax.CreateChallengeResponse, error) {
	return s.phylax.CreateChallenge(ctx, req)
}

func (s *Service) VerifyAttestation(ctx context.Context, req phylax.VerifyAttestationRequest) (phylax.VerifyAttestationResponse, error) {
	return s.phylax.VerifyAttestation(ctx, req)
}

func (s *Service) Introspect(ctx context.Context, token string) phylax.IntrospectResponse {
	return s.phylax.Introspect(ctx, token)
}
