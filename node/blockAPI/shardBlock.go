package blockAPI

import (
	"encoding/hex"
	"github.com/ElrondNetwork/elrond-go/process"
	"time"

	"github.com/ElrondNetwork/elrond-go/data/api"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node/filters"
)

type shardAPIBlockProcessor struct {
	*baseAPIBockProcessor
}

// NewShardApiBlockProcessor will create a new instance of shard api block processor
func NewShardApiBlockProcessor(arg *APIBlockProcessorArg) *shardAPIBlockProcessor {
	hasDbLookupExtensions := arg.HistoryRepo.IsEnabled()

	return &shardAPIBlockProcessor{
		baseAPIBockProcessor: &baseAPIBockProcessor{
			hasDbLookupExtensions:    hasDbLookupExtensions,
			selfShardID:              arg.SelfShardID,
			store:                    arg.Store,
			marshalizer:              arg.Marshalizer,
			uint64ByteSliceConverter: arg.Uint64ByteSliceConverter,
			historyRepo:              arg.HistoryRepo,
			unmarshalTx:              arg.UnmarshalTx,
			txStatusComputer:         arg.StatusComputer,
		},
	}
}

// GetBlockByNonce will return a shard APIBlock by nonce
func (sbp *shardAPIBlockProcessor) GetBlockByNonce(nonce uint64, withTxs bool) (*api.Block, error) {
	storerUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(sbp.selfShardID)

	nonceToByteSlice := sbp.uint64ByteSliceConverter.ToByteSlice(nonce)
	headerHash, err := sbp.store.Get(storerUnit, nonceToByteSlice)
	if err != nil {
		return nil, err
	}

	blockBytes, err := sbp.getFromStorer(dataRetriever.BlockHeaderUnit, headerHash)
	if err != nil {
		return nil, err
	}

	return sbp.convertShardBlockBytesToAPIBlock(headerHash, blockBytes, withTxs)
}

// GetBlockByHash will return a shard APIBlock by hash
func (sbp *shardAPIBlockProcessor) GetBlockByHash(hash []byte, withTxs bool) (*api.Block, error) {
	blockBytes, err := sbp.getFromStorer(dataRetriever.BlockHeaderUnit, hash)
	if err != nil {
		return nil, err
	}

	blockAPI, err := sbp.convertShardBlockBytesToAPIBlock(hash, blockBytes, withTxs)
	if err != nil {
		return nil, err
	}

	storerUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(sbp.selfShardID)

	return sbp.computeStatusAndPutInBlock(blockAPI, storerUnit)
}

func (sbp *shardAPIBlockProcessor) convertShardBlockBytesToAPIBlock(hash []byte, blockBytes []byte, withTxs bool) (*api.Block, error) {
	blockHeader, err := process.CreateShardHeader(sbp.marshalizer, blockBytes)
	if err != nil {
		return nil, err
	}

	headerEpoch := blockHeader.GetEpoch()

	numOfTxs := uint32(0)
	miniblocks := make([]*api.MiniBlock, 0)
	for _, mb := range blockHeader.GetMiniBlockHeaderHandlers() {
		if block.Type(mb.GetTypeInt32()) == block.PeerBlock {
			continue
		}

		numOfTxs += mb.GetTxCount()

		miniblockAPI := &api.MiniBlock{
			Hash:             hex.EncodeToString(mb.GetHash()),
			Type:             block.Type(mb.GetTypeInt32()).String(),
			SourceShard:      mb.GetSenderShardID(),
			DestinationShard: mb.GetReceiverShardID(),
		}
		if withTxs {
			miniBlockCopy := mb
			miniblockAPI.Transactions = sbp.getTxsByMb(miniBlockCopy, headerEpoch)
		}

		miniblocks = append(miniblocks, miniblockAPI)
	}

	statusFilters := filters.NewStatusFilters(sbp.selfShardID)
	statusFilters.ApplyStatusFilters(miniblocks)

	return &api.Block{
		Nonce:           blockHeader.GetNonce(),
		Round:           blockHeader.GetRound(),
		Epoch:           blockHeader.GetEpoch(),
		Shard:           blockHeader.GetShardID(),
		Hash:            hex.EncodeToString(hash),
		PrevBlockHash:   hex.EncodeToString(blockHeader.GetPrevHash()),
		NumTxs:          numOfTxs,
		MiniBlocks:      miniblocks,
		AccumulatedFees: blockHeader.GetAccumulatedFees().String(),
		DeveloperFees:   blockHeader.GetDeveloperFees().String(),
		Timestamp:       time.Duration(blockHeader.GetTimeStamp()),
		Status:          BlockStatusOnChain,
	}, nil
}
