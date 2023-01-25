package disabled

import "github.com/multiversx/mx-chain-core-go/core"

type disabledPeersRatingHandler struct {
}

// NewDisabledPeersRatingHandler returns a new instance of disabledPeersRatingHandler
func NewDisabledPeersRatingHandler() *disabledPeersRatingHandler {
	return &disabledPeersRatingHandler{}
}

// AddPeers does nothing as it is disabled
func (dprs *disabledPeersRatingHandler) AddPeers(_ []core.PeerID) {
}

// IncreaseRating does nothing as it is disabled
func (dprs *disabledPeersRatingHandler) IncreaseRating(_ core.PeerID) {
}

// DecreaseRating does nothing as it is disabled
func (dprs *disabledPeersRatingHandler) DecreaseRating(_ core.PeerID) {
}

// GetTopRatedPeersFromList returns the provided peers list as it is disabled
func (dprs *disabledPeersRatingHandler) GetTopRatedPeersFromList(peers []core.PeerID, _ int) []core.PeerID {
	return peers
}

// Close returns the nil as it is disabled
func (dprs *disabledPeersRatingHandler) Close() error {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (dprs *disabledPeersRatingHandler) IsInterfaceNil() bool {
	return dprs == nil
}
