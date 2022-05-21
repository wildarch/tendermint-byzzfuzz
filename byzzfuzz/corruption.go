package byzzfuzz

import (
	"bytes"
	"log"
	"math/rand"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

func actionForCorruption(corruptionType CorruptionType) testlib.Action {
	switch corruptionType {
	case ChangeProposalToNil:
		return changeProposalToNil
	case ChangeVoteToNil:
		return common.ChangeVoteToNil()
	case ChangeVoteRound:
		return changeVoteRound()
	default:
		panic("Invalid type of corruption")
	}
}

func corruptMessage(random *rand.Rand) testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}

		// Debugging
		from := -1
		to := -1
		for i, r := range c.Replicas.Iter() {
			if r.ID == tMsg.From {
				from = i
			}
			if r.ID == tMsg.To {
				to = i
			}
		}
		totalRounds, ok := c.Vars.GetInt(totalRoundsKey(e.Replica))
		if !ok {
			totalRounds = 0
		}
		log.Printf("Corrupting message (from=%d, to=%d, round=%d)", from, to, totalRounds)

		switch tMsg.Type {
		case util.Proposal:
			// TODO:
			// - ChangeProposalLockValue (TODO?)
			actions := [1]testlib.Action{
				changeProposalToNil,
			}
			return actions[random.Intn(len(actions))](e, c)
		case util.Precommit:
			fallthrough
		case util.Prevote:
			// TODO:
			// - ChangeVoteToProposalMessage (see relocked)
			// - ChangeVoteTime
			actions := [2]testlib.Action{
				common.ChangeVoteToNil(),
				changeVoteRound(),
			}
			return actions[random.Intn(len(actions))](e, c)
		default:
			return []*types.Message{m}
		}
	}
}

func changeVoteRound() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}
		if tMsg.Type != util.Precommit && tMsg.Type != util.Prevote {
			return []*types.Message{}
		}
		valAddr, ok := util.GetVoteValidator(tMsg)
		if !ok {
			return []*types.Message{}
		}
		var replica *types.Replica = nil
		for _, r := range c.Replicas.Iter() {
			addr, err := util.GetReplicaAddress(r)
			if err != nil {
				continue
			}
			if bytes.Equal(addr, valAddr) {
				replica = r
				break
			}
		}
		if replica == nil {
			return []*types.Message{}
		}
		newVote, err := util.ChangeVoteRound(replica, tMsg, int32(tMsg.Round()+2))
		if err != nil {
			return []*types.Message{}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{}
		}
		return []*types.Message{c.NewMessage(m, msgB)}
	}
}

func changeProposalToNil(e *types.Event, c *testlib.Context) []*types.Message {
	message, _ := c.GetMessage(e)
	tMsg, ok := util.GetParsedMessage(message)
	if !ok {
		return []*types.Message{}
	}
	replica, _ := c.Replicas.Get(tMsg.From)
	newProp, err := util.ChangeProposalBlockIDToNil(replica, tMsg)
	if err != nil {
		//c.Logger().With(log.LogParams{"error": err}).Error("Failed to change proposal")
		return []*types.Message{message}
	}
	newMsgB, err := newProp.Marshal()
	if err != nil {
		//c.Logger().With(log.LogParams{"error": err}).Error("Failed to marshal changed proposal")
		return []*types.Message{message}
	}
	return []*types.Message{c.NewMessage(message, newMsgB)}
}
