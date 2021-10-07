package detector

import (
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
	"github.com/ElrondNetwork/elrond-go/process/slash"
)

// CacheSize represents how many rounds will be cached in the slashing detector
const CacheSize = 10

// minSlashableNoOfHeaders represents the min number of headers required for a
// proof to be considered slashable
const minSlashableNoOfHeaders = 2

// MaxDeltaToCurrentRound represents the max delta from the current round to any
// other round from an intercepted data in order for a detector to process it and cache it
const MaxDeltaToCurrentRound = 3

type baseSlashingDetector struct {
	roundHandler process.RoundHandler
}

func (bsd *baseSlashingDetector) isRoundRelevant(round uint64) bool {
	currRound := uint64(bsd.roundHandler.Index())
	return absDiff(currRound, round) < MaxDeltaToCurrentRound
}

func absDiff(x, y uint64) uint64 {
	if x < y {
		return y - x
	}
	return x - y
}

func computeSlashLevel(data []process.InterceptedData) slash.ThreatLevel {
	ret := slash.Low
	// TODO: Maybe a linear interpolation to deduce severity?
	if len(data) == 2 {
		ret = slash.Medium
	} else if len(data) >= 3 {
		ret = slash.High
	}

	return ret
}

func checkSlashLevel(headers []*interceptedBlocks.InterceptedHeader, level slash.ThreatLevel) error {
	if level < slash.Medium || level > slash.High {
		return process.ErrInvalidSlashLevel
	}
	if len(headers) < minSlashableNoOfHeaders {
		return process.ErrNotEnoughHeadersProvided
	}
	if len(headers) == minSlashableNoOfHeaders && level != slash.Medium {
		return process.ErrSlashLevelDoesNotMatchSlashType
	}
	if len(headers) > minSlashableNoOfHeaders && level != slash.High {
		return process.ErrSlashLevelDoesNotMatchSlashType
	}

	return nil
}
