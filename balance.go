package reghelp

import "context"

// GetBalance returns the current account balance.
func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	raw, err := c.do(ctx, "/balance", nil, "", false)
	if err != nil {
		return nil, err
	}
	out := &BalanceResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
