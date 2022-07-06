from dataclasses import dataclass
from enum import IntEnum
import itertools

@dataclass(eq=True, order=True)
class MessageDrop:
    step: int
    partition: list[list[int]]

class CorruptionType(IntEnum):
	CHANGE_PROPOSAL_TO_NIL = 0
	CHANGE_VOTE_TO_NIL = 1
	CHANGE_VOTE_ROUND = 2
	OMIT = 3
	CHANGE_BLOCK_ID = 4

@dataclass(eq=True, order=True)
class MessageCorruption:
    step: int
    from_node: int
    to_nodes: list[int]
    corruption_type: CorruptionType
    seed: int

@dataclass(eq=True)
class ByzzFuzzInstanceConfig:
    drops: list[MessageDrop]
    corruptions: list[MessageCorruption]


ALL_PARTITIONS = [
	# 1 partition (disabled because it is equivalent to having no partition at all)
	# [[0, 1, 2, 3]],

	# 2 partitions
	[[0, 1], [2, 3]],
	[[0, 2], [1, 3]],
	[[0, 3], [1, 2]],
	[[0], [1, 2, 3]],
	[[1], [0, 2, 3]],
	[[2], [0, 1, 3]],
	[[3], [0, 1, 2]],

	# 3 partitions
	[[0], [1], [2, 3]],
	[[0], [2], [1, 3]],
	[[0], [3], [1, 2]],
	[[1], [2], [0, 3]],
	[[1], [3], [0, 2]],
	[[2], [3], [0, 1]],

	# 4 partitions
	[[0], [1], [2], [3]],
]

MAX_STEPS = 10
ALL_DROPS = [MessageDrop(step, part) for step, part in itertools.product(range(MAX_STEPS), ALL_PARTITIONS)]

ALL_PROPOSAL_CORRUPTION_TYPES = [
	CorruptionType.CHANGE_BLOCK_ID,
	CorruptionType.OMIT,
]

ALL_VOTE_CORRUPTION_TYPES = [
	CorruptionType.CHANGE_BLOCK_ID,
	CorruptionType.OMIT,
]

ALL_SUBSETS = [
	# Four
	[0, 1, 2, 3],
	# Three
	[0, 1, 2],
	[0, 1, 3],
	[0, 2, 3],
	[1, 2, 3],
	# Two
	[0, 1],
	[0, 2],
	[0, 3],
	[1, 2],
	[1, 3],
	[2, 3],
	# One
	[0],
	[1],
	[2],
	[3],
]

ALL_NODES = range(4)