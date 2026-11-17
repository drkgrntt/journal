// Creates the Stripe products and prices the app expects (found by lookup key)
// and makes sure the customer portal has a configuration. Safe to re-run: it
// only creates what's missing. It acts on whatever STRIPE_SECRET_KEY is set,
// test or live, and says which.
package main

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	stripego "github.com/stripe/stripe-go/v84"
	bpconfig "github.com/stripe/stripe-go/v84/billingportal/configuration"
	"github.com/stripe/stripe-go/v84/price"
	"github.com/stripe/stripe-go/v84/product"
)

type planSpec struct {
	key      string
	nickname string
	amount   int64
	interval string // empty = one-time
}

var specs = []planSpec{
	{"contour_monthly", "Monthly", 300, "month"},
	{"contour_annual", "Annual", 2400, "year"},
	{"contour_lifetime", "Lifetime", 4800, ""},
}

func main() {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		fmt.Println("STRIPE_SECRET_KEY is not set")
		os.Exit(1)
	}
	stripego.Key = key
	mode := "LIVE"
	if strings.HasPrefix(key, "sk_test_") {
		mode = "TEST"
	}
	fmt.Printf("Stripe mode: %s\n", mode)
	if mode == "LIVE" && os.Getenv("CONFIRM_LIVE") != "yes" {
		fmt.Println("Refusing to touch a live account without CONFIRM_LIVE=yes")
		os.Exit(1)
	}

	var productID string
	for _, s := range specs {
		iter := price.List(&stripego.PriceListParams{
			Active:     stripego.Bool(true),
			LookupKeys: stripego.StringSlice([]string{s.key}),
		})
		if iter.Next() {
			fmt.Printf("exists   %-18s %s\n", s.key, iter.Price().ID)
			productID = iter.Price().Product.ID
			continue
		}
		if err := iter.Err(); err != nil {
			fail(err)
		}

		if productID == "" {
			p, err := product.New(&stripego.ProductParams{
				Name:        stripego.String("Contour Journal"),
				Description: stripego.String("Routines, custom topics, themes, entry templates, and reminder emails."),
			})
			if err != nil {
				fail(err)
			}
			productID = p.ID
		}

		params := &stripego.PriceParams{
			Product:    stripego.String(productID),
			Currency:   stripego.String("usd"),
			UnitAmount: stripego.Int64(s.amount),
			LookupKey:  stripego.String(s.key),
			Nickname:   stripego.String(s.nickname),
		}
		if s.interval != "" {
			params.Recurring = &stripego.PriceRecurringParams{Interval: stripego.String(s.interval)}
		}
		created, err := price.New(params)
		if err != nil {
			fail(err)
		}
		fmt.Printf("created  %-18s %s\n", s.key, created.ID)
	}

	// The portal needs a configuration; Stripe has none by default.
	cfgs := bpconfig.List(&stripego.BillingPortalConfigurationListParams{Active: stripego.Bool(true)})
	hasDefault := false
	for cfgs.Next() {
		if cfgs.BillingPortalConfiguration().IsDefault {
			hasDefault = true
		}
	}
	if err := cfgs.Err(); err != nil {
		fail(err)
	}
	if hasDefault {
		fmt.Println("exists   customer portal default configuration")
		return
	}
	cfg, err := bpconfig.New(&stripego.BillingPortalConfigurationParams{
		BusinessProfile: &stripego.BillingPortalConfigurationBusinessProfileParams{
			Headline: stripego.String("Contour Journal"),
		},
		Features: &stripego.BillingPortalConfigurationFeaturesParams{
			InvoiceHistory:      &stripego.BillingPortalConfigurationFeaturesInvoiceHistoryParams{Enabled: stripego.Bool(true)},
			PaymentMethodUpdate: &stripego.BillingPortalConfigurationFeaturesPaymentMethodUpdateParams{Enabled: stripego.Bool(true)},
			SubscriptionCancel: &stripego.BillingPortalConfigurationFeaturesSubscriptionCancelParams{
				Enabled: stripego.Bool(true),
				Mode:    stripego.String("at_period_end"),
			},
		},
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("created  customer portal configuration %s (is_default=%v)\n", cfg.ID, cfg.IsDefault)
}

func fail(err error) {
	fmt.Println("error:", err)
	os.Exit(1)
}
