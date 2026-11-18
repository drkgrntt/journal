package controllers

import (
	"bytes"
	"encoding/json"
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
	"gorm.io/gorm"
)

const testWebhookSecret = "whsec_test_secret"

// webhookEvent builds an event payload with the given type and data object.
func webhookEvent(eventType string, object map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{
		"id":          "evt_test",
		"object":      "event",
		"api_version": stripe.APIVersion,
		"type":        eventType,
		"data":        map[string]any{"object": object},
	})
	return b
}

func subscriptionEvent(eventType, id, customer, status string, periodEnd int64) []byte {
	return webhookEvent(eventType, map[string]any{
		"id":       id,
		"object":   "subscription",
		"customer": customer,
		"status":   status,
		"items": map[string]any{
			"object": "list",
			"data":   []any{map[string]any{"id": "si_1", "object": "subscription_item", "current_period_end": periodEnd}},
		},
	})
}

func checkoutEvent(mode, userID, paymentStatus, subID string) []byte {
	obj := map[string]any{
		"id":                  "cs_test",
		"object":              "checkout.session",
		"mode":                mode,
		"client_reference_id": userID,
		"payment_status":      paymentStatus,
	}
	if subID != "" {
		obj["subscription"] = subID
	}
	return webhookEvent("checkout.session.completed", obj)
}

func postWebhook(t *testing.T, tx *gorm.DB, payload []byte, signed bool) int {
	t.Helper()
	t.Setenv("STRIPE_WEBHOOK_SECRET", testWebhookSecret)
	app := newTestApp(t, tx, &StripeController{})
	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", bytes.NewReader(payload))
	if signed {
		sp := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: testWebhookSecret})
		req.Header.Set("Stripe-Signature", sp.Header)
	} else {
		req.Header.Set("Stripe-Signature", "t=1,v1=bad")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func reload(t *testing.T, tx *gorm.DB, u *models.User) models.User {
	t.Helper()
	var got models.User
	if err := tx.First(&got, "id = ?", u.ID).Error; err != nil {
		t.Fatal(err)
	}
	return got
}

func withCustomer(id string) func(*models.User) {
	return func(u *models.User) { u.StripeCustomerID = id }
}

func TestWebhookBadSignature(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		if got := postWebhook(t, tx, webhookEvent("ping", map[string]any{"id": "x"}), false); got != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", got)
		}
	})
}

func TestWebhookUnknownEventIsNoOp(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		if got := postWebhook(t, tx, webhookEvent("invoice.paid", map[string]any{"id": "in_1"}), true); got != http.StatusOK {
			t.Fatalf("status = %d, want 200", got)
		}
	})
}

func TestWebhookCheckoutSubscriptionLinksID(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "link@example.com")
		if got := postWebhook(t, tx, checkoutEvent("subscription", u.ID.String(), "paid", "sub_1"), true); got != 200 {
			t.Fatalf("status = %d", got)
		}
		if g := reload(t, tx, u); g.StripeSubscriptionID != "sub_1" {
			t.Fatalf("subscription id = %q", g.StripeSubscriptionID)
		}
	})
}

func TestWebhookCheckoutSubscriptionIgnoredForLifetimeAndComped(t *testing.T) {
	for _, status := range []string{models.SubscriptionLifetime, models.SubscriptionComped} {
		t.Run(status, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				u := testutil.User(t, tx, status+"@example.com", func(u *models.User) { u.SubscriptionStatus = status })
				postWebhook(t, tx, checkoutEvent("subscription", u.ID.String(), "paid", "sub_1"), true)
				if g := reload(t, tx, u); g.StripeSubscriptionID != "" {
					t.Fatalf("subscription id = %q, want empty", g.StripeSubscriptionID)
				}
			})
		})
	}
}

func TestWebhookCheckoutPaymentMode(t *testing.T) {
	t.Run("paid marks lifetime", func(t *testing.T) {
		testutil.Tx(t, func(tx *gorm.DB) {
			// No stripe_subscription_id, so the Stripe cancel call isn't reached.
			u := testutil.User(t, tx, "life@example.com")
			if got := postWebhook(t, tx, checkoutEvent("payment", u.ID.String(), "paid", ""), true); got != 200 {
				t.Fatalf("status = %d", got)
			}
			g := reload(t, tx, u)
			if g.SubscriptionStatus != models.SubscriptionLifetime || g.CurrentPeriodEnd != nil || g.StripeSubscriptionID != "" {
				t.Fatalf("got status=%q end=%v sub=%q", g.SubscriptionStatus, g.CurrentPeriodEnd, g.StripeSubscriptionID)
			}
		})
	})
	t.Run("unpaid does nothing", func(t *testing.T) {
		testutil.Tx(t, func(tx *gorm.DB) {
			u := testutil.User(t, tx, "unpaid@example.com")
			before := reload(t, tx, u).SubscriptionStatus
			postWebhook(t, tx, checkoutEvent("payment", u.ID.String(), "unpaid", ""), true)
			if g := reload(t, tx, u); g.SubscriptionStatus != before {
				t.Fatalf("status changed to %q", g.SubscriptionStatus)
			}
		})
	})
}

func TestWebhookCheckoutBadReference(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		if got := postWebhook(t, tx, checkoutEvent("payment", "not-a-uuid", "paid", ""), true); got != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", got)
		}
	})
}

func TestWebhookSubscriptionStatusMapping(t *testing.T) {
	end := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second).UTC()
	tests := []struct {
		event, stripeStatus, want string
	}{
		{"customer.subscription.created", "active", models.SubscriptionActive},
		{"customer.subscription.updated", "trialing", models.SubscriptionActive},
		{"customer.subscription.updated", "past_due", models.SubscriptionPastDue},
		{"customer.subscription.updated", "unpaid", models.SubscriptionCanceled},
		{"customer.subscription.deleted", "canceled", models.SubscriptionCanceled},
	}
	for _, tt := range tests {
		t.Run(tt.event+"/"+tt.stripeStatus, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				u := testutil.User(t, tx, "map@example.com", withCustomer("cus_1"))
				body := subscriptionEvent(tt.event, "sub_1", "cus_1", tt.stripeStatus, end.Unix())
				if got := postWebhook(t, tx, body, true); got != 200 {
					t.Fatalf("status = %d", got)
				}
				g := reload(t, tx, u)
				if g.SubscriptionStatus != tt.want || g.StripeSubscriptionID != "sub_1" {
					t.Fatalf("status=%q sub=%q", g.SubscriptionStatus, g.StripeSubscriptionID)
				}
				if g.CurrentPeriodEnd == nil || !g.CurrentPeriodEnd.Equal(end) {
					t.Fatalf("period end = %v, want %v", g.CurrentPeriodEnd, end)
				}
			})
		})
	}
}

func TestWebhookSubscriptionIncompleteIgnored(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "inc@example.com", withCustomer("cus_1"))
		before := reload(t, tx, u)
		postWebhook(t, tx, subscriptionEvent("customer.subscription.created", "sub_1", "cus_1", "incomplete", 1), true)
		if g := reload(t, tx, u); g.SubscriptionStatus != before.SubscriptionStatus || g.StripeSubscriptionID != "" {
			t.Fatalf("changed: status=%q sub=%q", g.SubscriptionStatus, g.StripeSubscriptionID)
		}
	})
}

func TestWebhookSubscriptionIdempotent(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "idem@example.com", withCustomer("cus_1"))
		body := subscriptionEvent("customer.subscription.updated", "sub_1", "cus_1", "active", time.Now().Add(time.Hour).Unix())
		postWebhook(t, tx, body, true)
		first := reload(t, tx, u)
		if got := postWebhook(t, tx, body, true); got != 200 {
			t.Fatalf("redelivery status = %d", got)
		}
		second := reload(t, tx, u)
		if first.SubscriptionStatus != second.SubscriptionStatus || first.StripeSubscriptionID != second.StripeSubscriptionID ||
			first.CurrentPeriodEnd == nil || second.CurrentPeriodEnd == nil || !first.CurrentPeriodEnd.Equal(*second.CurrentPeriodEnd) {
			t.Fatalf("state changed on redelivery: %+v vs %+v", first, second)
		}
	})
}

func TestWebhookStaleSubscriptionIgnored(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "stale@example.com", withCustomer("cus_1"), func(u *models.User) {
			u.StripeSubscriptionID = "sub_new"
			u.SubscriptionStatus = models.SubscriptionActive
		})
		postWebhook(t, tx, subscriptionEvent("customer.subscription.deleted", "sub_old", "cus_1", "canceled", 1), true)
		g := reload(t, tx, u)
		if g.SubscriptionStatus != models.SubscriptionActive || g.StripeSubscriptionID != "sub_new" {
			t.Fatalf("stale event applied: status=%q sub=%q", g.SubscriptionStatus, g.StripeSubscriptionID)
		}
	})
}

func TestWebhookSubscriptionIgnoredForLifetimeAndComped(t *testing.T) {
	for _, status := range []string{models.SubscriptionLifetime, models.SubscriptionComped} {
		t.Run(status, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				u := testutil.User(t, tx, status+"@example.com", withCustomer("cus_1"), func(u *models.User) { u.SubscriptionStatus = status })
				postWebhook(t, tx, subscriptionEvent("customer.subscription.deleted", "sub_1", "cus_1", "canceled", 1), true)
				if g := reload(t, tx, u); g.SubscriptionStatus != status {
					t.Fatalf("status = %q, want %q", g.SubscriptionStatus, status)
				}
			})
		})
	}
}

// An unknown customer currently yields 400 (Stripe will retry), not a 500.
func TestWebhookSubscriptionUnknownCustomer(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		got := postWebhook(t, tx, subscriptionEvent("customer.subscription.updated", "sub_1", "cus_nobody", "active", 1), true)
		if got != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", got)
		}
	})
}
