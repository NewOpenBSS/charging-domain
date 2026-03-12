package quota

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCurrentTaxPercent(t *testing.T) {
	expected := decimal.NewFromFloat(0.15)
	assert.True(t, expected.Equal(CurrentTaxPercent()))
}

func TestTaxCalculation_TotalAmount(t *testing.T) {
	tc := TaxCalculation{
		TaxAmount:  decimal.NewFromFloat(1.5),
		ExTaxValue: decimal.NewFromFloat(10.0),
	}
	expected := decimal.NewFromFloat(11.5)
	assert.True(t, expected.Equal(tc.TotalAmount()))
}

func TestCalculateTaxFromInclusiveAmount(t *testing.T) {
	amount := decimal.NewFromFloat(115.0)
	taxRate := decimal.NewFromFloat(0.15)

	// Note: The implementation in taxcalculator.go seems to calculate taxAmount = amount * taxRate
	// which is not the standard way to calculate tax from an *inclusive* amount.
	// Usually, for inclusive amount: taxAmount = amount - (amount / (1 + taxRate))
	// However, I should follow the current implementation's logic.
	
	tc := CalculateTaxFromInclusiveAmount(amount, taxRate)

	expectedTaxAmount := decimal.NewFromFloat(17.25) // 115 * 0.15
	expectedExTaxValue := decimal.NewFromFloat(97.75) // 115 - 17.25

	assert.True(t, expectedTaxAmount.Equal(tc.TaxAmount), "TaxAmount should be 17.25")
	assert.True(t, expectedExTaxValue.Equal(tc.ExTaxValue), "ExTaxValue should be 97.75")
	assert.True(t, taxRate.Equal(tc.TaxRate))
}

func TestCalculateDefaultTaxFromInclusiveAmount(t *testing.T) {
	amount := decimal.NewFromFloat(100.0)
	tc := CalculateDefaultTaxFromInclusiveAmount(amount)

	expectedTaxAmount := decimal.NewFromFloat(15.0) // 100 * 0.15
	expectedExTaxValue := decimal.NewFromFloat(85.0)  // 100 - 15.0

	assert.True(t, expectedTaxAmount.Equal(tc.TaxAmount))
	assert.True(t, expectedExTaxValue.Equal(tc.ExTaxValue))
	assert.True(t, CurrentTaxPercent().Equal(tc.TaxRate))
}

func TestCalculateTax(t *testing.T) {
	exTaxValue := decimal.NewFromFloat(100.0)
	taxRate := decimal.NewFromFloat(0.15)

	tc := CalculateTax(exTaxValue, taxRate)

	expectedTaxAmount := decimal.NewFromFloat(15.0)
	assert.True(t, expectedTaxAmount.Equal(tc.TaxAmount))
	assert.True(t, exTaxValue.Equal(tc.ExTaxValue))
	assert.True(t, taxRate.Equal(tc.TaxRate))
}

func TestCalculateDefaultTax(t *testing.T) {
	exTaxValue := decimal.NewFromFloat(200.0)
	tc := CalculateDefaultTax(exTaxValue)

	expectedTaxAmount := decimal.NewFromFloat(30.0) // 200 * 0.15
	assert.True(t, expectedTaxAmount.Equal(tc.TaxAmount))
	assert.True(t, exTaxValue.Equal(tc.ExTaxValue))
	assert.True(t, CurrentTaxPercent().Equal(tc.TaxRate))
}
