package stripe

import (
	"fmt"
	"journal/internal/logger"
	"os"
	"sync"

	_ "github.com/joho/godotenv/autoload"
	stripego "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/price"
)

func init() {
	logger.Info("Initializing Stripe")
	stripego.Key = os.Getenv("STRIPE_SECRET_KEY")
}

// Plan is one way to pay for the paid features. Prices are found in Stripe by
// lookup key rather than by ID, so the same code works against test and live
// accounts; `go run cmd/stripesetup/stripesetup.go` creates them.
type Plan string

const (
	PlanMonthly  Plan = "monthly"
	PlanAnnual   Plan = "annual"
	PlanLifetime Plan = "lifetime"
)

var Plans = []Plan{PlanMonthly, PlanAnnual, PlanLifetime}

var lookupKeys = map[Plan]string{
	PlanMonthly:  "contour_monthly",
	PlanAnnual:   "contour_annual",
	PlanLifetime: "contour_lifetime",
}

func ParsePlan(s string) (Plan, bool) {
	p := Plan(s)
	_, ok := lookupKeys[p]
	return p, ok
}

func (p Plan) LookupKey() string { return lookupKeys[p] }

// IsRecurring is false only for lifetime, which is a one-time payment.
func (p Plan) IsRecurring() bool { return p != PlanLifetime }

var (
	priceCache   = make(map[Plan]*stripego.Price)
	priceCacheMu sync.RWMutex
)

// GetPlanPrice returns the Stripe price for a plan. Failures aren't cached.
func GetPlanPrice(plan Plan) (*stripego.Price, error) {
	priceCacheMu.RLock()
	cached, ok := priceCache[plan]
	priceCacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	params := &stripego.PriceListParams{
		Active:     stripego.Bool(true),
		LookupKeys: stripego.StringSlice([]string{plan.LookupKey()}),
	}
	iter := price.List(params)
	if iter.Next() {
		found := iter.Price()
		priceCacheMu.Lock()
		priceCache[plan] = found
		priceCacheMu.Unlock()
		return found, nil
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no active stripe price with lookup key %q", plan.LookupKey())
}

// PlanLabel is a short price description like "$3 / month", or "" if the
// price can't be loaded.
func PlanLabel(plan Plan) string {
	p, err := GetPlanPrice(plan)
	if err != nil {
		logger.Error("Error getting plan price", "plan", string(plan), "error", err)
		return ""
	}
	amount := float64(p.UnitAmount) / 100
	label := fmt.Sprintf("$%.2f", amount)
	if amount == float64(int(amount)) {
		label = fmt.Sprintf("$%d", int(amount))
	}
	switch {
	case p.Recurring == nil:
		return label + " once"
	case p.Recurring.Interval == stripego.PriceRecurringIntervalYear:
		return label + " / year"
	default:
		return label + " / month"
	}
}
