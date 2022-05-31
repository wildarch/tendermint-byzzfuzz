package spec

import (
	"fmt"
	"strconv"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/util"
)

type heightRound struct {
	height int
	round  int
}

type nodeSet struct {
	nodes []types.ReplicaID
}

func newNodeSet() nodeSet {
	return nodeSet{
		nodes: make([]types.ReplicaID, 0),
	}
}

func (s *nodeSet) Add(r types.ReplicaID) {
	found := false
	for _, v := range s.nodes {
		if v == r {
			found = true
			break
		}
	}
	if !found {
		s.nodes = append(s.nodes, r)
	}
}

type roundsReceived struct {
	received map[heightRound]nodeSet
}

func (rr *roundsReceived) Update(from types.ReplicaID, hr heightRound) {
	set, found := rr.received[hr]
	if !found {
		set = newNodeSet()
	}
	set.Add(from)
	rr.received[hr] = set
}

type expectedSteps struct {
	steps []heightRound
}

func (es *expectedSteps) Add(step heightRound) {
	found := false
	for _, v := range es.steps {
		if v == step {
			found = true
			break
		}
	}

	if !found {
		es.steps = append(es.steps, step)
	}
}

func (es *expectedSteps) Remove(step heightRound) {
	index := -1
	for i, v := range es.steps {
		if v == step {
			index = i
			break
		}
	}

	if index != -1 {
		es.steps = append(es.steps[:index], es.steps[index+1:]...)
	}
}

func TrackRoundsReceived(e *types.Event, ctx *testlib.Context) (messages []*types.Message, handled bool) {
	if !testlib.IsMessageReceive()(e, ctx) {
		return
	}
	message, ok := util.GetMessageFromEvent(e, ctx)
	if !ok {
		return
	}
	height, round := message.HeightRound()
	if round < 0 || (height == 1 && round == 0) {
		return
	}
	hr := heightRound{height: height, round: round}

	// Update the counts
	rrR, found := ctx.Vars.Get(roundsReceivedKey(e.Replica))
	if !found {
		rrR = roundsReceived{received: make(map[heightRound]nodeSet)}
	}
	rr := rrR.(roundsReceived)
	rr.Update(message.From, hr)
	//fmt.Printf("!! recv,%s,%s,%d,%d\n", getPartLabel(ctx, message.From), getPartLabel(ctx, e.Replica), hr.height, hr.round)
	ctx.Vars.Set(roundsReceivedKey(e.Replica), rr)

	currentHrR, found := ctx.Vars.Get(currentHeightRoundKey(e.Replica))
	if !found {
		currentHrR = heightRound{height: 1, round: 0}
	}
	currentHr := currentHrR.(heightRound)

	if hr.height <= currentHr.height || (hr.height == currentHr.height && hr.round <= currentHr.round) {
		// Not interesting, we are already at that or a later round
		return
	}

	// Checks if we expect a newStep
	faults, ok := ctx.Vars.GetInt("faults")
	if !ok {
		panic("Number of faulty nodes not saved in vars")
	}
	if len(rr.received[hr].nodes) >= (faults + 1) {
		// Add an expected step
		esR, found := ctx.Vars.Get(expectedStepsKey(e.Replica))
		if !found {
			esR = expectedSteps{steps: make([]heightRound, 0)}
		}
		es := esR.(expectedSteps)
		es.Add(hr)
		logExpectedSteps(ctx, e.Replica, es)
		ctx.Vars.Set(expectedStepsKey(e.Replica), es)
	}

	return
}

func roundsReceivedKey(id types.ReplicaID) string {
	return fmt.Sprintf("BF_rounds_received_%s", id)
}

func expectedStepsKey(id types.ReplicaID) string {
	return fmt.Sprintf("BF_expected_steps_%s", id)
}

func TrackCurrentHeightRound(e *types.Event, ctx *testlib.Context) (messages []*types.Message, handled bool) {
	TrackRoundsReceived(e, ctx)
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
	//fmt.Printf("!! step,%s,%d,%d\n", getPartLabel(ctx, e.Replica), height, round)
	ctx.Logger().With(log.LogParams{
		"replica": e.Replica,
		"height":  height,
		"round":   round,
	}).Debug("newStep")
	hr := heightRound{height, round}
	ctx.Vars.Set(currentHeightRoundKey(e.Replica), hr)

	// Check if we expected this step already
	esR, found := ctx.Vars.Get(expectedStepsKey(e.Replica))
	if !found {
		return
	}
	es := esR.(expectedSteps)
	es.Remove(hr)
	ctx.Vars.Set(expectedStepsKey(e.Replica), es)
	logExpectedSteps(ctx, e.Replica, es)
	return
}

func currentHeightRoundKey(id types.ReplicaID) string {
	return fmt.Sprintf("BF_current_height_round_%s", id)
}

func logExpectedSteps(ctx *testlib.Context, r types.ReplicaID, es expectedSteps) {
	if len(es.steps) == 0 {
		ctx.Logger().With(log.LogParams{
			"node": getPartLabel(ctx, r),
		}).Info("No more steps expected")
	} else {
		ctx.Logger().With(log.LogParams{
			"node":  getPartLabel(ctx, r),
			"steps": fmt.Sprintf("%v", es.steps),
		}).Info("Expected more steps")
	}
}

func getPartLabel(ctx *testlib.Context, id types.ReplicaID) string {
	partitionR, ok := ctx.Vars.Get("partition")
	if !ok {
		panic("No partition found")
	}
	partition := partitionR.(*util.Partition)
	for _, p := range partition.Parts {
		if p.Contains(id) {
			return p.Label
		}
	}
	panic("Replica not found")
}
