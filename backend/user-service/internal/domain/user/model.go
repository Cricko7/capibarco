package user

import (
	"time"
)

type ProfileType int

const (
	ProfileTypeUnspecified ProfileType = iota
	ProfileTypeUser
	ProfileTypeShelter
	ProfileTypeKennel
)

type Profile struct {
	ID          string
	AuthUserID  string
	ProfileType ProfileType
	DisplayName string
	Bio         string
	AvatarURL   string
	City        string
	Visibility  int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Review struct {
	ID              string
	TargetProfileID string
	AuthorProfileID string
	Rating          int32
	Text            string
	MatchID         string
	Visibility      int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Reputation struct {
	ProfileID     string
	AverageRating float64
	ReviewsCount  int32
	UpdatedAt     time.Time
}

func (r Review) Validate() error {
	if r.TargetProfileID == "" || r.AuthorProfileID == "" || r.Text == "" {
		return ErrInvalidArgument
	}
	if r.Rating < 1 || r.Rating > 5 {
		return ErrInvalidArgument
	}
	return nil
}

func (p Profile) Validate() error {
	if p.ID == "" || p.DisplayName == "" {
		return ErrInvalidArgument
	}
	return nil
}
