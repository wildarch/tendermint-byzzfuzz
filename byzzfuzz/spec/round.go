package spec

import (
	"fmt"

	"github.com/netrixframework/netrix/log"
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
		ctx.Logger().With(log.LogParams{
			"replica": e.Replica,
			"from":    message.From,
			"height":  height,
			"round":   round,
		}).Info("Update highest round")

		ctx.Vars.Set(highestRoundReceivedKey(e.Replica), hrr)
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

func (hrr *highestRoundReceived) Update(from types.ReplicaID, new heightRound) bool {
	hr, found := hrr.from[from]
	if !found || hr.height < new.height || (hr.height == new.height && hr.round < new.round) {
		hrr.from[from] = new
		return true
	}
	return false
}

func (hrr *highestRoundReceived) HasHigherMajority(faults int, height int, round int) bool {
	for _, hr := range hrr.from {
		if hr.height > height || (hr.height == height && hr.round > round) {
			if hrr.hasMajority(faults, hr.height, hr.round) {
				return true
			}
		}
	}

	return false
}

func (hrr *highestRoundReceived) hasMajority(faults int, height int, round int) bool {
	count := 0
	for _, hr := range hrr.from {
		if hr.height == height && hr.round == round {
			count++
		}
	}

	return count >= (faults + 1)
}

func highestRoundReceivedKey(id types.ReplicaID) string {
	return fmt.Sprintf("BF_highest_round_received_%s", id)
}

func SendsMessageWithTooLowRound(e *types.Event, c *testlib.Context) bool {
	if !testlib.IsMessageSend()(e, c) {
		return false
	}
	message, ok := util.GetMessageFromEvent(e, c)
	if !ok {
		return false
	}
	height, round := message.HeightRound()
	if round < 0 {
		return false
	}
	hrrR, found := c.Vars.Get(highestRoundReceivedKey(e.Replica))
	if !found {
		return false
	}
	hrr := hrrR.(highestRoundReceived)

	faults, ok := c.Vars.GetInt("faults")
	if !ok {
		panic("Number of faulty nodes not saved in vars")
	}

	c.Logger().With(log.LogParams{
		"highest_round_received": fmt.Sprint(hrr),
		"height":                 height,
		"round":                  round,
	}).Info("checking message round is not too low")

	return hrr.HasHigherMajority(faults, height, round)
}
