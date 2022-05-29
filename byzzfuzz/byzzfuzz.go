package byzzfuzz

import (
	"encoding/json"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
)

func ByzzFuzzExpectNewRound(sp *common.SystemParams) *testlib.TestCase {
	isolatedValidator := 0
	faulty := 1
	drops := []MessageDrop{
		// Drop all Prevote or Precommit to isolatedValidator in round 0
		{Step: 1, From: 0, To: isolatedValidator},
		{Step: 1, From: 1, To: isolatedValidator},
		{Step: 1, From: 2, To: isolatedValidator},
		{Step: 1, From: 3, To: isolatedValidator},
		{Step: 2, From: 0, To: isolatedValidator},
		{Step: 2, From: 1, To: isolatedValidator},
		{Step: 2, From: 2, To: isolatedValidator},
		{Step: 2, From: 3, To: isolatedValidator},

		// Drop all Prevote or Precommit from isolatedValidator in round 0
		{Step: 1, From: isolatedValidator, To: 0},
		{Step: 1, From: isolatedValidator, To: 1},
		{Step: 1, From: isolatedValidator, To: 2},
		{Step: 1, From: isolatedValidator, To: 3},
		{Step: 2, From: isolatedValidator, To: 0},
		{Step: 2, From: isolatedValidator, To: 1},
		{Step: 2, From: isolatedValidator, To: 2},
		{Step: 2, From: isolatedValidator, To: 3},
	}
	// Change all votes from faulty to nil
	allNodes := []int{0, 1, 2, 3}
	corruptions := []MessageCorruption{
		{Step: 1, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 2, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 4, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 5, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 7, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Step: 8, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		// Assume three rounds is enough
	}

	return ByzzFuzzInst(sp, drops, corruptions, 2*time.Minute)
}

type ByzzFuzzInstanceConfig struct {
	sysParams   *common.SystemParams
	Drops       []MessageDrop       `json:"drops"`
	Corruptions []MessageCorruption `json:"corruptions"`
	Timeout     time.Duration       `json:"timeout"`
}

func (c *ByzzFuzzInstanceConfig) TestCase() *testlib.TestCase {
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
	maxDrops int,
	maxCorruptions int,
	maxSteps int,
	timeout time.Duration) ByzzFuzzInstanceConfig {

	nDrops := r.Intn(maxDrops)
	drops := make([]MessageDrop, nDrops)
	for i := range drops {
		drops[i] = MessageDrop{
			Step: r.Intn(maxSteps),
			From: r.Intn(sp.N),
			To:   r.Intn(sp.N),
		}
	}

	byzantineNode := r.Intn(sp.N)
	nCorruptions := r.Intn(maxCorruptions)
	corruptions := make([]MessageCorruption, nCorruptions)
	for i := range corruptions {
		step := r.Intn(maxSteps)
		corruptions[i] = MessageCorruption{
			Step:       step,
			From:       byzantineNode,
			To:         randomSubset(r, sp.N),
			Corruption: randomCorruption(r, step),
		}
	}

	return ByzzFuzzInstanceConfig{sp, drops, corruptions, timeout}
}

func randomSubset(r *rand.Rand, n int) []int {
	subset := r.Perm(n)[0:r.Intn(n)]
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
