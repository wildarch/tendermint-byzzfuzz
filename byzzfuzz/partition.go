package byzzfuzz

import (
	"log"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

type Partition = [][]int

func FromToIsolated(p Partition) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		from := replicaIdx(c, message.From)
		to := replicaIdx(c, message.To)

		return isolates(p, from, to)
	}
}

func isolates(p Partition, a int, b int) bool {
	for _, part := range p {
		if partContains(part, a) && partContains(part, b) {
			return false
		}
	}
	return true
}

func partContains(part []int, i int) bool {
	for _, v := range part {
		if v == i {
			return true
		}
	}

	return false
}

func replicaIdx(c *testlib.Context, id types.ReplicaID) int {
	for i, v := range c.Replicas.Iter() {
		if id == v.ID {
			return i
		}
	}
	log.Fatalf("cannot find replica %s", id)
	return -1
}
