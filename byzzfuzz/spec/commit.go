package spec

import (
	"fmt"
	"strconv"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

func DiffCommits(e *types.Event, c *testlib.Context) bool {
	eType, ok := e.Type.(*types.GenericEventType)
	if ok && eType.T == "Committing block" {
		heightS, ok := eType.Params["height"]
		if !ok {
			return false
		}
		height, err := strconv.Atoi(heightS)
		if err != nil {
			return false
		}

		blockID, ok := eType.Params["block_id"]
		if ok {
			curBlockID, exists := c.Vars.GetString(blockIdKey(height))
			if !exists {
				c.Vars.Set(blockIdKey(height), blockID)
				return false
			}
			c.Logger().With(log.LogParams{
				"cur_block_id": curBlockID,
				"block_id":     blockID,
			}).Info("Checking if block IDs match")
			return blockID != curBlockID
		}
	}
	return false
}

func blockIdKey(height int) string {
	return fmt.Sprintf("BF_block_id_height_%d", height)
}
