package byzzfuzz

import (
	"byzzfuzz/liveness"
	"fmt"
	"strconv"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

func trackTotalRounds(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	// Parse event to retrieve height and round
	eType, ok := e.Type.(*types.GenericEventType)
	if !ok {
		return
	}
	if eType.T != "newStep" {
		return
	}
	heightS, ok := eType.Params["height"]
	if !ok {
		return
	}
	height, err := strconv.Atoi(heightS)
	if err != nil {
		return
	}
	roundS, ok := eType.Params["round"]
	if !ok {
		return
	}
	round, err := strconv.Atoi(roundS)
	if err != nil {
		return
	}

	// Retrieve previous values
	prevHeight, ok := c.Vars.GetInt(prevHeightKey(e.Replica))
	if !ok {
		prevHeight = 1
	}
	prevRound, ok := c.Vars.GetInt(prevRoundKey(e.Replica))
	if !ok {
		prevRound = 0
	}
	totalRounds, ok := c.Vars.GetInt(totalRoundsKey(e.Replica))
	if !ok {
		totalRounds = 0
	}
	oldTotalRounds := totalRounds

	if height > prevHeight {
		// We have changed to a new height (+1), and possible skipped over some rounds (round)
		totalRounds += 1 + round
	} else if round > prevRound {
		// We have moved to a higher round
		totalRounds += round - prevRound
	}

	if totalRounds != oldTotalRounds {
		c.Logger().With(log.LogParams{
			"totalRounds": totalRounds,
			"height":      height,
			"round":       round,
		}).Debug("Updated total rounds")
	}

	c.Vars.Set(prevHeightKey(e.Replica), height)
	c.Vars.Set(prevRoundKey(e.Replica), round)
	c.Vars.Set(totalRoundsKey(e.Replica), totalRounds)
	c.Vars.Set(totalRoundForHeightRoundKey(e.Replica, height, round), totalRounds)

	return
}

func isMessageFromTotalRound(round int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		if liveness.IsTestFinished(e, c) {
			return false
		}
		if !testlib.IsMessageSend()(e, c) {
			panic("isMessageFromTotalRound uses the round as perceived by the sender, " +
				"thus must be used together with isMessageSend")
		}
		message, ok := util.GetMessageFromEvent(e, c)
		if !ok {
			panic("Message not found!")
		}
		if message.Round() == -1 {
			return false
		}

		totalRounds, ok := c.Vars.GetInt(totalRoundsKey(e.Replica))
		if !ok {
			// This can happen if the node is byzantine and produces a message for an invalid round
			return false
		}
		return totalRounds == round
	}
}

func isMessageOfTotalRound(round int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		if liveness.IsTestFinished(e, c) {
			return false
		}
		if !testlib.IsMessageSend()(e, c) {
			panic("isMessageOfTotalRound uses the round as perceived by the sender, " +
				"thus must be used together with isMessageSend")
		}
		message, ok := util.GetMessageFromEvent(e, c)
		if !ok {
			panic("Message not found!")
		}
		if message.Round() == -1 {
			return false
		}

		totalRounds, ok := c.Vars.GetInt(
			totalRoundForHeightRoundKey(e.Replica, message.Height(), message.Round()))
		if !ok {
			// This can happen if the node is byzantine and produces a message for an invalid round
			return false
		}
		return totalRounds == round
	}
}

func prevHeightKey(r types.ReplicaID) string {
	return fmt.Sprintf("BF_prev_height_%s", r)
}

func prevRoundKey(r types.ReplicaID) string {
	return fmt.Sprintf("BF_prev_round_%s", r)
}

func totalRoundsKey(r types.ReplicaID) string {
	return fmt.Sprintf("BF_total_rounds_%s", r)
}

func totalRoundForHeightRoundKey(r types.ReplicaID, height int, round int) string {
	return fmt.Sprintf("BF_total_round_%s_%d_%d", r, height, round)
}
