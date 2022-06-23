package nexus

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressBATOMVaul = "terra1lh3h7l5vsul2pxlevraucwev42ar6kyx33u4c8"
	AddressNATOM     = "terra1jtdc6zpf95tvh9peuaxwp3v0yqszcnwl8j5ade"

	PsiNATOMPair = "terra1spcf4486jjn8678hstwqzeeudu98yp7pyyltnl"
	LpToken      = "terra1pyavxxun3vuakqq0wyqft69l3zjns0q76wut7z"

	AddressAnchorOverseer = "terra1tmnqgvg567ypvsvk6rwsga3srp7e3lg6u0elp8"
)

func ExportNexus(app *terra.TerraApp) (util.SnapshotBalanceAggregateMap, error) {
	var ctx context.Context = util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	keeper := app.WasmKeeper

	// get all LP token holders
	var nAtomHolderMapFromLp = make(util.BalanceMap)
	if err := util.GetCW20AccountsAndBalances(ctx, keeper, LpToken, nAtomHolderMapFromLp); err != nil {
		return nil, fmt.Errorf("failed to fetch lp token holders; %v", err)
	}

	var LPTokenSupply struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: LpToken,
		QueryMsg:        []byte("{\"token_info\":{}}"),
	}, &LPTokenSupply); err != nil {
		return nil, fmt.Errorf("failed to fetch lp token total supply: %v", err)
	}

	var totalNAtomInLP struct {
		Balance sdk.Int `json:"balance"`
	}
	query_string := fmt.Sprintf("{\"balance\":{%s}}", PsiNATOMPair)
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressNATOM,
		QueryMsg:        []byte(query_string),
	}, &totalNAtomInLP); err != nil {
		return nil, fmt.Errorf("failed to fetch lp total nAtom amout: %v", err)
	}

	// todo: figure out how much bAsset was provided by lp token amount
	// lp token balance/total supply = x
	// pair nAsset balance * x;
	for _, v := range nAtomHolderMapFromLp {
		v = sdk.NewDecFromInt(v).QuoInt(LPTokenSupply.TotalSupply).MulInt(totalNAtomInLP.Balance).RoundInt()
	}

	// get all nAtom holders
	var nAtomHolderMap = make(util.BalanceMap)
	if err := util.GetCW20AccountsAndBalances(ctx, keeper, AddressNATOM, nAtomHolderMap); err != nil {
		return nil, fmt.Errorf("failed to fetch nAtom holders: %v", err)
	}

	// merge holder maps + nLUNA holdings from LP
	mergednAtomHolderMap := util.MergeMaps(nAtomHolderMap, nAtomHolderMapFromLp)

	// iterate over merged nLUNA holder map, apply nLUNA -> bLUNA ratio
	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for userAddr, nAtomHolding := range mergednAtomHolderMap {

		// there can't be more than 1 holding -- this is fine
		finalBalance[userAddr] = []util.SnapshotBalance{
			{
				Denom:   "natom",
				Balance: nAtomHolding,
			},
		}
	}

	return finalBalance, nil
}
