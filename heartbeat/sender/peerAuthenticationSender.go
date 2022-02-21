package sender

import (
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/batch"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
)

// argPeerAuthenticationSender represents the arguments for the peer authentication sender
type argPeerAuthenticationSender struct {
	argBaseSender
	peerSignatureHandler crypto.PeerSignatureHandler
	privKey              crypto.PrivateKey
	redundancyHandler    heartbeat.NodeRedundancyHandler
}

type peerAuthenticationSender struct {
	baseSender
	peerSignatureHandler crypto.PeerSignatureHandler
	redundancy           heartbeat.NodeRedundancyHandler
	privKey              crypto.PrivateKey
	publicKey            crypto.PublicKey
	observerPublicKey    crypto.PublicKey
}

// newPeerAuthenticationSender will create a new instance of type peerAuthenticationSender
func newPeerAuthenticationSender(args argPeerAuthenticationSender) (*peerAuthenticationSender, error) {
	err := checkPeerAuthenticationSenderArgs(args)
	if err != nil {
		return nil, err
	}

	redundancyHandler := args.redundancyHandler
	sender := &peerAuthenticationSender{
		baseSender:           createBaseSender(args.argBaseSender),
		peerSignatureHandler: args.peerSignatureHandler,
		redundancy:           redundancyHandler,
		privKey:              args.privKey,
		publicKey:            args.privKey.GeneratePublic(),
		observerPublicKey:    redundancyHandler.ObserverPrivateKey().GeneratePublic(),
	}

	return sender, nil
}

func checkPeerAuthenticationSenderArgs(args argPeerAuthenticationSender) error {
	err := checkBaseSenderArgs(args.argBaseSender)
	if err != nil {
		return err
	}
	if check.IfNil(args.peerSignatureHandler) {
		return heartbeat.ErrNilPeerSignatureHandler
	}
	if check.IfNil(args.privKey) {
		return heartbeat.ErrNilPrivateKey
	}
	if check.IfNil(args.redundancyHandler) {
		return heartbeat.ErrNilRedundancyHandler
	}

	return nil
}

// Execute will handle the execution of a cycle in which the peer authentication message will be sent
func (sender *peerAuthenticationSender) Execute() {
	duration := sender.computeRandomDuration()
	err := sender.execute()
	if err != nil {
		duration = sender.timeBetweenSendsWhenError
		log.Error("error sending peer authentication message", "error", err, "next send will be in", duration)
	} else {
		log.Debug("peer authentication message sent", "next send will be in", duration)
	}

	sender.CreateNewTimer(duration)
}

func (sender *peerAuthenticationSender) execute() error {
	sk, pk := sender.getCurrentPrivateAndPublicKeys()

	msg := &heartbeat.PeerAuthentication{
		Pid: sender.messenger.ID().Bytes(),
	}
	payload := &heartbeat.Payload{
		Timestamp:       time.Now().Unix(),
		HardforkMessage: "", // TODO add the hardfork message, if required
	}
	payloadBytes, err := sender.marshaller.Marshal(payload)
	if err != nil {
		return err
	}
	msg.Payload = payloadBytes
	msg.PayloadSignature, err = sender.messenger.Sign(payloadBytes)
	if err != nil {
		return err
	}

	msg.Pubkey, err = pk.ToByteArray()
	if err != nil {
		return err
	}

	msg.Signature, err = sender.peerSignatureHandler.GetPeerSignature(sk, msg.Pid)
	if err != nil {
		return err
	}

	msgBytes, err := sender.marshaller.Marshal(msg)
	if err != nil {
		return err
	}

	b := batch.Batch{
		Data: make([][]byte, 1),
	}
	b.Data[0] = msgBytes
	data, err := sender.marshaller.Marshal(b)
	if err != nil {
		return err
	}

	sender.messenger.Broadcast(sender.topic, data)

	return nil
}

func (sender *peerAuthenticationSender) getCurrentPrivateAndPublicKeys() (crypto.PrivateKey, crypto.PublicKey) {
	shouldUseOriginalKeys := !sender.redundancy.IsRedundancyNode() || (sender.redundancy.IsRedundancyNode() && !sender.redundancy.IsMainMachineActive())
	if shouldUseOriginalKeys {
		return sender.privKey, sender.publicKey
	}

	return sender.redundancy.ObserverPrivateKey(), sender.observerPublicKey
}

// IsInterfaceNil returns true if there is no value under the interface
func (sender *peerAuthenticationSender) IsInterfaceNil() bool {
	return sender == nil
}
