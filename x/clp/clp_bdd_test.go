package clp

import (
	"github.com/Sifchain/sifnode/x/clp/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/store"
	//"github.com/Sifchain/sifnode/x/clp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/supply"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"
	//"testing"
)

//nolint:deadcode,unused
var (
	delPk1   = ed25519.GenPrivKey().PubKey()
	delPk2   = ed25519.GenPrivKey().PubKey()
	delPk3   = ed25519.GenPrivKey().PubKey()
	delAddr1 = sdk.AccAddress(delPk1.Address())
	delAddr2 = sdk.AccAddress(delPk2.Address())
	delAddr3 = sdk.AccAddress(delPk3.Address())

	valOpPk1    = ed25519.GenPrivKey().PubKey()
	valOpPk2    = ed25519.GenPrivKey().PubKey()
	valOpPk3    = ed25519.GenPrivKey().PubKey()
	valAccAddr1 = sdk.AccAddress(valOpPk1.Address())
	valAccAddr2 = sdk.AccAddress(valOpPk2.Address())
	valAccAddr3 = sdk.AccAddress(valOpPk3.Address())

	TestAddrs = []sdk.AccAddress{
		delAddr1, delAddr2, delAddr3,
		valAccAddr1, valAccAddr2, valAccAddr3,
	}

	distrAcc = supply.NewEmptyModuleAccount(types.ModuleName)
)

// create a codec used only for testing
func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	supply.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	types.RegisterCodec(cdc) // distr
	return cdc
}

// test input with default values
func CreateTestInputDefaultBdd(isCheckTx bool, initPower int64) (
	sdk.Context, Keeper) {
	ctx, keeper := CreateTestInputAdvancedBdd(isCheckTx, initPower)
	return ctx, keeper
}

func CreateTestInputAdvancedBdd(isCheckTx bool, initPower int64) (sdk.Context, Keeper) {

	initTokens := sdk.TokensFromConsensusPower(initPower)

	keyClp := sdk.NewKVStoreKey(types.StoreKey)
	keyDistr := sdk.NewKVStoreKey(distribution.StoreKey)
	keyStaking := sdk.NewKVStoreKey(staking.StoreKey)
	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keySupply := sdk.NewKVStoreKey(supply.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)

	ms.MountStoreWithDB(keyDistr, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyClp, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySupply, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)

	err := ms.LoadLatestVersion()
	Expect(err).To(BeNil())
	//require.Nil(t, err)

	feeCollectorAcc := supply.NewEmptyModuleAccount(auth.FeeCollectorName)
	notBondedPool := supply.NewEmptyModuleAccount(staking.NotBondedPoolName, supply.Burner, supply.Staking)
	bondPool := supply.NewEmptyModuleAccount(staking.BondedPoolName, supply.Burner, supply.Staking)

	blacklistedAddrs := make(map[string]bool)
	blacklistedAddrs[feeCollectorAcc.GetAddress().String()] = true
	blacklistedAddrs[notBondedPool.GetAddress().String()] = true
	blacklistedAddrs[bondPool.GetAddress().String()] = true
	blacklistedAddrs[distrAcc.GetAddress().String()] = true

	cdc := MakeTestCodec()
	pk := params.NewKeeper(cdc, keyParams, tkeyParams)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, isCheckTx, log.NewNopLogger())
	accountKeeper := auth.NewAccountKeeper(cdc, keyAcc, pk.Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount)
	bankKeeper := bank.NewBaseKeeper(accountKeeper, pk.Subspace(bank.DefaultParamspace), blacklistedAddrs)
	maccPerms := map[string][]string{
		auth.FeeCollectorName:     nil,
		types.ModuleName:          nil,
		staking.NotBondedPoolName: {supply.Burner, supply.Staking},
		staking.BondedPoolName:    {supply.Burner, supply.Staking},
	}
	supplyKeeper := supply.NewKeeper(cdc, keySupply, accountKeeper, bankKeeper, maccPerms)
	sk := staking.NewKeeper(cdc, keyStaking, supplyKeeper, pk.Subspace(staking.DefaultParamspace))
	sk.SetParams(ctx, staking.DefaultParams())
	keeper := NewKeeper(cdc, keyClp, bankKeeper, pk.Subspace(types.DefaultParamspace))
	keeper.SetParams(ctx, types.DefaultParams())
	Expect(types.DefaultParams()).To(Equal(keeper.GetParams(ctx)))
	//require.Equal(t, types.DefaultParams(), keeper.GetParams(ctx))
	initCoins := sdk.NewCoins(sdk.NewCoin(sk.BondDenom(ctx), initTokens))
	totalSupply := sdk.NewCoins(sdk.NewCoin(sk.BondDenom(ctx), initTokens.MulRaw(int64(len(TestAddrs)))))
	supplyKeeper.SetSupply(ctx, supply.NewSupply(totalSupply))
	for _, addr := range TestAddrs {
		_, err = bankKeeper.AddCoins(ctx, addr, initCoins)
		Expect(err).To(BeNil())
		//require.Nil(t, err)
	}

	return ctx, keeper
}

var _ = Describe("ClpBdd", func() {
	var (
		ctx sdk.Context
		keeper Keeper
	)

	BeforeEach(func() {
		ctx, keeper = CreateTestInputDefaultBdd(false, 1000)
	})

	Describe("Test set pool", func() {
		Context("With a random pool", func() {
			It("should be fetchable", func() {
				pool := GenerateRandomPool(1)[0]
				getpool, err := keeper.GetPool(ctx, pool.ExternalAsset.Ticker)
				Expect(getpool).To(Equal(pool))
				Expect(err).To(BeNil())
			})

		})
	})
})
