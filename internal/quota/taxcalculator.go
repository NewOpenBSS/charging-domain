package quota

import "github.com/shopspring/decimal"

type TaxCalculation struct {
	TaxAmount  decimal.Decimal
	ExTaxValue decimal.Decimal
	TaxRate    decimal.Decimal
}

func (t TaxCalculation) TotalAmount() decimal.Decimal {
	return t.TaxAmount.Add(t.ExTaxValue)
}

func CurrentTaxPercent() decimal.Decimal {
	return decimal.NewFromFloat(0.15)
}

func CalculateTaxFromInclusiveAmount(amount decimal.Decimal, taxRate decimal.Decimal) TaxCalculation {
	taxAmount := amount.Mul(taxRate)
	exTaxValue := amount.Sub(taxAmount)

	return TaxCalculation{
		TaxAmount:  taxAmount,
		ExTaxValue: exTaxValue,
		TaxRate:    taxRate,
	}
}

func CalculateDefaultTaxFromInclusiveAmount(amount decimal.Decimal) TaxCalculation {
	return CalculateTaxFromInclusiveAmount(amount, CurrentTaxPercent())
}

func CalculateTax(exTaxValue decimal.Decimal, taxRate decimal.Decimal) TaxCalculation {

	taxAmount := exTaxValue.Mul(taxRate)

	return TaxCalculation{
		TaxAmount:  taxAmount,
		ExTaxValue: exTaxValue,
		TaxRate:    taxRate,
	}
}

func CalculateDefaultTax(exTaxValue decimal.Decimal) TaxCalculation {
	return CalculateTax(exTaxValue, CurrentTaxPercent())
}
