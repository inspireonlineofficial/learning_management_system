package courses

import (
	"testing"

	domaincourses "lms-backend/internal/domain/courses"
)

func TestValidateCoursePricing(t *testing.T) {
	tests := []struct {
		name          string
		existing      domaincourses.PriceType
		priceType     string
		price         float64
		currency      string
		totalEnrolled int
		wantType      domaincourses.PriceType
		wantPrice     float64
		wantCurrency  string
		wantErr       bool
	}{
		{name: "defaults empty type and currency", price: 100, wantType: domaincourses.PriceTypePaid, wantPrice: 100, wantCurrency: "BDT"},
		{name: "accepts free zero price", priceType: "free", price: 0, currency: "bdt", wantType: domaincourses.PriceTypeFree, wantPrice: 0, wantCurrency: "BDT"},
		{name: "rejects free nonzero price", priceType: "free", price: 1, wantErr: true},
		{name: "rejects paid zero price", priceType: "paid", price: 0, wantErr: true},
		{name: "rejects type change after enrollment", existing: domaincourses.PriceTypePaid, priceType: "free", totalEnrolled: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotPrice, gotCurrency, err := validateCoursePricing(tt.existing, tt.priceType, tt.price, tt.currency, tt.totalEnrolled)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateCoursePricing() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotType != tt.wantType || gotPrice != tt.wantPrice || gotCurrency != tt.wantCurrency {
				t.Fatalf("validateCoursePricing() = (%q, %v, %q), want (%q, %v, %q)", gotType, gotPrice, gotCurrency, tt.wantType, tt.wantPrice, tt.wantCurrency)
			}
		})
	}
}
