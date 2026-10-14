package orders

import "errors"

var ErrNotFound = errors.New("order not found")

type Order struct {
	ID    string `json:"id"`
	Total int64  `json:"total"`
	Code  string `json:"code"`
}

func (o Order) Validate() error {
	if o.Total <= 0 {
		return errors.New("total must be positive")
	}
	return nil
}
