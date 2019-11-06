package config

// EconomicsAddresses will hold economics addresses
type EconomicsAddresses struct {
	CommunityAddress string
	BurnAddress      string
}

// RewardsSettings will hold economics rewards settings
type RewardsSettings struct {
	RewardsValue        string
	CommunityPercentage float64
	LeaderPercentage    float64
	BurnPercentage      float64
}

// FeeSettings will hold economics fee settings
type FeeSettings struct {
	MinGasPrice string
	MinGasLimit string
}

// RatingSettings will hold rating settings
type RatingSettings struct {
	StartRating                     uint64
	MaxRating                       uint64
	MinRating                       uint64
	IncreaseRatingStep              uint64
	DecreaseRatingStep              uint64
	ProposerExtraIncreaseRatingStep uint64
	ProposerExtraDecreaseRatingStep uint64
}

// ConfigEconomics will hold economics config
type ConfigEconomics struct {
	EconomicsAddresses EconomicsAddresses
	RewardsSettings    RewardsSettings
	FeeSettings        FeeSettings
	RatingSettings     RatingSettings
}
