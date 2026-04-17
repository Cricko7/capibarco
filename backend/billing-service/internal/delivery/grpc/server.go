package grpcdelivery

import (
	"context"

	"github.com/go-playground/validator/v10"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	billingv1.UnimplementedBillingServiceServer
	service  *application.Service
	validate *validator.Validate
}

func NewServer(service *application.Service) *Server {
	return &Server{service: service, validate: validator.New()}
}

func (s *Server) CreateDonationIntent(ctx context.Context, req *billingv1.CreateDonationIntentRequest) (*billingv1.CreateDonationIntentResponse, error) {
	if err := s.validate.Var(req.GetPayerProfileId(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	amount, err := moneyFromProto(req.GetAmount())
	if err != nil {
		return nil, toStatus(err)
	}
	result, err := s.service.CreateDonationIntent(ctx, application.CreateDonationIntentInput{
		PayerProfileID: req.GetPayerProfileId(),
		TargetType:     donationTargetFromProto(req.GetTargetType()),
		TargetID:       req.GetTargetId(),
		Amount:         amount,
		Provider:       req.GetProvider(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.CreateDonationIntentResponse{
		Donation:     donationToProto(result.Donation),
		PaymentUrl:   result.Payment.PaymentURL,
		ClientSecret: result.Payment.ClientSecret,
	}, nil
}

func (s *Server) ConfirmDonation(ctx context.Context, req *billingv1.ConfirmDonationRequest) (*billingv1.ConfirmDonationResponse, error) {
	result, err := s.service.ConfirmDonation(ctx, application.ConfirmDonationInput{
		DonationID:        req.GetDonationId(),
		ProviderPaymentID: req.GetProviderPaymentId(),
		IdempotencyKey:    req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.ConfirmDonationResponse{Donation: donationToProto(result.Donation)}, nil
}

func (s *Server) GetDonation(ctx context.Context, req *billingv1.GetDonationRequest) (*billingv1.GetDonationResponse, error) {
	donation, err := s.service.GetDonation(ctx, req.GetDonationId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.GetDonationResponse{Donation: donationToProto(donation)}, nil
}

func (s *Server) ListDonations(ctx context.Context, req *billingv1.ListDonationsRequest) (*billingv1.ListDonationsResponse, error) {
	page := req.GetPage()
	items, next, err := s.service.ListDonations(ctx, application.ListDonationsFilter{
		ProfileID:  req.GetProfileId(),
		TargetType: donationTargetFromProto(req.GetTargetType()),
		PageSize:   int(page.GetPageSize()),
		PageToken:  page.GetPageToken(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &billingv1.ListDonationsResponse{Page: &commonv1.PageResponse{NextPageToken: next}}
	for _, item := range items {
		resp.Donations = append(resp.Donations, donationToProto(item))
	}
	return resp, nil
}

func (s *Server) CreateBoost(ctx context.Context, req *billingv1.CreateBoostRequest) (*billingv1.CreateBoostResponse, error) {
	result, err := s.service.CreateBoost(ctx, application.CreateBoostInput{
		AnimalID:       req.GetAnimalId(),
		OwnerProfileID: req.GetOwnerProfileId(),
		DonationID:     req.GetDonationId(),
		Duration:       req.GetDuration().AsDuration(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.CreateBoostResponse{Boost: boostToProto(result.Boost)}, nil
}

func (s *Server) CancelBoost(ctx context.Context, req *billingv1.CancelBoostRequest) (*billingv1.CancelBoostResponse, error) {
	result, err := s.service.CancelBoost(ctx, application.CancelBoostInput{BoostID: req.GetBoostId(), Reason: req.GetReason()})
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.CancelBoostResponse{Boost: boostToProto(result.Boost)}, nil
}

func (s *Server) PurchaseEntitlement(ctx context.Context, req *billingv1.PurchaseEntitlementRequest) (*billingv1.PurchaseEntitlementResponse, error) {
	amount, err := moneyFromProto(req.GetAmount())
	if err != nil {
		return nil, toStatus(err)
	}
	result, err := s.service.PurchaseEntitlement(ctx, application.PurchaseEntitlementInput{
		OwnerProfileID: req.GetOwnerProfileId(),
		Type:           entitlementTypeFromProto(req.GetType()),
		ResourceID:     req.GetResourceId(),
		Amount:         amount,
		Duration:       req.GetDuration().AsDuration(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &billingv1.PurchaseEntitlementResponse{
		Entitlement: entitlementToProto(result.Entitlement),
		Donation:    donationToProto(result.Donation),
	}, nil
}

func (s *Server) GetEntitlements(ctx context.Context, req *billingv1.GetEntitlementsRequest) (*billingv1.GetEntitlementsResponse, error) {
	types := make([]domain.EntitlementType, 0, len(req.GetTypes()))
	for _, typ := range req.GetTypes() {
		types = append(types, entitlementTypeFromProto(typ))
	}
	items, err := s.service.GetEntitlements(ctx, application.GetEntitlementsFilter{
		OwnerProfileID: req.GetOwnerProfileId(),
		Types:          types,
		ResourceID:     req.GetResourceId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &billingv1.GetEntitlementsResponse{}
	for _, item := range items {
		resp.Entitlements = append(resp.Entitlements, entitlementToProto(item))
	}
	return resp, nil
}

func (s *Server) ListLedgerEntries(ctx context.Context, req *billingv1.ListLedgerEntriesRequest) (*billingv1.ListLedgerEntriesResponse, error) {
	page := req.GetPage()
	items, next, err := s.service.ListLedgerEntries(ctx, application.ListLedgerEntriesFilter{
		ProfileID: req.GetProfileId(),
		PageSize:  int(page.GetPageSize()),
		PageToken: page.GetPageToken(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &billingv1.ListLedgerEntriesResponse{Page: &commonv1.PageResponse{NextPageToken: next}}
	for _, item := range items {
		resp.Entries = append(resp.Entries, ledgerToProto(item))
	}
	return resp, nil
}
