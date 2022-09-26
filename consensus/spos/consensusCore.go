package spos

import (
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
	cryptoCommon "github.com/ElrondNetwork/elrond-go/common/crypto"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
)

// ConsensusCore implements ConsensusCoreHandler and provides access to common functionality
//  for the rest of the consensus structures
type ConsensusCore struct {
	blockChain                    data.ChainHandler
	blockProcessor                process.BlockProcessor
	bootstrapper                  process.Bootstrapper
	broadcastMessenger            consensus.BroadcastMessenger
	chronologyHandler             consensus.ChronologyHandler
	hasher                        hashing.Hasher
	marshalizer                   marshal.Marshalizer
	blsPrivateKey                 crypto.PrivateKey
	blsSingleSigner               crypto.SingleSigner
	keyGenerator                  crypto.KeyGenerator
	multiSignerContainer          cryptoCommon.MultiSignerContainer
	roundHandler                  consensus.RoundHandler
	shardCoordinator              sharding.Coordinator
	nodesCoordinator              nodesCoordinator.NodesCoordinator
	syncTimer                     ntp.SyncTimer
	epochStartRegistrationHandler epochStart.RegistrationHandler
	antifloodHandler              consensus.P2PAntifloodHandler
	peerHonestyHandler            consensus.PeerHonestyHandler
	headerSigVerifier             consensus.HeaderSigVerifier
	fallbackHeaderValidator       consensus.FallbackHeaderValidator
	nodeRedundancyHandler         consensus.NodeRedundancyHandler
	scheduledProcessor            consensus.ScheduledProcessor
	messageSigningHandler         consensus.P2PSigningHandler
	signatureHandler              consensus.SignatureHandler
}

// ConsensusCoreArgs store all arguments that are needed to create a ConsensusCore object
type ConsensusCoreArgs struct {
	BlockChain                    data.ChainHandler
	BlockProcessor                process.BlockProcessor
	Bootstrapper                  process.Bootstrapper
	BroadcastMessenger            consensus.BroadcastMessenger
	ChronologyHandler             consensus.ChronologyHandler
	Hasher                        hashing.Hasher
	Marshalizer                   marshal.Marshalizer
	BlsPrivateKey                 crypto.PrivateKey
	BlsSingleSigner               crypto.SingleSigner
	KeyGenerator                  crypto.KeyGenerator
	MultiSignerContainer          cryptoCommon.MultiSignerContainer
	RoundHandler                  consensus.RoundHandler
	ShardCoordinator              sharding.Coordinator
	NodesCoordinator              nodesCoordinator.NodesCoordinator
	SyncTimer                     ntp.SyncTimer
	EpochStartRegistrationHandler epochStart.RegistrationHandler
	AntifloodHandler              consensus.P2PAntifloodHandler
	PeerHonestyHandler            consensus.PeerHonestyHandler
	HeaderSigVerifier             consensus.HeaderSigVerifier
	FallbackHeaderValidator       consensus.FallbackHeaderValidator
	NodeRedundancyHandler         consensus.NodeRedundancyHandler
	ScheduledProcessor            consensus.ScheduledProcessor
	MessageSigningHandler         consensus.P2PSigningHandler
	SignatureHandler              consensus.SignatureHandler
}

// NewConsensusCore creates a new ConsensusCore instance
func NewConsensusCore(
	args *ConsensusCoreArgs,
) (*ConsensusCore, error) {
	consensusCore := &ConsensusCore{
		blockChain:                    args.BlockChain,
		blockProcessor:                args.BlockProcessor,
		bootstrapper:                  args.Bootstrapper,
		broadcastMessenger:            args.BroadcastMessenger,
		chronologyHandler:             args.ChronologyHandler,
		hasher:                        args.Hasher,
		marshalizer:                   args.Marshalizer,
		blsPrivateKey:                 args.BlsPrivateKey,
		blsSingleSigner:               args.BlsSingleSigner,
		keyGenerator:                  args.KeyGenerator,
		multiSignerContainer:          args.MultiSignerContainer,
		roundHandler:                  args.RoundHandler,
		shardCoordinator:              args.ShardCoordinator,
		nodesCoordinator:              args.NodesCoordinator,
		syncTimer:                     args.SyncTimer,
		epochStartRegistrationHandler: args.EpochStartRegistrationHandler,
		antifloodHandler:              args.AntifloodHandler,
		peerHonestyHandler:            args.PeerHonestyHandler,
		headerSigVerifier:             args.HeaderSigVerifier,
		fallbackHeaderValidator:       args.FallbackHeaderValidator,
		nodeRedundancyHandler:         args.NodeRedundancyHandler,
		scheduledProcessor:            args.ScheduledProcessor,
		messageSigningHandler:         args.MessageSigningHandler,
		signatureHandler:              args.SignatureHandler,
	}

	err := ValidateConsensusCore(consensusCore)
	if err != nil {
		return nil, err
	}

	return consensusCore, nil
}

// Blockchain gets the ChainHandler stored in the ConsensusCore
func (cc *ConsensusCore) Blockchain() data.ChainHandler {
	return cc.blockChain
}

// GetAntiFloodHandler will return the antiflood handler which will be used in subrounds
func (cc *ConsensusCore) GetAntiFloodHandler() consensus.P2PAntifloodHandler {
	return cc.antifloodHandler
}

// BlockProcessor gets the BlockProcessor stored in the ConsensusCore
func (cc *ConsensusCore) BlockProcessor() process.BlockProcessor {
	return cc.blockProcessor
}

// BootStrapper gets the Bootstrapper stored in the ConsensusCore
func (cc *ConsensusCore) BootStrapper() process.Bootstrapper {
	return cc.bootstrapper
}

// BroadcastMessenger gets the BroadcastMessenger stored in the ConsensusCore
func (cc *ConsensusCore) BroadcastMessenger() consensus.BroadcastMessenger {
	return cc.broadcastMessenger
}

// Chronology gets the ChronologyHandler stored in the ConsensusCore
func (cc *ConsensusCore) Chronology() consensus.ChronologyHandler {
	return cc.chronologyHandler
}

// Hasher gets the Hasher stored in the ConsensusCore
func (cc *ConsensusCore) Hasher() hashing.Hasher {
	return cc.hasher
}

// Marshalizer gets the Marshalizer stored in the ConsensusCore
func (cc *ConsensusCore) Marshalizer() marshal.Marshalizer {
	return cc.marshalizer
}

// MultiSignerContainer gets the MultiSignerContainer stored in the ConsensusCore
func (cc *ConsensusCore) MultiSignerContainer() cryptoCommon.MultiSignerContainer {
	return cc.multiSignerContainer
}

//RoundHandler gets the RoundHandler stored in the ConsensusCore
func (cc *ConsensusCore) RoundHandler() consensus.RoundHandler {
	return cc.roundHandler
}

// ShardCoordinator gets the ShardCoordinator stored in the ConsensusCore
func (cc *ConsensusCore) ShardCoordinator() sharding.Coordinator {
	return cc.shardCoordinator
}

//SyncTimer gets the SyncTimer stored in the ConsensusCore
func (cc *ConsensusCore) SyncTimer() ntp.SyncTimer {
	return cc.syncTimer
}

// NodesCoordinator gets the NodesCoordinator stored in the ConsensusCore
func (cc *ConsensusCore) NodesCoordinator() nodesCoordinator.NodesCoordinator {
	return cc.nodesCoordinator
}

// EpochStartRegistrationHandler returns the epoch start registration handler
func (cc *ConsensusCore) EpochStartRegistrationHandler() epochStart.RegistrationHandler {
	return cc.epochStartRegistrationHandler
}

// PrivateKey returns the BLS private key stored in the ConsensusStore
func (cc *ConsensusCore) PrivateKey() crypto.PrivateKey {
	return cc.blsPrivateKey
}

// SingleSigner returns the bls single signer stored in the ConsensusStore
func (cc *ConsensusCore) SingleSigner() crypto.SingleSigner {
	return cc.blsSingleSigner
}

// KeyGenerator returns the bls key generator stored in the ConsensusStore
func (cc *ConsensusCore) KeyGenerator() crypto.KeyGenerator {
	return cc.keyGenerator
}

// PeerHonestyHandler will return the peer honesty handler which will be used in subrounds
func (cc *ConsensusCore) PeerHonestyHandler() consensus.PeerHonestyHandler {
	return cc.peerHonestyHandler
}

// HeaderSigVerifier returns the sig verifier handler which will be used in subrounds
func (cc *ConsensusCore) HeaderSigVerifier() consensus.HeaderSigVerifier {
	return cc.headerSigVerifier
}

// FallbackHeaderValidator will return the fallback header validator which will be used in subrounds
func (cc *ConsensusCore) FallbackHeaderValidator() consensus.FallbackHeaderValidator {
	return cc.fallbackHeaderValidator
}

// NodeRedundancyHandler will return the node redundancy handler which will be used in subrounds
func (cc *ConsensusCore) NodeRedundancyHandler() consensus.NodeRedundancyHandler {
	return cc.nodeRedundancyHandler
}

// ScheduledProcessor will return the scheduled processor
func (cc *ConsensusCore) ScheduledProcessor() consensus.ScheduledProcessor {
	return cc.scheduledProcessor
}

// MessageSigningHandler will return the message signing handler
func (cc *ConsensusCore) MessageSigningHandler() consensus.P2PSigningHandler {
	return cc.messageSigningHandler
}

// SignatureHandler will return the signature handler component
func (cc *ConsensusCore) SignatureHandler() consensus.SignatureHandler {
	return cc.signatureHandler
}

// IsInterfaceNil returns true if there is no value under the interface
func (cc *ConsensusCore) IsInterfaceNil() bool {
	return cc == nil
}
