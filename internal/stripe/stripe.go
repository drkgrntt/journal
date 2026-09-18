package stripe

import (
	"journal/internal/logger"
	"os"
	"sync"

	_ "github.com/joho/godotenv/autoload"
	stripego "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/price"
	"github.com/stripe/stripe-go/v84/product"
)

func init() {
	logger.Info("Initializing Stripe")
	stripego.Key = os.Getenv("STRIPE_SECRET_KEY")
}

var (
	priceMap   = make(map[string]*stripego.Price)
	priceMapMu sync.RWMutex
)

func GetPrice(priceId string) *stripego.Price {
	priceMapMu.RLock()
	if price, ok := priceMap[priceId]; ok {
		priceMapMu.RUnlock()
		return price
	}
	priceMapMu.RUnlock()

	params := &stripego.PriceParams{}
	result, err := price.Get(priceId, params)
	if err != nil {
		logger.Error("Error getting price", "error", err)
	}

	product, err := product.Get(result.Product.ID, &stripego.ProductParams{})
	if err != nil {
		logger.Error("Error getting product", "error", err)
	}
	result.Product = product

	priceMapMu.Lock()
	priceMap[priceId] = result
	priceMapMu.Unlock()
	return result
}

func GetPriceName(priceId string) string {
	price := GetPrice(priceId)
	return price.Product.Name
}

func GetPriceCost(priceId string) float64 {
	price := GetPrice(priceId)
	return float64(price.UnitAmount) / 100
}
