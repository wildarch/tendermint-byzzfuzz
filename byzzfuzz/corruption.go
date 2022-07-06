package byzzfuzz

import (
	"bytes"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
	ttypes "github.com/tendermint/tendermint/types"
)

func (c *MessageCorruption) Action() testlib.Action {
	switch c.Corruption {
	case ChangeProposalToNil:
		return changeProposalToNil
	case ChangeVoteToNil:
		return changeVoteToNil
	case ChangeVoteRound:
		return changeVoteRound(c.Seed)
	case Omit:
		return omitMessage
	case ChangeBlockId:
		return changeBlockId(c.Seed)
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
	//fmt.Printf("ChangeVoteToNil H=%d/R=%d %s %s -> %s\n", tMsg.Height(), tMsg.Round(), message.Type, message.From, message.To)
	c.Logger().With(log.LogParams{
		"height": tMsg.Height(),
		"round":  tMsg.Round(),
		"from":   getPartLabel(c, e.Replica),
		"to":     getPartLabel(c, tMsg.To),
		"type":   "ChangeVoteToNil",
	}).Info("Corruption")
	return []*types.Message{c.NewMessage(message, msgB)}
}

func changeVoteRound(seed int) testlib.Action {
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
		newVote, err := util.ChangeVoteRound(replica, tMsg, int32(seed))
		if err != nil {
			return []*types.Message{m}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{m}
		}
		c.Logger().With(log.LogParams{
			"height": tMsg.Height(),
			"round":  tMsg.Round(),
			"from":   getPartLabel(c, e.Replica),
			"to":     getPartLabel(c, tMsg.To),
			"type":   "ChangeVoteRound",
		}).Info("Corruption")
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
	c.Logger().With(log.LogParams{
		"height": tMsg.Height(),
		"round":  tMsg.Round(),
		"from":   getPartLabel(c, e.Replica),
		"to":     getPartLabel(c, tMsg.To),
		"type":   "ChangeProposalToNil",
	}).Info("Corruption")
	return []*types.Message{c.NewMessage(message, newMsgB)}
}

func omitMessage(e *types.Event, c *testlib.Context) []*types.Message {
	message, _ := c.GetMessage(e)
	tMsg, ok := util.GetParsedMessage(message)
	if !ok {
		return []*types.Message{}
	}
	c.Logger().With(log.LogParams{
		"height": tMsg.Height(),
		"round":  tMsg.Round(),
		"from":   getPartLabel(c, e.Replica),
		"to":     getPartLabel(c, tMsg.To),
		"type":   "Omit",
	}).Info("Corruption")
	return []*types.Message{}
}

func changeBlockId(seed int) testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		c.Logger().Info("Attempt to change block id")
		blockIdsR, ok := c.Vars.Get("BF_blockids")
		if !ok {
			blockIdsR = make([]*ttypes.BlockID, 0)
		}
		blockIds := blockIdsR.([]*ttypes.BlockID)
		newBlockId := blockIds[seed%len(blockIds)]

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
		newVote, err := util.ChangeVote(replica, tMsg, newBlockId)
		if err != nil {
			return []*types.Message{m}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{m}
		}
		c.Logger().With(log.LogParams{
			"height":   tMsg.Height(),
			"round":    tMsg.Round(),
			"from":     getPartLabel(c, e.Replica),
			"to":       getPartLabel(c, tMsg.To),
			"type":     "ChangeBlockId",
			"block_id": newBlockId,
		}).Info("Corruption")
		return []*types.Message{c.NewMessage(m, msgB)}
	}
}

func logBlockIds(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	message, ok := util.GetMessageFromEvent(e, c)
	if !ok {
		return
	}

	blockId, ok := util.GetProposalBlockID(message)
	if !ok {
		return
	}

	// Append the block id
	blockIdsR, ok := c.Vars.Get("BF_blockids")
	if !ok {
		blockIdsR = make([]*ttypes.BlockID, 0)
	}
	blockIds := blockIdsR.([]*ttypes.BlockID)
	blockIds = append(blockIds, blockId)
	c.Vars.Set("BF_blockids", blockIds)

	c.Logger().With(log.LogParams{
		"block_id": nil,
	}).Info("blockID")

	return
}
