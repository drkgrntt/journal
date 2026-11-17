package stripe

import (
	"testing"
	"time"

	"github.com/stripe/stripe-go/v84"

	"journal/internal/models"
)

// subWith builds a subscription with the given status and one item per end.
func subWith(status stripe.SubscriptionStatus, itemEnds ...int64) *stripe.Subscription {
	sub := &stripe.Subscription{Status: status}
	if itemEnds == nil {
		return sub
	}
	sub.Items = &stripe.SubscriptionItemList{}
	for _, end := range itemEnds {
		sub.Items.Data = append(sub.Items.Data, &stripe.SubscriptionItem{CurrentPeriodEnd: end})
	}
	return sub
}

func TestSubscriptionStateStatus(t *testing.T) {
	tests := []struct {
		name   string
		status stripe.SubscriptionStatus
		want   string
		wantOK bool
	}{
		{"active", stripe.SubscriptionStatusActive, models.SubscriptionActive, true},
		{"trialing", stripe.SubscriptionStatusTrialing, models.SubscriptionActive, true},
		{"past_due", stripe.SubscriptionStatusPastDue, models.SubscriptionPastDue, true},
		{"canceled", stripe.SubscriptionStatusCanceled, models.SubscriptionCanceled, true},
		{"unpaid", stripe.SubscriptionStatusUnpaid, models.SubscriptionCanceled, true},
		{"incomplete_expired", stripe.SubscriptionStatusIncompleteExpired, models.SubscriptionCanceled, true},
		{"paused", stripe.SubscriptionStatusPaused, models.SubscriptionCanceled, true},
		{"incomplete", stripe.SubscriptionStatusIncomplete, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, end, ok := SubscriptionState(subWith(tt.status))
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("got (%q, ok=%v), want (%q, ok=%v)", got, ok, tt.want, tt.wantOK)
			}
			if end != nil {
				t.Errorf("periodEnd = %v, want nil", end)
			}
		})
	}
}

func TestSubscriptionStatePeriodEnd(t *testing.T) {
	tests := []struct {
		name string
		sub  *stripe.Subscription
		want int64 // 0 means nil
	}{
		{"nil items", subWith(stripe.SubscriptionStatusActive), 0},
		{"empty items", subWith(stripe.SubscriptionStatusActive, []int64{}...), 0},
		{"all zero", subWith(stripe.SubscriptionStatusActive, 0, 0), 0},
		{"single item", subWith(stripe.SubscriptionStatusActive, 1700000000), 1700000000},
		{"latest wins", subWith(stripe.SubscriptionStatusActive, 1700000000, 1800000000, 1750000000), 1800000000},
		{"zero ignored", subWith(stripe.SubscriptionStatusPastDue, 0, 1700000000), 1700000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, end, ok := SubscriptionState(tt.sub)
			if !ok {
				t.Fatal("ok = false, want true")
			}
			if tt.want == 0 {
				if end != nil {
					t.Errorf("periodEnd = %v, want nil", end)
				}
				return
			}
			if end == nil {
				t.Fatal("periodEnd = nil, want a time")
			}
			if want := time.Unix(tt.want, 0).UTC(); !end.Equal(want) {
				t.Errorf("periodEnd = %v, want %v", end, want)
			}
		})
	}
}
