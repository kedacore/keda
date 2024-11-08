package contextkeys

import "context"

// contextKeys gets and sets context values from a context.
type contextKeys struct {
	accountID struct{}
}

var (
	keys = contextKeys{
		accountID: struct{}{},
	}
)

// SetAccountID inserts the account ID value into context.
func SetAccountID(ctx context.Context, value string) context.Context {
	return keys.SetAccountID(ctx, value)
}

// GetAccountID returns the account ID from the context.
func GetAccountID(ctx context.Context) (string, bool) {
	return keys.GetAccountID(ctx)
}

// SetAccountID inserts the account ID value into context.
func (c contextKeys) SetAccountID(ctx context.Context, value string) context.Context {
	if nil == ctx {
		ctx = context.Background()
	}

	return context.WithValue(ctx, c.accountID, value)
}

// GetAccountID returns the account ID from the context.
func (c contextKeys) GetAccountID(ctx context.Context) (string, bool) {
	if nil != ctx {
		accountID, ok := ctx.Value(keys.accountID).(string)
		return accountID, ok
	}

	return "", false
}
