package quota

type ReasonCode string

const (
	// A quota provisioned for a subscriber.
	ReasonQuotaProvisioned ReasonCode = "QUOTA_PROVISIONED"

	// Standard service usage (voice, data, SMS, USSD).
	ReasonServiceUsage ReasonCode = "SERVICE_USAGE"

	// A conversion between units MONETARY and SERVICE Unit types.
	ReasonConversion ReasonCode = "CONVERSION"

	// A quota transfer in from one connection to another.
	ReasonTransferIn ReasonCode = "TRANSFER_IN"

	// A quota transfer out from one connection to another.
	ReasonTransferOut ReasonCode = "TRANSFER_OUT"

	// Used to repay an existing loan from newly acquired quota.
	ReasonLoanRepayment ReasonCode = "LOAN_REPAYMENT"

	// A transaction fee.
	ReasonTransactionFee ReasonCode = "TRANSACTION_FEE"

	// A quota expiry.
	ReasonQuotaExpiry ReasonCode = "QUOTA_EXPIRY"
)
