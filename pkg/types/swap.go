package types

// SwapRequest represents a user's swap command
type SwapRequest struct {
	Amount          string
	SourceToken     string
	DestToken       string
	SourceChain     string
	DestChain       string
	RecipientAddr   string
	RefundAddr      string
}

// QuoteDisplay holds formatted quote information for display
type QuoteDisplay struct {
	SourceAmount    string
	SourceToken     string
	DestAmount      string
	DestToken       string
	Rate            string
	Fee             string
	EstimatedTime   string
	DepositAddress  string
	IntentID        string
}

// SwapStatus represents the current status of a swap
type SwapStatus struct {
	IntentID    string
	Status      string
	Message     string
	TxHash      string
	Timestamp   string
}
