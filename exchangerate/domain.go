package exchangerate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes exchangerate as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/exchangerate-cli/exchangerate"
//
// The same Domain also builds the standalone exchangerate binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the exchangerate driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "exchangerate",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "exchangerate",
			Short:  "Exchange rate lookup and currency conversion",
			Long: `exchangerate fetches live currency exchange rates from open.er-api.com.
No API key required. 166 currencies supported.`,
			Site: Host,
			Repo: "https://github.com/tamnd/exchangerate-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// rates: list exchange rates for a base currency
	kit.Handle(app, kit.OpMeta{
		Name:    "rates",
		Group:   "read",
		List:    true,
		Summary: "List exchange rates for a base currency",
		Args: []kit.Arg{
			{Name: "base", Help: "base currency code (e.g. USD, EUR, GBP)"},
		},
	}, latestOp)

	// convert: convert an amount between currencies
	kit.Handle(app, kit.OpMeta{
		Name:    "convert",
		Group:   "read",
		Single:  true,
		Summary: "Convert an amount from one currency to another",
		Args: []kit.Arg{
			{Name: "amount", Help: "amount to convert"},
			{Name: "from", Help: "source currency (ISO 4217)"},
			{Name: "to", Help: "target currency (ISO 4217)"},
		},
	}, convertOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type latestInput struct {
	Base   string        `kit:"arg" help:"base currency code (e.g. USD, EUR, GBP)"`
	Limit  int           `kit:"flag,inherit" help:"max results (0 = all)"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type convertInput struct {
	Amount string  `kit:"arg" help:"amount to convert"`
	From   string  `kit:"arg" help:"source currency (ISO 4217)"`
	To     string  `kit:"arg" help:"target currency (ISO 4217)"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func latestOp(ctx context.Context, in latestInput, emit func(Rate) error) error {
	base := in.Base
	if base == "" {
		base = "USD"
	}
	items, err := in.Client.Latest(ctx, base, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func convertOp(ctx context.Context, in convertInput, emit func(*Conversion) error) error {
	amount, err := strconv.ParseFloat(in.Amount, 64)
	if err != nil {
		return errs.Usage("invalid amount %q: %v", in.Amount, err)
	}
	conv, err := in.Client.Convert(ctx, strings.ToUpper(in.From), strings.ToUpper(in.To), amount)
	if err != nil {
		return mapErr(err)
	}
	return emit(conv)
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty exchangerate reference")
	}
	return "rate", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "rate":
		return fmt.Sprintf("https://open.er-api.com/v6/latest/%s", strings.ToUpper(id)), nil
	default:
		return "", errs.Usage("exchangerate has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
