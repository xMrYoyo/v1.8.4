package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	dataVm "github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/multiversx/mx-chain-go/config"
	testsChainSimulator "github.com/multiversx/mx-chain-go/integrationTests/chainSimulator"
	"github.com/multiversx/mx-chain-go/integrationTests/chainSimulator/staking"
	"github.com/multiversx/mx-chain-go/integrationTests/vm/txsFee"
	"github.com/multiversx/mx-chain-go/integrationTests/vm/txsFee/utils"
	"github.com/multiversx/mx-chain-go/node/chainSimulator"
	"github.com/multiversx/mx-chain-go/node/chainSimulator/components/api"
	"github.com/multiversx/mx-chain-go/node/chainSimulator/configs"
	"github.com/multiversx/mx-chain-go/node/chainSimulator/dtos"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-go/vm"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/require"
)

const (
	defaultPathToInitialConfig = "../../../cmd/node/config/"

	minGasPrice = 1000000000
)

var log = logger.GetOrCreate("integrationTests/chainSimulator/vm")

// Test scenario #1
//
// Initial setup: Create fungible, NFT,  SFT and metaESDT tokens
//  (before the activation of DynamicEsdtFlag)
//
// 1.check that the metadata for all tokens is saved on the system account
// 2. wait for DynamicEsdtFlag activation
// 3. transfer the tokens to another account
// 4. check that the metadata for all tokens is saved on the system account
// 5. make an updateTokenID@tokenID function call on the ESDTSystem SC for all token types
// 6. check that the metadata for all tokens is saved on the system account
// 7. transfer the tokens to another account
// 8. check that the metaData for the NFT was removed from the system account and moved to the user account
// 9. check that the metaData for the rest of the tokens is still present on the system account and not on the userAccount
// 10. do the test for both intra and cross shard txs
func TestChainSimulator_CheckTokensMetadata_TransferTokens(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	t.Run("transfer and check all tokens - intra shard", func(t *testing.T) {
		transferAndCheckTokensMetaData(t, false)
	})

	t.Run("transfer and check all tokens - intra shard", func(t *testing.T) {
		transferAndCheckTokensMetaData(t, true)
	})
}

func transferAndCheckTokensMetaData(t *testing.T, isCrossShard bool) {
	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(4)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	addrs := createAddresses(t, cs, isCrossShard)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch) - 1)
	require.Nil(t, err)

	log.Info("Initial setup: Create fungible, NFT, SFT and metaESDT tokens (before the activation of DynamicEsdtFlag)")

	// issue metaESDT
	metaESDTTicker := []byte("METATTICKER")
	tx := issueMetaESDTTx(0, addrs[0].Bytes, metaESDTTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	metaESDTTokenID := txResult.Logs.Events[0].Topics[0]

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleTransfer),
	}
	setAddressEsdtRoles(t, cs, addrs[0], metaESDTTokenID, roles)

	log.Info("Issued metaESDT token id", "tokenID", string(metaESDTTokenID))

	// issue fungible
	fungibleTicker := []byte("FUNGIBLETICKER")
	tx = issueTx(1, addrs[0].Bytes, fungibleTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	fungibleTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], fungibleTokenID, roles)

	log.Info("Issued fungible token id", "tokenID", string(fungibleTokenID))

	// issue NFT
	nftTicker := []byte("NFTTICKER")
	tx = issueNonFungibleTx(2, addrs[0].Bytes, nftTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	// issue SFT
	sftTicker := []byte("SFTTICKER")
	tx = issueSemiFungibleTx(3, addrs[0].Bytes, sftTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	sftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], sftTokenID, roles)

	log.Info("Issued SFT token id", "tokenID", string(sftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	sftMetaData := txsFee.GetDefaultMetaData()
	sftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	esdtMetaData := txsFee.GetDefaultMetaData()
	esdtMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	fungibleMetaData := txsFee.GetDefaultMetaData()
	fungibleMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tokenIDs := [][]byte{
		nftTokenID,
		sftTokenID,
		metaESDTTokenID,
		// fungibleTokenID,
	}

	tokensMetadata := []*txsFee.MetaData{
		nftMetaData,
		sftMetaData,
		esdtMetaData,
		// fungibleMetaData,
	}

	nonce := uint64(4)
	for i := range tokenIDs {
		tx = nftCreateTx(nonce, addrs[0].Bytes, tokenIDs[i], tokensMetadata[i])

		txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
		require.Nil(t, err)
		require.NotNil(t, txResult)
		require.Equal(t, "success", txResult.Status.String())

		nonce++
	}

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Step 1. check that the metadata for all tokens is saved on the system account")

	checkMetaData(t, cs, core.SystemAccountAddress, nftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, sftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, metaESDTTokenID, esdtMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, fungibleTokenID, fungibleMetaData)

	log.Info("Step 2. wait for DynamicEsdtFlag activation")

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	log.Info("Step 3. transfer the tokens to another account")

	for _, tokenID := range tokenIDs {
		log.Info("transfering token id", "tokenID", tokenID)

		tx = esdtNFTTransferTx(nonce, addrs[0].Bytes, addrs[1].Bytes, tokenID)
		txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
		require.Nil(t, err)
		require.NotNil(t, txResult)
		require.Equal(t, "success", txResult.Status.String())

		nonce++
	}

	log.Info("Step 4. check that the metadata for all tokens is saved on the system account")

	checkMetaData(t, cs, core.SystemAccountAddress, nftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, sftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, metaESDTTokenID, esdtMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, fungibleTokenID, fungibleMetaData)

	log.Info("Step 5. make an updateTokenID@tokenID function call on the ESDTSystem SC for all token types")

	for _, tokenID := range tokenIDs {
		tx = updateTokenIDTx(nonce, addrs[0].Bytes, tokenID)

		log.Info("updating token id", "tokenID", tokenID)

		txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
		require.Nil(t, err)
		require.NotNil(t, txResult)
		require.Equal(t, "success", txResult.Status.String())

		nonce++
	}

	log.Info("Step 6. check that the metadata for all tokens is saved on the system account")

	checkMetaData(t, cs, core.SystemAccountAddress, nftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, sftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, metaESDTTokenID, esdtMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, fungibleTokenID, fungibleMetaData)

	log.Info("Step 7. transfer the tokens to another account")

	nonce = uint64(0)
	for _, tokenID := range tokenIDs {
		log.Info("transfering token id", "tokenID", tokenID)

		tx = esdtNFTTransferTx(nonce, addrs[1].Bytes, addrs[2].Bytes, tokenID)
		txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
		require.Nil(t, err)
		require.NotNil(t, txResult)
		require.Equal(t, "success", txResult.Status.String())

		nonce++
	}

	log.Info("Step 8. check that the metaData for the NFT was removed from the system account and moved to the user account")

	checkMetaData(t, cs, addrs[2].Bytes, nftTokenID, nftMetaData)

	log.Info("Step 9. check that the metaData for the rest of the tokens is still present on the system account and not on the userAccount")

	checkMetaData(t, cs, core.SystemAccountAddress, sftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, metaESDTTokenID, esdtMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, fungibleTokenID, fungibleMetaData)
}

func createAddresses(
	t *testing.T,
	cs testsChainSimulator.ChainSimulator,
	isCrossShard bool,
) []dtos.WalletAddress {
	var shardIDs []uint32
	if !isCrossShard {
		shardIDs = []uint32{1, 1, 1}
	} else {
		shardIDs = []uint32{0, 1, 2}
	}

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(shardIDs[0], mintValue)
	require.Nil(t, err)

	address2, err := cs.GenerateAndMintWalletAddress(shardIDs[1], mintValue)
	require.Nil(t, err)

	address3, err := cs.GenerateAndMintWalletAddress(shardIDs[2], mintValue)
	require.Nil(t, err)

	return []dtos.WalletAddress{address, address2, address3}
}

func checkMetaData(
	t *testing.T,
	cs testsChainSimulator.ChainSimulator,
	addressBytes []byte,
	token []byte,
	expectedMetaData *txsFee.MetaData,
) {
	shardID := cs.GetNodeHandler(0).GetProcessComponents().ShardCoordinator().ComputeId(addressBytes)

	retrievedMetaData := getMetaDataFromAcc(t, cs, addressBytes, token, shardID)

	require.Equal(t, expectedMetaData.Nonce, []byte(hex.EncodeToString(big.NewInt(int64(retrievedMetaData.Nonce)).Bytes())))
	require.Equal(t, expectedMetaData.Name, []byte(hex.EncodeToString(retrievedMetaData.Name)))
	require.Equal(t, expectedMetaData.Royalties, []byte(hex.EncodeToString(big.NewInt(int64(retrievedMetaData.Royalties)).Bytes())))
	require.Equal(t, expectedMetaData.Hash, []byte(hex.EncodeToString(retrievedMetaData.Hash)))
	for i, uri := range expectedMetaData.Uris {
		require.Equal(t, uri, []byte(hex.EncodeToString(retrievedMetaData.URIs[i])))
	}
	require.Equal(t, expectedMetaData.Attributes, []byte(hex.EncodeToString(retrievedMetaData.Attributes)))
}

func esdtNFTTransferTx(nonce uint64, sndAdr, rcvAddr, token []byte) *transaction.Transaction {
	tx := utils.CreateESDTNFTTransferTx(
		nonce,
		sndAdr,
		rcvAddr,
		token,
		1,
		big.NewInt(1),
		minGasPrice,
		10_000_000,
		"",
	)
	tx.Version = 1
	tx.Signature = []byte("dummySig")
	tx.ChainID = []byte(configs.ChainID)

	return tx
}

func issueTx(nonce uint64, sndAdr []byte, ticker []byte, baseIssuingCost string) *transaction.Transaction {
	callValue, _ := big.NewInt(0).SetString(baseIssuingCost, 10)

	txDataField := bytes.Join(
		[][]byte{
			[]byte("issue"),
			[]byte(hex.EncodeToString([]byte("asdname1"))),
			[]byte(hex.EncodeToString(ticker)),
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
		},
		[]byte("@"),
	)

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   core.ESDTSCAddress,
		GasLimit:  100_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     callValue,
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func issueMetaESDTTx(nonce uint64, sndAdr []byte, ticker []byte, baseIssuingCost string) *transaction.Transaction {
	callValue, _ := big.NewInt(0).SetString(baseIssuingCost, 10)

	txDataField := bytes.Join(
		[][]byte{
			[]byte("registerMetaESDT"),
			[]byte(hex.EncodeToString([]byte("asdname"))),
			[]byte(hex.EncodeToString(ticker)),
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
		},
		[]byte("@"),
	)

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   core.ESDTSCAddress,
		GasLimit:  100_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     callValue,
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func issueNonFungibleTx(nonce uint64, sndAdr []byte, ticker []byte, baseIssuingCost string) *transaction.Transaction {
	callValue, _ := big.NewInt(0).SetString(baseIssuingCost, 10)

	txDataField := bytes.Join(
		[][]byte{
			[]byte("issueNonFungible"),
			[]byte(hex.EncodeToString([]byte("asdname"))),
			[]byte(hex.EncodeToString(ticker)),
		},
		[]byte("@"),
	)

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   core.ESDTSCAddress,
		GasLimit:  100_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     callValue,
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func issueSemiFungibleTx(nonce uint64, sndAdr []byte, ticker []byte, baseIssuingCost string) *transaction.Transaction {
	callValue, _ := big.NewInt(0).SetString(baseIssuingCost, 10)

	txDataField := bytes.Join(
		[][]byte{
			[]byte("issueSemiFungible"),
			[]byte(hex.EncodeToString([]byte("asdname"))),
			[]byte(hex.EncodeToString(ticker)),
		},
		[]byte("@"),
	)

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   core.ESDTSCAddress,
		GasLimit:  100_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     callValue,
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func updateTokenIDTx(nonce uint64, sndAdr []byte, tokenID []byte) *transaction.Transaction {
	txDataField := []byte("updateTokenID@" + hex.EncodeToString(tokenID))

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   vm.ESDTSCAddress,
		GasLimit:  100_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func nftCreateTx(
	nonce uint64,
	sndAdr []byte,
	tokenID []byte,
	metaData *txsFee.MetaData,
) *transaction.Transaction {
	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.BuiltInFunctionESDTNFTCreate),
			[]byte(hex.EncodeToString(tokenID)),
			[]byte(hex.EncodeToString(big.NewInt(1).Bytes())), // quantity
			metaData.Name,
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
			metaData.Hash,
			metaData.Attributes,
			metaData.Uris[0],
			metaData.Uris[1],
			metaData.Uris[2],
		},
		[]byte("@"),
	)

	return &transaction.Transaction{
		Nonce:     nonce,
		SndAddr:   sndAdr,
		RcvAddr:   sndAdr,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}
}

func executeQuery(cs testsChainSimulator.ChainSimulator, shardID uint32, scAddress []byte, funcName string, args [][]byte) (*dataVm.VMOutputApi, error) {
	output, _, err := cs.GetNodeHandler(shardID).GetFacadeHandler().ExecuteSCQuery(&process.SCQuery{
		ScAddress: scAddress,
		FuncName:  funcName,
		Arguments: args,
	})
	return output, err
}

func getMetaDataFromAcc(
	t *testing.T,
	cs testsChainSimulator.ChainSimulator,
	addressBytes []byte,
	token []byte,
	shardID uint32,
) *esdt.MetaData {
	account, err := cs.GetNodeHandler(shardID).GetStateComponents().AccountsAdapter().LoadAccount(addressBytes)
	require.Nil(t, err)
	userAccount, ok := account.(state.UserAccountHandler)
	require.True(t, ok)

	baseEsdtKeyPrefix := core.ProtectedKeyPrefix + core.ESDTKeyIdentifier
	key := append([]byte(baseEsdtKeyPrefix), token...)

	key2 := append(key, big.NewInt(0).SetUint64(1).Bytes()...)
	esdtDataBytes, _, err := userAccount.RetrieveValue(key2)
	require.Nil(t, err)

	esdtData := &esdt.ESDigitalToken{}
	err = cs.GetNodeHandler(shardID).GetCoreComponents().InternalMarshalizer().Unmarshal(esdtData, esdtDataBytes)
	require.Nil(t, err)
	require.NotNil(t, esdtData.TokenMetaData)

	return esdtData.TokenMetaData
}

func setAddressEsdtRoles(
	t *testing.T,
	cs testsChainSimulator.ChainSimulator,
	address dtos.WalletAddress,
	token []byte,
	roles [][]byte,
) {
	marshaller := cs.GetNodeHandler(0).GetCoreComponents().InternalMarshalizer()

	rolesKey := append([]byte(core.ProtectedKeyPrefix), append([]byte(core.ESDTRoleIdentifier), []byte(core.ESDTKeyIdentifier)...)...)
	rolesKey = append(rolesKey, token...)

	rolesData := &esdt.ESDTRoles{
		Roles: roles,
	}

	rolesDataBytes, err := marshaller.Marshal(rolesData)
	require.Nil(t, err)

	keys := make(map[string]string)
	keys[hex.EncodeToString(rolesKey)] = hex.EncodeToString(rolesDataBytes)

	err = cs.SetStateMultiple([]*dtos.AddressState{
		{
			Address: address.Bech32,
			Balance: "10000000000000000000000",
			Keys:    keys,
		},
	})
	require.Nil(t, err)
}

// Test scenario #3
//
// Initial setup: Create fungible, NFT,  SFT and metaESDT tokens
// (after the activation of DynamicEsdtFlag)
//
// 1. check that the metaData for the NFT was saved in the user account and not on the system account
// 2. check that the metaData for the other token types is saved on the system account and not at the user account level
func TestChainSimulator_CreateTokensAfterActivation(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	addrs := createAddresses(t, cs, false)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create fungible, NFT,  SFT and metaESDT tokens (after the activation of DynamicEsdtFlag)")

	// issue metaESDT
	metaESDTTicker := []byte("METATTICKER")
	tx := issueMetaESDTTx(0, addrs[0].Bytes, metaESDTTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	metaESDTTokenID := txResult.Logs.Events[0].Topics[0]

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleTransfer),
	}
	setAddressEsdtRoles(t, cs, addrs[0], metaESDTTokenID, roles)

	log.Info("Issued metaESDT token id", "tokenID", string(metaESDTTokenID))

	// issue fungible
	fungibleTicker := []byte("FUNGIBLETICKER")
	tx = issueTx(1, addrs[0].Bytes, fungibleTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	fungibleTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], fungibleTokenID, roles)

	log.Info("Issued fungible token id", "tokenID", string(fungibleTokenID))

	// issue NFT
	nftTicker := []byte("NFTTICKER")
	tx = issueNonFungibleTx(2, addrs[0].Bytes, nftTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	// issue SFT
	sftTicker := []byte("SFTTICKER")
	tx = issueSemiFungibleTx(3, addrs[0].Bytes, sftTicker, baseIssuingCost)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	sftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, addrs[0], sftTokenID, roles)

	log.Info("Issued SFT token id", "tokenID", string(sftTokenID))

	tokenIDs := [][]byte{
		nftTokenID,
		sftTokenID,
		metaESDTTokenID,
		// fungibleTokenID,
	}

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	sftMetaData := txsFee.GetDefaultMetaData()
	sftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	esdtMetaData := txsFee.GetDefaultMetaData()
	esdtMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	fungibleMetaData := txsFee.GetDefaultMetaData()
	fungibleMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tokensMetadata := []*txsFee.MetaData{
		nftMetaData,
		sftMetaData,
		esdtMetaData,
		// fungibleMetaData,
	}

	nonce := uint64(4)
	for i := range tokenIDs {
		tx = nftCreateTx(nonce, addrs[0].Bytes, tokenIDs[i], tokensMetadata[i])

		txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
		require.Nil(t, err)
		require.NotNil(t, txResult)
		require.Equal(t, "success", txResult.Status.String())

		nonce++
	}

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Step 1. check that the metaData for the NFT was saved in the user account and not on the system account")

	checkMetaData(t, cs, addrs[0].Bytes, nftTokenID, nftMetaData)

	log.Info("Step 2. check that the metaData for the other token types is saved on the system account and not at the user account level")

	checkMetaData(t, cs, core.SystemAccountAddress, sftTokenID, nftMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, metaESDTTokenID, esdtMetaData)
	checkMetaData(t, cs, core.SystemAccountAddress, fungibleTokenID, fungibleMetaData)
}

// Test scenario #4
//
// Initial setup: Create NFT
//
// Call ESDTMetaDataRecreate to rewrite the meta data for the nft
// (The sender must have the ESDTMetaDataRecreate role)
func TestChainSimulator_NFT_ESDTMetaDataRecreate(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create NFT")

	nftTicker := []byte("NFTTICKER")
	tx := issueNonFungibleTx(0, address.Bytes, nftTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTMetaDataRecreate),
	}

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tx = nftCreateTx(1, address.Bytes, nftTokenID, nftMetaData)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Call ESDTMetaDataRecreate to rewrite the meta data for the nft")

	nonce := []byte(hex.EncodeToString(big.NewInt(1).Bytes()))
	nftMetaData.Name = []byte(hex.EncodeToString([]byte("name2")))
	nftMetaData.Hash = []byte(hex.EncodeToString([]byte("hash2")))
	nftMetaData.Attributes = []byte(hex.EncodeToString([]byte("attributes2")))

	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.ESDTMetaDataRecreate),
			[]byte(hex.EncodeToString(nftTokenID)),
			nonce,
			nftMetaData.Name,
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
			nftMetaData.Hash,
			nftMetaData.Attributes,
			nftMetaData.Uris[0],
			nftMetaData.Uris[1],
			nftMetaData.Uris[2],
		},
		[]byte("@"),
	)

	tx = &transaction.Transaction{
		Nonce:     2,
		SndAddr:   address.Bytes,
		RcvAddr:   address.Bytes,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)

	require.Equal(t, "success", txResult.Status.String())

	checkMetaData(t, cs, address.Bytes, nftTokenID, nftMetaData)
}

// Test scenario #5
//
// Initial setup: Create NFT
//
// Call ESDTMetaDataUpdate to update some of the meta data parameters
// (The sender must have the ESDTRoleNFTUpdate role)
func TestChainSimulator_NFT_ESDTMetaDataUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create  NFT")

	nftTicker := []byte("NFTTICKER")
	tx := issueNonFungibleTx(0, address.Bytes, nftTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleNFTUpdate),
	}

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tx = nftCreateTx(1, address.Bytes, nftTokenID, nftMetaData)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	log.Info("Call ESDTMetaDataRecreate to rewrite the meta data for the nft")

	nonce := []byte(hex.EncodeToString(big.NewInt(1).Bytes()))
	nftMetaData.Name = []byte(hex.EncodeToString([]byte("name2")))
	nftMetaData.Hash = []byte(hex.EncodeToString([]byte("hash2")))
	nftMetaData.Attributes = []byte(hex.EncodeToString([]byte("attributes2")))

	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.ESDTMetaDataUpdate),
			[]byte(hex.EncodeToString(nftTokenID)),
			nonce,
			nftMetaData.Name,
			[]byte(hex.EncodeToString(big.NewInt(10).Bytes())),
			nftMetaData.Hash,
			nftMetaData.Attributes,
			nftMetaData.Uris[0],
			nftMetaData.Uris[1],
			nftMetaData.Uris[2],
		},
		[]byte("@"),
	)

	tx = &transaction.Transaction{
		Nonce:     2,
		SndAddr:   address.Bytes,
		RcvAddr:   address.Bytes,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)

	require.Equal(t, "success", txResult.Status.String())

	checkMetaData(t, cs, address.Bytes, nftTokenID, nftMetaData)
}

// Test scenario #6
//
// Initial setup: Create NFT
//
// Call ESDTModifyCreator and check that the creator was modified
// (The sender must have the ESDTRoleModifyCreator role)
func TestChainSimulator_NFT_ESDTModifyCreator(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create NFT")

	nftTicker := []byte("NFTTICKER")
	tx := issueNonFungibleTx(0, address.Bytes, nftTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleNFTUpdate),
	}

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tx = nftCreateTx(1, address.Bytes, nftTokenID, nftMetaData)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	log.Info("Call ESDTModifyCreator and check that the creator was modified")

	newCreatorAddress, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	roles = [][]byte{
		[]byte(core.ESDTRoleModifyCreator),
	}
	setAddressEsdtRoles(t, cs, newCreatorAddress, nftTokenID, roles)

	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.ESDTModifyCreator),
			[]byte(hex.EncodeToString(nftTokenID)),
			nftMetaData.Nonce,
		},
		[]byte("@"),
	)

	tx = &transaction.Transaction{
		Nonce:     0,
		SndAddr:   newCreatorAddress.Bytes,
		RcvAddr:   newCreatorAddress.Bytes,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)

	fmt.Println(txResult)
	fmt.Println(txResult.Logs.Events[0])
	fmt.Println(string(txResult.Logs.Events[0].Topics[0]))
	fmt.Println(string(txResult.Logs.Events[0].Topics[1]))

	require.Equal(t, "success", txResult.Status.String())

	shardID := cs.GetNodeHandler(0).GetProcessComponents().ShardCoordinator().ComputeId(address.Bytes)
	retrievedMetaData := getMetaDataFromAcc(t, cs, address.Bytes, nftTokenID, shardID)

	require.Equal(t, newCreatorAddress.Bytes, retrievedMetaData.Creator)
}

// Test scenario #7
//
// Initial setup: Create NFT
//
// Call ESDTSetNewURIs and check that the new URIs were set for the NFT
// (The sender must have the ESDTRoleSetNewURI role)
func TestChainSimulator_NFT_ESDTSetNewURIs(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create NFT")

	nftTicker := []byte("NFTTICKER")
	tx := issueNonFungibleTx(0, address.Bytes, nftTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleNFTUpdate),
	}

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tx = nftCreateTx(1, address.Bytes, nftTokenID, nftMetaData)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	log.Info("Call ESDTSetNewURIs and check that the new URIs were set for the NFT")

	roles = [][]byte{
		[]byte(core.ESDTRoleSetNewURI),
	}
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	nonce := []byte(hex.EncodeToString(big.NewInt(1).Bytes()))
	uris := [][]byte{
		[]byte(hex.EncodeToString([]byte("uri0"))),
		[]byte(hex.EncodeToString([]byte("uri1"))),
		[]byte(hex.EncodeToString([]byte("uri2"))),
	}

	expUris := [][]byte{
		[]byte("uri0"),
		[]byte("uri1"),
		[]byte("uri2"),
	}

	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.ESDTSetNewURIs),
			[]byte(hex.EncodeToString(nftTokenID)),
			nonce,
			uris[0],
			uris[1],
			uris[2],
		},
		[]byte("@"),
	)

	tx = &transaction.Transaction{
		Nonce:     2,
		SndAddr:   address.Bytes,
		RcvAddr:   address.Bytes,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)

	require.Equal(t, "success", txResult.Status.String())

	shardID := cs.GetNodeHandler(0).GetProcessComponents().ShardCoordinator().ComputeId(address.Bytes)
	retrievedMetaData := getMetaDataFromAcc(t, cs, address.Bytes, nftTokenID, shardID)

	require.Equal(t, expUris, retrievedMetaData.URIs)
}

// Test scenario #8
//
// Initial setup: Create NFT
//
// Call ESDTModifyRoyalties and check that the royalties were changed
// (The sender must have the ESDTRoleModifyRoyalties role)
func TestChainSimulator_NFT_ESDTModifyRoyalties(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	startTime := time.Now().Unix()
	roundDurationInMillis := uint64(6000)
	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	activationEpoch := uint32(2)

	baseIssuingCost := "1000"

	numOfShards := uint32(3)
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:      false,
		TempDir:                     t.TempDir(),
		PathToInitialConfig:         defaultPathToInitialConfig,
		NumOfShards:                 numOfShards,
		GenesisTimestamp:            startTime,
		RoundDurationInMillis:       roundDurationInMillis,
		RoundsPerEpoch:              roundsPerEpoch,
		ApiInterface:                api.NewNoApiInterface(),
		MinNodesPerShard:            3,
		MetaChainMinNodes:           3,
		ConsensusGroupSize:          1,
		MetaChainConsensusGroupSize: 1,
		NumNodesWaitingListMeta:     0,
		NumNodesWaitingListShard:    0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.EpochConfig.EnableEpochs.DynamicESDTEnableEpoch = activationEpoch
			cfg.SystemSCConfig.ESDTSystemSCConfig.BaseIssuingCost = baseIssuingCost
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	mintValue := big.NewInt(10)
	mintValue = mintValue.Mul(staking.OneEGLD, mintValue)

	address, err := cs.GenerateAndMintWalletAddress(core.AllShardId, mintValue)
	require.Nil(t, err)

	err = cs.GenerateBlocksUntilEpochIsReached(int32(activationEpoch))
	require.Nil(t, err)

	err = cs.GenerateBlocks(10)
	require.Nil(t, err)

	log.Info("Initial setup: Create NFT")

	nftTicker := []byte("NFTTICKER")
	tx := issueNonFungibleTx(0, address.Bytes, nftTicker, baseIssuingCost)

	txResult, err := cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	roles := [][]byte{
		[]byte(core.ESDTRoleNFTCreate),
		[]byte(core.ESDTRoleNFTUpdate),
	}

	nftTokenID := txResult.Logs.Events[0].Topics[0]
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	log.Info("Issued NFT token id", "tokenID", string(nftTokenID))

	nftMetaData := txsFee.GetDefaultMetaData()
	nftMetaData.Nonce = []byte(hex.EncodeToString(big.NewInt(1).Bytes()))

	tx = nftCreateTx(1, address.Bytes, nftTokenID, nftMetaData)

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)
	require.Equal(t, "success", txResult.Status.String())

	log.Info("Call ESDTModifyRoyalties and check that the royalties were changed")

	roles = [][]byte{
		[]byte(core.ESDTRoleModifyRoyalties),
	}
	setAddressEsdtRoles(t, cs, address, nftTokenID, roles)

	nonce := []byte(hex.EncodeToString(big.NewInt(1).Bytes()))
	royalties := []byte(hex.EncodeToString(big.NewInt(20).Bytes()))

	txDataField := bytes.Join(
		[][]byte{
			[]byte(core.ESDTModifyRoyalties),
			[]byte(hex.EncodeToString(nftTokenID)),
			nonce,
			royalties,
		},
		[]byte("@"),
	)

	tx = &transaction.Transaction{
		Nonce:     2,
		SndAddr:   address.Bytes,
		RcvAddr:   address.Bytes,
		GasLimit:  10_000_000,
		GasPrice:  minGasPrice,
		Signature: []byte("dummySig"),
		Data:      txDataField,
		Value:     big.NewInt(0),
		ChainID:   []byte(configs.ChainID),
		Version:   1,
	}

	txResult, err = cs.SendTxAndGenerateBlockTilTxIsExecuted(tx, staking.MaxNumOfBlockToGenerateWhenExecutingTx)
	require.Nil(t, err)
	require.NotNil(t, txResult)

	require.Equal(t, "success", txResult.Status.String())

	shardID := cs.GetNodeHandler(0).GetShardCoordinator().ComputeId(address.Bytes)
	retrievedMetaData := getMetaDataFromAcc(t, cs, address.Bytes, nftTokenID, shardID)

	require.Equal(t, uint32(big.NewInt(20).Uint64()), retrievedMetaData.Royalties)
}
