package byzzfuzz

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
)

func ByzzFuzzExpectNewRound(sp *common.SystemParams) *testlib.TestCase {
	isolatedValidator := 0
	faulty := 1

	drops := []MessageDrop{
		// ROUND 0
		// Drops everything from isolatedValidator
		{Round: 0, From: isolatedValidator, To: 0},
		{Round: 0, From: isolatedValidator, To: 1},
		{Round: 0, From: isolatedValidator, To: 2},
		{Round: 0, From: isolatedValidator, To: 3},
		// Drops everything to isolatedValidator
		{Round: 0, From: 0, To: isolatedValidator},
		{Round: 0, From: 1, To: isolatedValidator},
		{Round: 0, From: 2, To: isolatedValidator},
		{Round: 0, From: 3, To: isolatedValidator},
	}

	allNodes := []int{0, 1, 2, 3}
	corruptions := []MessageCorruption{
		{Round: 0, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
		{Round: 1, From: faulty, To: allNodes, Corruption: ChangeVoteToNil},
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
	maxRounds int,
	timeout time.Duration) ByzzFuzzInstanceConfig {

	nDrops := rand.Intn(maxDrops)
	drops := make([]MessageDrop, nDrops)
	for i, _ := range drops {
		drops[i] = MessageDrop{
			Round: rand.Intn(maxRounds),
			From:  rand.Intn(sp.N),
			To:    rand.Intn(sp.N),
		}
	}

	byzantineNode := rand.Intn(sp.N)
	nCorruptions := rand.Intn(maxCorruptions)
	corruptions := make([]MessageCorruption, nCorruptions)
	for i, _ := range corruptions {
		corruptions[i] = MessageCorruption{
			Round:      rand.Intn(maxRounds),
			From:       byzantineNode,
			To:         randomSubset(r, sp.N),
			Corruption: randomCorruption(r),
		}
	}

	return ByzzFuzzInstanceConfig{sp, drops, corruptions, timeout}
}

func randomSubset(r *rand.Rand, n int) []int {
	return r.Perm(n)[0:r.Intn(n)]
}

func randomCorruption(r *rand.Rand) CorruptionType {
	return CorruptionTypes[r.Intn(len(CorruptionTypes))]
}
