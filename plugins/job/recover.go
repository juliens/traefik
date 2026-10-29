package job

import (
	"fmt"
	"runtime/debug"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
)

// OperationWithRecover wrap a backoff operation in a Recover.
func OperationWithRecover(operation backoff.Operation) backoff.Operation {
	return func() (err error) {
		defer func() {
			if res := recover(); res != nil {
				log.Error().Interface("error", res).Msg("Error in Go routine")
				log.Error().Msgf("Stack: %s", debug.Stack())
				err = fmt.Errorf("panic in operation: %w", err)
			}
		}()
		return operation()
	}
}
