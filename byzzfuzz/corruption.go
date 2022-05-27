package byzzfuzz

import (
	"bytes"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

func actionForCorruption(corruptionType CorruptionType) testlib.Action {
	switch corruptionType {
	case ChangeProposalToNil:
		return changeProposalToNil
	case ChangeVoteToNil:
		return changeVoteToNil
	case ChangeVoteRound:
		return changeVoteRound
	default:
		panic("Invalid type of corruption")
	}
}

func changeVoteToNil(e *types.Event, c *testlib.Context) []*types.Message {
	message, ok := c.GetMessage(e)
	if !ok {
		return []*types.Message{}
	}
	tMsg, ok := util.GetParsedMessage(message)
	if !ok {
		return []*types.Message{message}
	}
	if tMsg.Type != util.Precommit && tMsg.Type != util.Prevote {
		return []*types.Message{message}
	}
	valAddr, ok := util.GetVoteValidator(tMsg)
	if !ok {
		return []*types.Message{message}
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
		return []*types.Message{message}
	}
	newVote, err := util.ChangeVoteToNil(replica, tMsg)
	if err != nil {
		return []*types.Message{message}
	}
	msgB, err := newVote.Marshal()
	if err != nil {
		return []*types.Message{message}
	}
	return []*types.Message{c.NewMessage(message, msgB)}
}

func changeVoteRound(e *types.Event, c *testlib.Context) []*types.Message {
	m, ok := c.GetMessage(e)
	if !ok {
		return []*types.Message{}
	}
	tMsg, ok := util.GetParsedMessage(m)
	if !ok {
		return []*types.Message{m}
	}
	if tMsg.Type != util.Precommit && tMsg.Type != util.Prevote {
		return []*types.Message{m}
	}
	valAddr, ok := util.GetVoteValidator(tMsg)
	if !ok {
		return []*types.Message{m}
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
		return []*types.Message{m}
	}
	newVote, err := util.ChangeVoteRound(replica, tMsg, int32(tMsg.Round()+2))
	if err != nil {
		return []*types.Message{m}
	}
	msgB, err := newVote.Marshal()
	if err != nil {
		return []*types.Message{m}
	}
	return []*types.Message{c.NewMessage(m, msgB)}
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
