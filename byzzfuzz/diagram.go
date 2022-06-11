package byzzfuzz

import (
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

func logConsensusMessages(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	/*
		message, ok := c.GetMessage(e)
		if !ok {
			return
		}
		tMessage, ok := util.GetParsedMessage(message)
		if !ok {
			return
		}
	*/
	message, ok := util.GetMessageFromEvent(e, c)
	if !ok {
		return
	}

	// Only log consensus message types
	switch message.Type {
	case util.Proposal:
	case util.Prevote:
	case util.Precommit:
	default:
		return
	}

	from := getPartLabel(c, message.From)
	to := getPartLabel(c, message.To)

	c.Logger().With(log.LogParams{
		"is_receive": e.IsMessageReceive(),
		"is_send":    e.IsMessageSend(),
		"sent_from":  from,
		"sent_to":    to,
		"type":       message.Type,
		"height":     message.Height(),
		"round":      message.Round(),
	}).Info("Consensus message")

	return
}
