package events

import (
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func donationToProto(d domain.Donation) *billingv1.Donation {
	return &billingv1.Donation{
		DonationId:        d.ID,
		PayerProfileId:    d.PayerProfileID,
		TargetType:        donationTargetToProto(d.TargetType),
		TargetId:          d.TargetID,
		Amount:            moneyToProto(d.Amount),
		Status:            paymentStatusToProto(d.Status),
		Provider:          d.Provider,
		ProviderPaymentId: d.ProviderPaymentID,
		CreatedAt:         timestamppb.New(d.CreatedAt),
		UpdatedAt:         timestamppb.New(d.UpdatedAt),
	}
}

func boostToProto(b domain.Boost) *billingv1.Boost {
	return &billingv1.Boost{
		BoostId:        b.ID,
		AnimalId:       b.AnimalID,
		OwnerProfileId: b.OwnerProfileID,
		DonationId:     b.DonationID,
		StartsAt:       timestamppb.New(b.StartsAt),
		ExpiresAt:      timestamppb.New(b.ExpiresAt),
		Active:         b.Active,
	}
}

func entitlementToProto(e domain.Entitlement) *billingv1.Entitlement {
	var resourceID *string
	if e.ResourceID != "" {
		resourceID = &e.ResourceID
	}
	return &billingv1.Entitlement{
		EntitlementId:  e.ID,
		OwnerProfileId: e.OwnerProfileID,
		Type:           entitlementTypeToProto(e.Type),
		ResourceId:     resourceID,
		StartsAt:       timestamppb.New(e.StartsAt),
		ExpiresAt:      timestamppb.New(e.ExpiresAt),
		Active:         e.Active,
	}
}

func moneyToProto(m domain.Money) *commonv1.MoneyAmount {
	return &commonv1.MoneyAmount{CurrencyCode: m.CurrencyCode, Units: m.Units, Nanos: m.Nanos}
}

func donationTargetToProto(t domain.DonationTargetType) billingv1.DonationTargetType {
	switch t {
	case domain.DonationTargetShelter:
		return billingv1.DonationTargetType_DONATION_TARGET_TYPE_SHELTER
	case domain.DonationTargetAnimal:
		return billingv1.DonationTargetType_DONATION_TARGET_TYPE_ANIMAL
	default:
		return billingv1.DonationTargetType_DONATION_TARGET_TYPE_UNSPECIFIED
	}
}

func paymentStatusToProto(s domain.PaymentStatus) billingv1.PaymentStatus {
	switch s {
	case domain.PaymentPending:
		return billingv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case domain.PaymentSucceeded:
		return billingv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED
	case domain.PaymentFailed:
		return billingv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case domain.PaymentCancelled:
		return billingv1.PaymentStatus_PAYMENT_STATUS_CANCELLED
	case domain.PaymentRefunded:
		return billingv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return billingv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func entitlementTypeToProto(t domain.EntitlementType) billingv1.EntitlementType {
	switch t {
	case domain.EntitlementAdvancedFilters:
		return billingv1.EntitlementType_ENTITLEMENT_TYPE_ADVANCED_FILTERS
	case domain.EntitlementExtendedAnimalStats:
		return billingv1.EntitlementType_ENTITLEMENT_TYPE_EXTENDED_ANIMAL_STATS
	case domain.EntitlementAnimalBoost:
		return billingv1.EntitlementType_ENTITLEMENT_TYPE_ANIMAL_BOOST
	default:
		return billingv1.EntitlementType_ENTITLEMENT_TYPE_UNSPECIFIED
	}
}
