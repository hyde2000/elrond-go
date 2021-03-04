package stakeValuesProcessor

import "errors"

// ErrCannotCastAccountHandlerToUserAccount signal that returned account is wrong
var ErrCannotCastAccountHandlerToUserAccount = errors.New("cannot cast AccountHandler to UserAccount")

// ErrNilMarshalizer signals that a nil marshalizer has been provided
var ErrNilMarshalizer = errors.New("trying to set nil marshalizer")

// ErrNilAccountsAdapter signals that a nil accounts adapter has been provided
var ErrNilAccountsAdapter = errors.New("trying to set nil accounts adapter")

// ErrInvalidNodePrice signals that an invalid node price has been provided
var ErrInvalidNodePrice = errors.New("invalid node price")
