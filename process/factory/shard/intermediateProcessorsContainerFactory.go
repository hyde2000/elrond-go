package shard

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/postprocess"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type intermediateProcessorsContainerFactory struct {
	shardCoordinator        sharding.Coordinator
	marshalizer             marshal.Marshalizer
	hasher                  hashing.Hasher
	pubkeyConverter         core.PubkeyConverter
	store                   dataRetriever.StorageService
	poolsHolder             dataRetriever.PoolsHolder
	economicsFee            process.FeeHandler
	postProcessorTxsHandler process.PostProcessorTxsHandler
}

// NewIntermediateProcessorsContainerFactory is responsible for creating a new intermediate processors factory object
func NewIntermediateProcessorsContainerFactory(
	shardCoordinator sharding.Coordinator,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	pubkeyConverter core.PubkeyConverter,
	store dataRetriever.StorageService,
	poolsHolder dataRetriever.PoolsHolder,
	economicsFee process.FeeHandler,
	postProcessorTxsHandler process.PostProcessorTxsHandler,
) (*intermediateProcessorsContainerFactory, error) {

	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(pubkeyConverter) {
		return nil, process.ErrNilPubkeyConverter
	}
	if check.IfNil(store) {
		return nil, process.ErrNilStorage
	}
	if check.IfNil(poolsHolder) {
		return nil, process.ErrNilPoolsHolder
	}
	if check.IfNil(economicsFee) {
		return nil, process.ErrNilEconomicsFeeHandler
	}
	if check.IfNil(postProcessorTxsHandler) {
		return nil, process.ErrNilPostProcessorTxsHandler
	}

	return &intermediateProcessorsContainerFactory{
		shardCoordinator:        shardCoordinator,
		marshalizer:             marshalizer,
		hasher:                  hasher,
		pubkeyConverter:         pubkeyConverter,
		store:                   store,
		poolsHolder:             poolsHolder,
		economicsFee:            economicsFee,
		postProcessorTxsHandler: postProcessorTxsHandler,
	}, nil
}

// Create returns a preprocessor container that will hold all preprocessors in the system
func (ppcm *intermediateProcessorsContainerFactory) Create() (process.IntermediateProcessorContainer, error) {
	container := containers.NewIntermediateTransactionHandlersContainer()

	interproc, err := ppcm.createSmartContractResultsIntermediateProcessor()
	if err != nil {
		return nil, err
	}

	err = container.Add(block.SmartContractResultBlock, interproc)
	if err != nil {
		return nil, err
	}

	interproc, err = ppcm.createReceiptIntermediateProcessor()
	if err != nil {
		return nil, err
	}

	err = container.Add(block.ReceiptBlock, interproc)
	if err != nil {
		return nil, err
	}

	interproc, err = ppcm.createBadTransactionsIntermediateProcessor()
	if err != nil {
		return nil, err
	}

	err = container.Add(block.InvalidBlock, interproc)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (ppcm *intermediateProcessorsContainerFactory) createSmartContractResultsIntermediateProcessor() (process.IntermediateTransactionHandler, error) {
	irp, err := postprocess.NewIntermediateResultsProcessor(
		ppcm.hasher,
		ppcm.marshalizer,
		ppcm.shardCoordinator,
		ppcm.pubkeyConverter,
		ppcm.store,
		block.SmartContractResultBlock,
		ppcm.poolsHolder.CurrentBlockTxs(),
		ppcm.economicsFee,
		ppcm.postProcessorTxsHandler,
	)

	return irp, err
}

func (ppcm *intermediateProcessorsContainerFactory) createReceiptIntermediateProcessor() (process.IntermediateTransactionHandler, error) {
	irp, err := postprocess.NewOneMiniBlockPostProcessor(
		ppcm.hasher,
		ppcm.marshalizer,
		ppcm.shardCoordinator,
		ppcm.store,
		block.ReceiptBlock,
		dataRetriever.UnsignedTransactionUnit,
		ppcm.economicsFee,
		ppcm.postProcessorTxsHandler,
	)

	return irp, err
}

func (ppcm *intermediateProcessorsContainerFactory) createBadTransactionsIntermediateProcessor() (process.IntermediateTransactionHandler, error) {
	irp, err := postprocess.NewOneMiniBlockPostProcessor(
		ppcm.hasher,
		ppcm.marshalizer,
		ppcm.shardCoordinator,
		ppcm.store,
		block.InvalidBlock,
		dataRetriever.TransactionUnit,
		ppcm.economicsFee,
		ppcm.postProcessorTxsHandler,
	)

	return irp, err
}

// IsInterfaceNil returns true if there is no value under the interface
func (ppcm *intermediateProcessorsContainerFactory) IsInterfaceNil() bool {
	return ppcm == nil
}
