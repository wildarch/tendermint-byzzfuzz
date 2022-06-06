package byzzfuzz

import (
	"encoding/json"
	"log"
	"time"

	"github.com/netrixframework/tendermint-testing/common"
)

var sysParams = common.NewSystemParams(4)

const bug001 = `{"drops":[{"step":8,"Partition":[[0],[1],[2,3]]},{"step":2,"Partition":[[3],[0,1,2]]},{"step":7,"Partition":[[0],[1],[2,3]]},{"step":0,"Partition":[[1],[3],[0,2]]},{"step":8,"Partition":[[1],[0,2,3]]}],"corruptions":[{"step":4,"from":3,"to":[],"corruption_type":1},{"step":5,"from":3,"to":[1],"corruption_type":2},{"step":9,"from":3,"to":[],"corruption_type":0},{"step":1,"from":3,"to":[0,1,3],"corruption_type":1},{"step":4,"from":3,"to":[0,1,2],"corruption_type":2}],"timeout":60000000000}`
const bug002 = `{"drops":[{"step":2,"Partition":[[1],[0,2,3]]},{"step":5,"Partition":[[0],[1,2,3]]},{"step":9,"Partition":[[2],[3],[0,1]]},{"step":7,"Partition":[[0],[1],[2],[3]]},{"step":3,"Partition":[[1],[0,2,3]]}],"corruptions":[{"step":1,"from":3,"to":[0,1],"corruption_type":2},{"step":7,"from":3,"to":[0,1],"corruption_type":2},{"step":5,"from":3,"to":[],"corruption_type":1},{"step":5,"from":3,"to":[],"corruption_type":1},{"step":5,"from":3,"to":[1,2,3],"corruption_type":2}],"timeout":60000000000}`
const bug003 = `{"drops":[{"step":3,"Partition":[[0],[2],[1,3]]},{"step":2,"Partition":[[3],[0,1,2]]},{"step":8,"Partition":[[0,1,2,3]]},{"step":7,"Partition":[[1],[3],[0,2]]},{"step":9,"Partition":[[0,1,2,3]]}],"corruptions":[{"step":1,"from":1,"to":[],"corruption_type":1},{"step":2,"from":1,"to":[],"corruption_type":2},{"step":1,"from":1,"to":[2],"corruption_type":1},{"step":6,"from":1,"to":[],"corruption_type":0},{"step":6,"from":1,"to":[],"corruption_type":0}],"timeout":60000000000}`
const bug004 = `{"drops":[{"step":2,"Partition":[[3],[0,1,2]]},{"step":3,"Partition":[[0,2],[1,3]]},{"step":5,"Partition":[[0,3],[1,2]]},{"step":8,"Partition":[[0],[3],[1,2]]},{"step":0,"Partition":[[0],[2],[1,3]]}],"corruptions":[{"step":4,"from":1,"to":[0,1],"corruption_type":2},{"step":2,"from":1,"to":[1],"corruption_type":1},{"step":5,"from":1,"to":[1,2,3],"corruption_type":2},{"step":2,"from":1,"to":[],"corruption_type":1},{"step":1,"from":1,"to":[0,1,3],"corruption_type":1}],"timeout":60000000000}`
const bug005 = `{"drops":[{"step":6,"Partition":[[1],[3],[0,2]]},{"step":6,"Partition":[[1],[2],[0,3]]},{"step":0,"Partition":[[1],[0,2,3]]},{"step":6,"Partition":[[0,1,2,3]]},{"step":3,"Partition":[[0,1,2,3]]}],"corruptions":[{"step":9,"from":3,"to":[3],"corruption_type":0},{"step":7,"from":3,"to":[0,1,3],"corruption_type":2},{"step":2,"from":3,"to":[],"corruption_type":1},{"step":1,"from":3,"to":[0,3],"corruption_type":2},{"step":6,"from":3,"to":[0,2],"corruption_type":0}],"timeout":60000000000}`
const bug006 = `{"drops":[{"step":3,"Partition":[[0],[1,2,3]]},{"step":8,"Partition":[[0,2],[1,3]]},{"step":8,"Partition":[[0],[1],[2],[3]]},{"step":7,"Partition":[[0,1,2,3]]},{"step":3,"Partition":[[0,1,2,3]]}],"corruptions":[{"step":7,"from":2,"to":[1,2,3],"corruption_type":2},{"step":6,"from":2,"to":[0],"corruption_type":0},{"step":4,"from":2,"to":[2,3],"corruption_type":2},{"step":5,"from":2,"to":[2],"corruption_type":1},{"step":8,"from":2,"to":[1,2,3],"corruption_type":1}],"timeout":60000000000}`

func Bug001() ByzzFuzzInstanceConfig { return makeConfig(bug001) } // Not reproducible, Looks like timeout
func Bug002() ByzzFuzzInstanceConfig { return makeConfig(bug002) } // Not reproducible
func Bug003() ByzzFuzzInstanceConfig { return makeConfig(bug003) } // Reproducible
func Bug004() ByzzFuzzInstanceConfig { return makeConfig(bug004) } // Reproducible
func Bug005() ByzzFuzzInstanceConfig { return makeConfig(bug005) } // Not reproducible
func Bug006() ByzzFuzzInstanceConfig { return makeConfig(bug006) } // Not reproducible

func Bug003Reprod() ByzzFuzzInstanceConfig {
	return ByzzFuzzInstanceConfig{
		sysParams: sysParams,
		Drops: []MessageDrop{
			{Step: 3, Partition: Partition{{0}, {2}, {1, 3}}},
			{Step: 2, Partition: Partition{{3}, {0, 1, 2}}},
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
