package byzzfuzz

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
)

func InstanceFromJson(r io.Reader) (ByzzFuzzInstanceConfig, error) {
	instconf := ByzzFuzzInstanceConfig{}
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&instconf)
	if err != nil {
		return instconf, err
	}
	instconf.sysParams = common.NewSystemParams(4)
	instconf.Timeout = 1 * time.Minute
	instconf.LivenessTimeout = 1 * time.Minute
	return instconf, nil
}

type ByzzFuzzInstanceConfig struct {
	sysParams       *common.SystemParams
	Drops           []MessageDrop       `json:"drops"`
	Corruptions     []MessageCorruption `json:"corruptions"`
	Timeout         time.Duration       `json:"timeout"`
	LivenessTimeout time.Duration       `json:"liveness_timeout"`
}

func (c *ByzzFuzzInstanceConfig) TestCase() *testlib.TestCase {
	return ByzzFuzzInst(c.sysParams, c.Drops, c.Corruptions, c.Timeout, c.LivenessTimeout)
}

func (c *ByzzFuzzInstanceConfig) Json() string {
	json, err := json.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	return string(json)
}
