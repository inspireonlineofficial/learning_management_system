package courses

import (
	"strings"

	domaincourses "lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"
)

func validateCoursePricing(existing domaincourses.PriceType, rawPriceType string, price float64, currency string, totalEnrolled int) (domaincourses.PriceType, float64, string, error) {
	priceType := domaincourses.PriceType(strings.TrimSpace(rawPriceType))
	if priceType == "" {
		priceType = domaincourses.PriceTypePaid
	}
	if priceType != domaincourses.PriceTypeFree && priceType != domaincourses.PriceTypePaid {
		return "", 0, "", apperrors.NewSimpleValidationError("INVALID_PRICE_TYPE", "price_type must be free or paid")
	}
	if existing != "" && existing != priceType && totalEnrolled > 0 {
		return "", 0, "", apperrors.NewSimpleValidationError("PRICE_TYPE_LOCKED", "price_type cannot be changed after enrollments exist")
	}
	if currency = strings.ToUpper(strings.TrimSpace(currency)); currency == "" {
		currency = "BDT"
	}
	if priceType == domaincourses.PriceTypeFree {
		if price != 0 {
			return "", 0, "", apperrors.NewSimpleValidationError("FREE_PRICE_INVALID", "free courses must have zero price")
		}
		return priceType, 0, currency, nil
	}
	if price <= 0 {
		return "", 0, "", apperrors.NewSimpleValidationError("PAID_PRICE_REQUIRED", "paid courses require a positive price")
	}
	return priceType, price, currency, nil
}
