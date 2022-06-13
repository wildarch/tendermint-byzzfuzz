package byzzfuzz

import (
	"encoding/json"
	"log"
	"time"

	"github.com/netrixframework/tendermint-testing/common"
)

var sysParams = common.NewSystemParams(4)

const bug001 = `{"drops":[{"step":1,"Partition":[[1],[3],[0,2]]},{"step":6,"Partition":[[0,2],[1,3]]}],"corruptions":[],"timeout":60000000000}`
const bug002 = `{"drops":[{"step":6,"Partition":[[0],[3],[1,2]]},{"step":2,"Partition":[[0],[2],[1,3]]}],"corruptions":[],"timeout":60000000000}`
const bug003 = `{"drops":[{"step":8,"Partition":[[0,2],[1,3]]},{"step":0,"Partition":[[0],[1,2,3]]}],"corruptions":[],"timeout":60000000000}`

func Bug001() ByzzFuzzInstanceConfig { return makeConfig(bug001) } // Does not pass, even with 5 minute liveness timeout
func Bug002() ByzzFuzzInstanceConfig { return makeConfig(bug002) } // Gets partitioned at step 2 and never recovers
func Bug003() ByzzFuzzInstanceConfig { return makeConfig(bug003) } // Gets stuck at step 8

func Lagging() ByzzFuzzInstanceConfig {
	return ByzzFuzzInstanceConfig{
		sysParams: sysParams,
		Drops: []MessageDrop{
			{Step: 2, Partition: Partition{{3}, {0, 1, 2}}},
			{Step: 5, Partition: Partition{{3}, {0, 1, 2}}},
			{Step: 8, Partition: Partition{{3}, {0, 1, 2}}},
		},
		Corruptions: []MessageCorruption{},
		Timeout:     time.Minute,
	}
}

func makeConfig(bug string) ByzzFuzzInstanceConfig {
	instconf := ByzzFuzzInstanceConfig{}
	err := json.Unmarshal([]byte(bug), &instconf)
	if err != nil {
		log.Fatalf("failed to parse JSON definition for bug: %s", err.Error())
	}
	instconf.sysParams = sysParams
	return instconf
}
