package notification

import "time"

type Channel int32

const (
	ChannelUnspecified Channel = 0
	ChannelPush        Channel = 1
	ChannelInApp       Channel = 2
	ChannelEmail       Channel = 3
)

type Type int32

const (
	TypeUnspecified       Type = 0
	TypeMatchCreated      Type = 1
	TypeChatMessage       Type = 2
	TypeDonationSucceeded Type = 3
	TypeBoostActivated    Type = 4
	TypeReviewCreated     Type = 5
)

type Status int32

const (
	StatusUnspecified Status = 0
	StatusPending     Status = 1
	StatusDelivered   Status = 2
	StatusFailed      Status = 3
	StatusRead        Status = 4
)

type DeviceToken struct {
	ID        string
	ProfileID string
	Token     string
	Platform  string
	Locale    string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Notification struct {
	ID                 string
	RecipientProfileID string
	Type               Type
	Channels           []Channel
	Title              string
	Body               string
	Data               map[string]string
	Status             Status
	ReadAt             *time.Time
	CreatedAt          time.Time
	IdempotencyKey     string
}

type Preference struct {
	RecipientProfileID string
	PushEnabled        bool
	InAppEnabled       bool
	EmailEnabled       bool
	QuietHoursEnabled  bool
	QuietHoursStart    string
	QuietHoursEnd      string
	Muted              bool
}

type PageRequest struct {
	PageSize  int32
	PageToken string
}

func DefaultPreference(recipientProfileID string) Preference {
	return Preference{
		RecipientProfileID: recipientProfileID,
		PushEnabled:        true,
		InAppEnabled:       true,
		EmailEnabled:       true,
	}
}
