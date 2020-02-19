package rbxl

import (
	"math"
)

func matrixFromID(i uint8) (m [9]float32) {
	i--
	// Ignore IDs that produce invalid matrices. 0, which is wrapped to 255,
	// indicates a non-special CFrame, so it must also be ignored.
	if i >= 35 || i/6%3 == i%3 {
		return m
	}
	// Set directions of X and Y axes.
	m[i/6%3*3] = 1 - float32(i/18*2)
	m[i%6%3*3+1] = 1 - float32(i%6/3*2)
	// Set Z axis to cross product of X and Y.
	m[2] = m[3]*m[7] - m[4]*m[6]
	m[5] = m[6]*m[1] - m[7]*m[0]
	m[8] = m[0]*m[4] - m[1]*m[3]
	return m
}

var _0 = float32(math.Copysign(0, -1))

// DIFF: Unspecified IDs produce either invalid matrices or garbage values, so
// these are assumed to be undefined.
var cframeSpecialMatrix = map[uint8][9]float32{
	// i: m if m := matrixFromID(i); m != [9]float32{}
	0x02: {+1, +0, +0, +0, +1, +0, +0, +0, +1},
	0x03: {+1, +0, +0, +0, +0, -1, +0, +1, +0},
	0x05: {+1, +0, +0, +0, -1, +0, +0, +0, -1},
	0x06: {+1, +0, _0, +0, +0, +1, +0, -1, +0},
	0x07: {+0, +1, +0, +1, +0, +0, +0, +0, -1},
	0x09: {+0, +0, +1, +1, +0, +0, +0, +1, +0},
	0x0A: {+0, -1, +0, +1, +0, _0, +0, +0, +1},
	0x0C: {+0, +0, -1, +1, +0, +0, +0, -1, +0},
	0x0D: {+0, +1, +0, +0, +0, +1, +1, +0, +0},
	0x0E: {+0, +0, -1, +0, +1, +0, +1, +0, +0},
	0x10: {+0, -1, +0, +0, +0, -1, +1, +0, +0},
	0x11: {+0, +0, +1, +0, -1, +0, +1, +0, _0},
	0x14: {-1, +0, +0, +0, +1, +0, +0, +0, -1},
	0x15: {-1, +0, +0, +0, +0, +1, +0, +1, _0},
	0x17: {-1, +0, +0, +0, -1, +0, +0, +0, +1},
	0x18: {-1, +0, _0, +0, +0, -1, +0, -1, _0},
	0x19: {+0, +1, _0, -1, +0, +0, +0, +0, +1},
	0x1B: {+0, +0, -1, -1, +0, +0, +0, +1, +0},
	0x1C: {+0, -1, _0, -1, +0, _0, +0, +0, -1},
	0x1E: {+0, +0, +1, -1, +0, +0, +0, -1, +0},
	0x1F: {+0, +1, +0, +0, +0, -1, -1, +0, +0},
	0x20: {+0, +0, +1, +0, +1, _0, -1, +0, +0},
	0x22: {+0, -1, +0, +0, +0, +1, -1, +0, +0},
	0x23: {+0, +0, -1, +0, -1, _0, -1, +0, _0},
}

var cframeSpecialNumber = map[[9]float32]uint8{
	cframeSpecialMatrix[0x02]: 0x02,
	cframeSpecialMatrix[0x03]: 0x03,
	cframeSpecialMatrix[0x05]: 0x05,
	cframeSpecialMatrix[0x06]: 0x06,
	cframeSpecialMatrix[0x07]: 0x07,
	cframeSpecialMatrix[0x09]: 0x09,
	cframeSpecialMatrix[0x0A]: 0x0A,
	cframeSpecialMatrix[0x0C]: 0x0C,
	cframeSpecialMatrix[0x0D]: 0x0D,
	cframeSpecialMatrix[0x0E]: 0x0E,
	cframeSpecialMatrix[0x10]: 0x10,
	cframeSpecialMatrix[0x11]: 0x11,
	cframeSpecialMatrix[0x14]: 0x14,
	cframeSpecialMatrix[0x15]: 0x15,
	cframeSpecialMatrix[0x17]: 0x17,
	cframeSpecialMatrix[0x18]: 0x18,
	cframeSpecialMatrix[0x19]: 0x19,
	cframeSpecialMatrix[0x1B]: 0x1B,
	cframeSpecialMatrix[0x1C]: 0x1C,
	cframeSpecialMatrix[0x1E]: 0x1E,
	cframeSpecialMatrix[0x1F]: 0x1F,
	cframeSpecialMatrix[0x20]: 0x20,
	cframeSpecialMatrix[0x22]: 0x22,
	cframeSpecialMatrix[0x23]: 0x23,
}
