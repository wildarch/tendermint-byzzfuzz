package spec

import (
	"fmt"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

// Keeps track of the highest round number (and associated height) received from peer nodes.
// This is used to check that nodes move to a higher round of they received f+1 messages with a higher round number.
func RecordHighestRoundNumberReceived(e *types.Event, ctx *testlib.Context) (messages []*types.Message, handled bool) {
	if !testlib.IsMessageReceive()(e, ctx) {
		return
	}
	message, ok := util.GetMessageFromEvent(e, ctx)
	if !ok {
		return
	}
	height, round := message.HeightRound()
	if round < 0 {
		return
	}
	hr := heightRound{height: height, round: round}
	hrrR, found := ctx.Vars.Get(highestRoundReceivedKey(e.Replica))
	if !found {
		hrrR = highestRoundReceived{from: make(map[types.ReplicaID]heightRound)}
	}
	hrr := hrrR.(highestRoundReceived)
	if hrr.Update(message.From, hr) {
		ctx.Logger().Debug(fmt.Sprintf("Update highest round received on %s from %s to (H=%d/R=%d)",
			e.Replica, message.From, height, round))

		ctx.Vars.Set(highestRoundReceivedKey(message.To), hrr)
	}

	return
}

type heightRound struct {
	height int
	round  int
}

type highestRoundReceived struct {
	from map[types.ReplicaID]heightRound
}

func (hrr *highestRoundReceived) Update(from types.ReplicaID, new heightRound) (updated bool) {
	hr, found := hrr.from[from]
	if !found || hr.height < new.height || (hr.height == new.height && hr.round < new.round) {
		hrr.from[from] = new
		return true
	}
	return
}

func highestRoundReceivedKey(id types.ReplicaID) string {
	return fmt.Sprintf("BF_highest_round_received_%s", id)
}
