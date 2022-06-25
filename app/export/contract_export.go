package app

import (
	"fmt"

	"github.com/terra-money/core/app/export/nexus"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp, height int64) []types.Balance {
	// var err error
	var snapshotType util.Snapshot
	if app.LastBlockHeight() == 7544910 {
		snapshotType = util.Snapshot(util.PreAttack)
	} else {
		snapshotType = util.Snapshot(util.PostAttack)
	}

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d - %s", app.LastBlockHeight(), snapshotType))

	nexusSs, err := nexus.ExportNexus(app, height)
	check(err)

	snapshot := util.MergeSnapshots(nexusSs)

	return snapshot.ExportToBalances()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
