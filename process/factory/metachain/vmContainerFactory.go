package metachain

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/vm"
	systemVMFactory "github.com/ElrondNetwork/elrond-go/vm/factory"
	systemVMProcess "github.com/ElrondNetwork/elrond-go/vm/process"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/parsers"
)

var _ process.VirtualMachinesContainerFactory = (*vmContainerFactory)(nil)

type vmContainerFactory struct {
	chanceComputer      sharding.ChanceComputer
	validatorAccountsDB state.AccountsAdapter
	blockChainHookImpl  *hooks.BlockChainHookImpl
	cryptoHook          vmcommon.CryptoHook
	systemContracts     vm.SystemSCContainer
	economics           *economics.EconomicsData
	messageSigVerifier  vm.MessageSignVerifier
	nodesConfigProvider vm.NodesConfigProvider
	gasSchedule         map[string]map[string]uint64
	hasher              hashing.Hasher
	marshalizer         marshal.Marshalizer
	systemSCConfig      *config.SystemSmartContractsConfig
	epochNotifier       process.EpochNotifier
}

// ArgsNewVMContainerFactory defines the arguments needed to create a new VM container factory
type ArgsNewVMContainerFactory struct {
	ArgBlockChainHook   hooks.ArgBlockChainHook
	Economics           *economics.EconomicsData
	MessageSignVerifier vm.MessageSignVerifier
	GasSchedule         map[string]map[string]uint64
	NodesConfigProvider vm.NodesConfigProvider
	Hasher              hashing.Hasher
	Marshalizer         marshal.Marshalizer
	SystemSCConfig      *config.SystemSmartContractsConfig
	ValidatorAccountsDB state.AccountsAdapter
	ChanceComputer      sharding.ChanceComputer
	EpochNotifier       process.EpochNotifier
	SystemSCContainer   vm.SystemSCContainer
}

// NewVMContainerFactory is responsible for creating a new virtual machine factory object
func NewVMContainerFactory(args ArgsNewVMContainerFactory) (*vmContainerFactory, error) {
	if args.Economics == nil {
		return nil, process.ErrNilEconomicsData
	}
	if check.IfNil(args.MessageSignVerifier) {
		return nil, process.ErrNilKeyGen
	}
	if check.IfNil(args.NodesConfigProvider) {
		return nil, process.ErrNilNodesConfigProvider
	}
	if check.IfNil(args.Hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(args.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if args.SystemSCConfig == nil {
		return nil, process.ErrNilSystemSCConfig
	}
	if check.IfNil(args.ValidatorAccountsDB) {
		return nil, vm.ErrNilValidatorAccountsDB
	}
	if check.IfNil(args.ChanceComputer) {
		return nil, vm.ErrNilChanceComputer
	}
	blockChainHookImpl, err := hooks.NewBlockChainHookImpl(args.ArgBlockChainHook)
	if err != nil {
		return nil, err
	}
	cryptoHook := hooks.NewVMCryptoHook()

	return &vmContainerFactory{
		blockChainHookImpl:  blockChainHookImpl,
		cryptoHook:          cryptoHook,
		economics:           args.Economics,
		messageSigVerifier:  args.MessageSignVerifier,
		gasSchedule:         args.GasSchedule,
		nodesConfigProvider: args.NodesConfigProvider,
		hasher:              args.Hasher,
		marshalizer:         args.Marshalizer,
		systemSCConfig:      args.SystemSCConfig,
		validatorAccountsDB: args.ValidatorAccountsDB,
		chanceComputer:      args.ChanceComputer,
		epochNotifier:       args.EpochNotifier,
		systemContracts:     args.SystemSCContainer,
	}, nil
}

// Create sets up all the needed virtual machine returning a container of all the VMs
func (vmf *vmContainerFactory) Create() (process.VirtualMachinesContainer, error) {
	container := containers.NewVirtualMachinesContainer()

	currVm, err := vmf.createSystemVM()
	if err != nil {
		return nil, err
	}

	err = container.Add(factory.SystemVirtualMachine, currVm)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (vmf *vmContainerFactory) createSystemVM() (vmcommon.VMExecutionHandler, error) {
	atArgumentParser := parsers.NewCallArgsParser()
	systemEI, err := systemSmartContracts.NewVMContext(
		vmf.blockChainHookImpl,
		vmf.cryptoHook,
		atArgumentParser,
		vmf.validatorAccountsDB,
		vmf.chanceComputer,
	)
	if err != nil {
		return nil, err
	}

	argsNewSystemScFactory := systemVMFactory.ArgsNewSystemSCFactory{
		SystemEI:            systemEI,
		SigVerifier:         vmf.messageSigVerifier,
		GasMap:              vmf.gasSchedule,
		NodesConfigProvider: vmf.nodesConfigProvider,
		Hasher:              vmf.hasher,
		Marshalizer:         vmf.marshalizer,
		SystemSCConfig:      vmf.systemSCConfig,
		Economics:           vmf.economics,
		EpochNotifier:       vmf.epochNotifier,
	}
	scFactory, err := systemVMFactory.NewSystemSCFactory(argsNewSystemScFactory)
	if err != nil {
		return nil, err
	}

	if check.IfNil(vmf.systemContracts) {
		vmf.systemContracts, err = scFactory.Create()
		if err != nil {
			return nil, err
		}
	}

	err = systemEI.SetSystemSCContainer(vmf.systemContracts)
	if err != nil {
		return nil, err
	}

	systemVM, err := systemVMProcess.NewSystemVM(systemEI, vmf.systemContracts, factory.SystemVirtualMachine)
	if err != nil {
		return nil, err
	}

	return systemVM, nil
}

// BlockChainHookImpl returns the created blockChainHookImpl
func (vmf *vmContainerFactory) BlockChainHookImpl() process.BlockChainHookHandler {
	return vmf.blockChainHookImpl
}

// SystemSmartContractContainer return the created system smart contracts
func (vmf *vmContainerFactory) SystemSmartContractContainer() vm.SystemSCContainer {
	return vmf.systemContracts
}

// IsInterfaceNil returns true if there is no value under the interface
func (vmf *vmContainerFactory) IsInterfaceNil() bool {
	return vmf == nil
}
