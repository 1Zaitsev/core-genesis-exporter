package nexus

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressWASAVAXVaul = "terra1hn9rzu66s422rl9kg0a7j2yxdjef0szkqvy7ws"
	AddressNAVAX       = "terra13k62n0285wj8ug0ngcgpf7dgnkzqeu279tz636"

	PsiNAVAXPair = "terra10usmg35qsa92fagh49np7phmhhr4ryhyl27749"
	LpToken      = "terra1p3zj8tkzufw9szmm97taj7x6kkd0cy7k2mpdws"

	AddressAnchorOverseer = "terra1tmnqgvg567ypvsvk6rwsga3srp7e3lg6u0elp8"
)

func ExportNexus(app *terra.TerraApp, height int64) (util.SnapshotBalanceAggregateMap, error) {
	var ctx context.Context = util.PrepCtxByHeight(app, height)
	qs := util.PrepWasmQueryServer(app)

	keeper := app.WasmKeeper

	//get all nAsset wallets with balances
	var nAssetHolders = make(util.BalanceMap)
	if err := getCW20Balances(ctx, keeper, AddressNAVAX, nAssetHolders); err != nil {
		return nil, fmt.Errorf("failed to fetch nasset token holders; %v", err)
	}

	// get all nAsset from astro LP
	var nAssetFromLP = make(util.BalanceMap)
	if err := getLPBalances(ctx, keeper, qs, LpToken, PsiNAVAXPair, AddressNAVAX, nAssetFromLP); err != nil {
		return nil, fmt.Errorf("failed to fetch nasset token holders from LP; %v", err)
	}

	// merge holder maps + nLUNA holdings from LP
	mergednAtomHolderMap := util.MergeMaps(nAssetHolders, nAssetFromLP)

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

func getLPBalances(ctx context.Context, k wasmkeeper.Keeper, q wasmtypes.QueryServer, lpTokenAddress string, pairAddress string, nAssetAddr string, balanceMap map[string]sdk.Int) error {
	lpSupply, err := util.GetCW20TotalSupply(ctx, q, lpTokenAddress)
	if err != nil {
		return err
	}

	pairNAssetBalance, err := util.GetCW20Balance(ctx, q, nAssetAddr, pairAddress)
	if err != nil {
		return err
	}

	prefix := util.GeneratePrefix("balance")
	lpTokenAddr, err := sdk.AccAddressFromBech32(lpTokenAddress)
	if err != nil {
		return err
	}
	k.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lpTokenAddr, prefix, func(key, value []byte) bool {
		if contract, _ := isContract(ctx, k, string(key)); contract {
			return false
		}
		lpBalance, ok := sdk.NewIntFromString(string(value[1 : len(value)-1]))
		if ok {
			if lpBalance.IsZero() {
				return false
			}
			balance := lpBalance.Mul(pairNAssetBalance).Quo(lpSupply)
			if strings.Contains(string(key), "terra") {
				balanceMap[string(key)] = balance
			} else {
				addr := sdk.AccAddress(key)
				balanceMap[addr.String()] = balance
			}
		}
		return false
	})
	return nil
}

func getCW20Balances(ctx context.Context, k wasmkeeper.Keeper, tokenAddress string, balanceMap map[string]sdk.Int) error {
	prefix := util.GeneratePrefix("balance")
	tokenAddr, err := sdk.AccAddressFromBech32(tokenAddress)
	if err != nil {
		return err
	}
	k.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), tokenAddr, prefix, func(key, value []byte) bool {
		if contract, _ := isContract(ctx, k, string(key)); contract {
			return false
		}
		balance, ok := sdk.NewIntFromString(string(value[1 : len(value)-1]))
		// fmt.Printf("%s %s\n", string(key), balance.String())
		if ok {
			if balance.IsZero() {
				return false
			}
			if strings.Contains(string(key), "terra") {
				balanceMap[string(key)] = balance
			} else {
				addr := sdk.AccAddress(key)
				balanceMap[addr.String()] = balance
			}
		}
		return false
	})
	return nil
}

func isContract(ctx context.Context, keeper wasmkeeper.Keeper, address string) (bool, error) {
	contractAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return true, err
	}

	if _, err := keeper.GetContractInfo(sdk.UnwrapSDKContext(ctx), contractAddr); err != nil {
		return false, nil
	}

	return true, nil
}
