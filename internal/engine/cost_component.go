package engine

import (
	"github.com/shopspring/decimal"
)

type LineItem struct {
	Name           string
	Unit           string
	UnitMultiplier decimal.Decimal
	// UnitRounding specifies the number of decimal places that the output unit should be rounded to.
	// This should be set to 0 if using MonthToHourUnitMultiplier otherwise the unit will show with
	// redundant .000 decimal places.
	UnitRounding         *int32
	IgnoreIfMissingPrice bool
	ProductFilter        *ProductSelector
	PriceFilter          *RateSelector
	HourlyQuantity       *decimal.Decimal
	MonthlyQuantity      *decimal.Decimal
	MonthlyDiscountPerc  float64
	price                decimal.Decimal
	customPrice          *decimal.Decimal
	priceHash            string
	HourlyCost           *decimal.Decimal
	MonthlyCost          *decimal.Decimal
	UsageBased           bool
	PriceNotFound        bool
}

func (c *LineItem) CalculateCosts() {
	c.fillQuantities()
	if c.HourlyQuantity != nil {
		c.HourlyCost = decimalPtr(c.price.Mul(*c.HourlyQuantity))
	}
	if c.MonthlyQuantity != nil {
		discountMul := decimal.NewFromFloat(1.0 - c.MonthlyDiscountPerc)
		c.MonthlyCost = decimalPtr(c.price.Mul(*c.MonthlyQuantity).Mul(discountMul))
	}
}

func (c *LineItem) fillQuantities() {
	if c.MonthlyQuantity != nil && c.HourlyQuantity == nil {
		c.HourlyQuantity = decimalPtr(c.MonthlyQuantity.Div(HourToMonthUnitMultiplier))
	} else if c.HourlyQuantity != nil && c.MonthlyQuantity == nil {
		c.MonthlyQuantity = decimalPtr(c.HourlyQuantity.Mul(HourToMonthUnitMultiplier))
	}
}

func (c *LineItem) SetPrice(price decimal.Decimal) {
	c.price = price
}

// SetPriceNotFound zeros the price and marks the component as having a price not
// found.
func (c *LineItem) SetPriceNotFound() {
	c.price = decimal.Zero
	c.PriceNotFound = true
}

func (c *LineItem) Price() decimal.Decimal {
	return c.price
}

func (c *LineItem) SetPriceHash(priceHash string) {
	c.priceHash = priceHash
}

func (c *LineItem) PriceHash() string {
	return c.priceHash
}

func (c *LineItem) SetCustomPrice(price *decimal.Decimal) {
	c.customPrice = price
}

func (c *LineItem) CustomPrice() *decimal.Decimal {
	return c.customPrice
}

func (c *LineItem) UnitMultiplierPrice() decimal.Decimal {
	// Round the final number to 16 decimal places to avoid floating point issues.
	return c.Price().Mul(c.UnitMultiplier)
}

func (c *LineItem) UnitMultiplierHourlyQuantity() *decimal.Decimal {
	if c.HourlyQuantity == nil {
		return nil
	}

	var m decimal.Decimal

	if c.UnitMultiplier.IsZero() {
		m = decimal.Zero
	} else {
		// Round the final number to 16 decimal places to avoid floating point issues.
		m = c.HourlyQuantity.Div(c.UnitMultiplier)
		if c.UnitRounding != nil {
			m = m.Round(*c.UnitRounding)
		}
	}

	return &m
}

func (c *LineItem) UnitMultiplierMonthlyQuantity() *decimal.Decimal {
	if c.MonthlyQuantity == nil {
		return nil
	}

	var m decimal.Decimal

	if c.UnitMultiplier.IsZero() {
		m = decimal.Zero
	} else {
		// Round the final number to 16 decimal places to avoid floating point issues.
		m = c.MonthlyQuantity.Div(c.UnitMultiplier)
		if c.UnitRounding != nil {
			m = m.Round(*c.UnitRounding)
		}
	}

	return &m
}
