package middleware

import (
	"github.com/taills/ai-gateway/model"
)

// lookupToken is a thin wrapper so the middleware package can access the model
// package without a circular import.
func lookupToken(key string) (*model.Token, error) {
	return model.GetTokenByKey(key)
}
