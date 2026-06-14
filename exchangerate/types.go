package exchangerate

// Rate is one currency's exchange rate against a base currency.
type Rate struct {
	Rank     int     `json:"rank"`
	Currency string  `kit:"id" json:"currency"` // ISO 4217 code, e.g. "EUR"
	Rate     float64 `json:"rate"`               // units of Currency per 1 unit of base
	Base     string  `json:"base"`               // the base currency, e.g. "USD"
}

// Conversion is the result of converting an amount from one currency to another.
type Conversion struct {
	From      string  `kit:"id" json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Result    float64 `json:"result"`
	Rate      float64 `json:"rate"`
	UpdatedAt string  `json:"updated_at"` // time_last_update_utc verbatim
}

// latestResponse is the JSON envelope from GET /v6/latest/{base}.
type latestResponse struct {
	Result    string             `json:"result"`
	BaseCode  string             `json:"base_code"`
	UpdatedAt string             `json:"time_last_update_utc"`
	Rates     map[string]float64 `json:"rates"`
}
