package spec

import (
	"byzzfuzz/liveness"
	"strconv"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

// Track height and round of received message
// Track current height/round

type StepEvent struct {
	Replica string
	Height  int
	Round   int
}

func (e *StepEvent) IsStep() bool    { return true }
func (e *StepEvent) IsMessage() bool { return false }

type MessageEvent struct {
	From   string
	To     string
	Height int
	Round  int
}

func (e *MessageEvent) IsStep() bool    { return false }
func (e *MessageEvent) IsMessage() bool { return true }

type Event interface {
	IsStep() bool
	IsMessage() bool
}

func Log(ch chan Event) testlib.FilterFunc {
	return func(e *types.Event, ctx *testlib.Context) (ms []*types.Message, handled bool) {
		if liveness.IsTestFinished(e, ctx) {
			return
		}
		// Handle message
		if testlib.IsMessageReceive()(e, ctx) {
			message, ok := util.GetMessageFromEvent(e, ctx)
			if ok {
				height, round := message.HeightRound()
				if round >= 0 && (message.Type == util.Prevote || message.Type == util.Precommit) {
					ch <- &MessageEvent{
						From:   getPartLabel(ctx, message.From),
						To:     getPartLabel(ctx, message.To),
						Height: height,
						Round:  round,
					}
				}
			}
		}
		// Handle step
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
		ch <- &StepEvent{
			Replica: getPartLabel(ctx, e.Replica),
			Height:  height,
			Round:   round,
		}
		return
	}
}
