package domain

import (
	"fmt"
	"regexp"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Money is a decimal money amount without floating point rounding.
type Money struct {
	CurrencyCode string
	Units        int64
	Nanos        int32
}

// NewMoney validates an ISO-4217-like positive amount.
func NewMoney(currencyCode string, units int64, nanos int32) (Money, error) {
	if !currencyCodePattern.MatchString(currencyCode) {
		return Money{}, fmt.Errorf("%w: currency code must be three uppercase letters", ErrInvalidMoney)
	}
	if units < 0 || nanos < 0 || nanos >= 1_000_000_000 {
		return Money{}, fmt.Errorf("%w: amount must be positive and nanos must be in [0, 1e9)", ErrInvalidMoney)
	}
	if units == 0 && nanos == 0 {
		return Money{}, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidMoney)
	}
	return Money{CurrencyCode: currencyCode, Units: units, Nanos: nanos}, nil
}
