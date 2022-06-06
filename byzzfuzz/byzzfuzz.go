package byzzfuzz

import (
	"byzzfuzz/byzzfuzz/spec"
	"encoding/json"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
)

func ByzzFuzzExpectNewRound(sp *common.SystemParams) (*testlib.TestCase, chan spec.Event) {
	isolatedValidator := 0
	otherNodes := []int{1, 2, 3}
	faulty := 1
	drops := []MessageDrop{
		// Isolate isolatedValidator in round 0
		{Step: 1, Partition: Partition{{isolatedValidator}, otherNodes}},
		{Step: 2, Partition: Partition{{isolatedValidator}, otherNodes}},
	}
	// Change all votes from faulty to nil
	allNodes := []int{0, 1, 2, 3}
	corruptions := []MessageCorruption{
		// Round 0
		{Step: 1, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 2, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		// Round 1
		{Step: 4, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 5, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		// Round 2
		{Step: 7, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 8, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		// Round 3
		{Step: 10, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 11, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		// Round 4
		{Step: 13, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 14, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
	}

	return ByzzFuzzInst(sp, drops, corruptions, 2*time.Minute)
}

type ByzzFuzzInstanceConfig struct {
	sysParams   *common.SystemParams
	Drops       []MessageDrop       `json:"drops"`
	Corruptions []MessageCorruption `json:"corruptions"`
	Timeout     time.Duration       `json:"timeout"`
}

func (c *ByzzFuzzInstanceConfig) TestCase() (*testlib.TestCase, chan spec.Event) {
	return ByzzFuzzInst(c.sysParams, c.Drops, c.Corruptions, c.Timeout)
}

func (c *ByzzFuzzInstanceConfig) Json() string {
	json, err := json.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	return string(json)
}

func ByzzFuzzRandom(sp *common.SystemParams,
	r *rand.Rand,
	nDrops int,
	nCorruptions int,
	steps int,
	timeout time.Duration) ByzzFuzzInstanceConfig {

	drops := make([]MessageDrop, nDrops)
	for i := range drops {
		drops[i] = MessageDrop{
			Step:      r.Intn(steps),
			Partition: RandomPartition(r),
		}
	}

	byzantineNode := r.Intn(sp.N)
	corruptions := make([]MessageCorruption, nCorruptions)
	for i := range corruptions {
		step := r.Intn(steps)
		corruptions[i] = MessageCorruption{
			Step:       step,
			From:       byzantineNode,
			To:         randomNonEmptySubset(r, sp.N),
			Corruption: randomCorruption(r, step),
		}
	}

	return ByzzFuzzInstanceConfig{sp, drops, corruptions, timeout}
}

func randomNonEmptySubset(r *rand.Rand, n int) []int {
	subset := r.Perm(n)[0:(1 + r.Intn(n-1))]
	sort.Ints(subset)
	return subset
}

func randomCorruption(r *rand.Rand, step int) CorruptionType {
	switch step % 3 {
	case 0:
		return ProposalCorruptionTypes[r.Intn(len(ProposalCorruptionTypes))]
	case 1:
		fallthrough
	case 2:
		return VoteCorruptionTypes[r.Intn(len(VoteCorruptionTypes))]
	default:
		panic("impossible")
	}
}
