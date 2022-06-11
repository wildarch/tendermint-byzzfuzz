package liveness

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
)

// Set to true once the test is done and we only check liveness
const testFinishedKey = "BF_test_finished"

const ExtraTimeout = 60 * time.Second

func SetupLivenessTimer(timeout time.Duration) common.SetupOption {
	return func(ctx *testlib.Context) {
		go func() {
			ctx.Logger().Info("Waiting for timeout to expire")
			time.Sleep(timeout)
			ctx.Logger().Info("Test finished, checking liveness")
			ctx.Vars.Set(testFinishedKey, true)
		}()
	}
}

func IsTestFinished(e *types.Event, ctx *testlib.Context) bool {
	r, ok := ctx.Vars.GetBool(testFinishedKey)
	if !ok {
		return false
	}
	return r
}
