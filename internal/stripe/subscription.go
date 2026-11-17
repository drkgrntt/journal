package stripe

import (
	"journal/internal/models"
	"time"

	stripego "github.com/stripe/stripe-go/v84"
)

// SubscriptionState maps a Stripe subscription onto our status and the end of
// the period that's been paid for. Stripe's own "trialing" can't happen (the
// trial is ours, no card), but is treated as active if it ever does. Statuses
// that mean "nothing was ever paid" return ok=false and are ignored.
func SubscriptionState(sub *stripego.Subscription) (status string, periodEnd *time.Time, ok bool) {
	switch sub.Status {
	case stripego.SubscriptionStatusActive, stripego.SubscriptionStatusTrialing:
		status = models.SubscriptionActive
	case stripego.SubscriptionStatusPastDue:
		status = models.SubscriptionPastDue
	case stripego.SubscriptionStatusCanceled, stripego.SubscriptionStatusUnpaid,
		stripego.SubscriptionStatusIncompleteExpired, stripego.SubscriptionStatusPaused:
		status = models.SubscriptionCanceled
	default: // incomplete: the first payment hasn't gone through
		return "", nil, false
	}

	// As of the current Stripe API the period lives on the items, not the
	// subscription itself.
	if sub.Items != nil {
		var latest int64
		for _, item := range sub.Items.Data {
			if item.CurrentPeriodEnd > latest {
				latest = item.CurrentPeriodEnd
			}
		}
		if latest > 0 {
			end := time.Unix(latest, 0).UTC()
			periodEnd = &end
		}
	}
	return status, periodEnd, true
}
