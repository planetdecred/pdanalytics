package propagation

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/planetdecred/pdanalytics/propagation/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	// chart data types
	BlockPropagation = "block-propagation"
	BlockTimestamp   = "block-timestamp"
	VotesReceiveTime = "votes-receive-time"
)

type syncDbProvider func(source string) (*PgDb, error)

type PgDb struct {
	db                   *sql.DB
	queryTimeout         time.Duration
	syncSourceDbProvider syncDbProvider
	syncSources          []string
}

type logWriter struct{}

func (l logWriter) Write(p []byte) (n int, err error) {
	log.Debug(string(p))
	return len(p), nil
}

func NewPgDb(db *sql.DB, debug bool) *PgDb {
	if debug {
		boil.DebugMode = true
		boil.DebugWriter = logWriter{}
	}
	return &PgDb{
		db:           db,
		queryTimeout: time.Second * 30,
	}
}

func (pg *PgDb) Close() error {
	log.Trace("Closing postgresql connection")
	return pg.db.Close()
}

func (pg *PgDb) BlockTableName() string {
	return models.TableNames.Block
}

func (pg *PgDb) VoteTableName() string {
	return models.TableNames.Vote
}

func (pg *PgDb) SaveBlock(ctx context.Context, block Block) error {
	blockModel := blockDtoToModel(block)
	err := blockModel.Insert(ctx, pg.db, boil.Infer())
	if err != nil {
		if !strings.Contains(err.Error(), "unique constraint") { // Ignore duplicate entries
			return err
		}
	}

	votes, err := pg.votesByBlock(ctx, int64(block.BlockHeight))
	if err == nil {
		for _, vote := range votes {
			voteModel, err := models.FindVote(ctx, pg.db, vote.Hash)
			if err == nil {
				voteModel.BlockReceiveTime = null.TimeFrom(block.BlockReceiveTime)
				voteModel.BlockHash = null.StringFrom(block.BlockHash)
				_, err = voteModel.Update(ctx, pg.db, boil.Infer())
				if err != nil {
					log.Errorf("Unable to fetch vote for block receive time update: %s", err.Error())
				}
			}
		}
	}

	log.Infof("New block received at %s, PropagationHeight: %d, Hash: ...%s",
		block.BlockReceiveTime.Format(dbhelper.DateMiliTemplate), block.BlockHeight, block.BlockHash[len(block.BlockHash)-23:])
	return nil
}

func blockDtoToModel(block Block) models.Block {
	return models.Block{
		Height:            int(block.BlockHeight),
		Hash:              null.StringFrom(block.BlockHash),
		InternalTimestamp: null.TimeFrom(block.BlockInternalTime),
		ReceiveTime:       null.TimeFrom(block.BlockReceiveTime),
	}
}

func (pg *PgDb) BlockCount(ctx context.Context) (int64, error) {
	return models.Blocks().Count(ctx, pg.db)
}

func (pg *PgDb) Blocks(ctx context.Context, offset int, limit int) ([]BlockDto, error) {
	blockSlice, err := models.Blocks(qm.OrderBy(fmt.Sprintf("%s DESC", models.BlockColumns.ReceiveTime)),
		qm.Offset(offset), qm.Limit(limit)).All(ctx, pg.db)

	if err != nil {
		return nil, err
	}

	var blocks []BlockDto

	for _, block := range blockSlice {
		timeDiff := block.ReceiveTime.Time.Sub(block.InternalTimestamp.Time).Seconds()

		votes, err := pg.votesByBlock(ctx, int64(block.Height))
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		blocks = append(blocks, BlockDto{
			BlockHash:         block.Hash.String,
			BlockHeight:       uint32(block.Height),
			BlockInternalTime: block.InternalTimestamp.Time.Format(dbhelper.DateTemplate),
			BlockReceiveTime:  block.ReceiveTime.Time.Format(dbhelper.DateTemplate),
			Delay:             fmt.Sprintf("%04.2f", timeDiff),
			Votes:             votes,
		})
	}

	return blocks, nil
}

func (pg *PgDb) BlocksWithoutVotes(ctx context.Context, offset int, limit int) ([]BlockDto, error) {
	blockSlice, err := models.Blocks(qm.OrderBy(fmt.Sprintf("%s DESC", models.BlockColumns.ReceiveTime)), qm.Offset(offset), qm.Limit(limit)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var blocks []BlockDto

	for _, block := range blockSlice {
		timeDiff := block.ReceiveTime.Time.Sub(block.InternalTimestamp.Time).Seconds()

		blocks = append(blocks, BlockDto{
			BlockHash:         block.Hash.String,
			BlockHeight:       uint32(block.Height),
			BlockInternalTime: block.InternalTimestamp.Time.Format(dbhelper.DateTemplate),
			BlockReceiveTime:  block.ReceiveTime.Time.Format(dbhelper.DateTemplate),
			Delay:             fmt.Sprintf("%04.2f", timeDiff),
		})
	}

	return blocks, nil
}

func (pg *PgDb) getBlock(ctx context.Context, height int) (*models.Block, error) {
	block, err := models.Blocks(models.BlockWhere.Height.EQ(height)).One(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (pg *PgDb) SaveVote(ctx context.Context, vote Vote) error {
	voteModel := models.Vote{
		Hash:              vote.Hash,
		VotingOn:          null.Int64From(vote.VotingOn),
		BlockHash:         null.StringFrom(vote.BlockHash),
		ReceiveTime:       null.TimeFrom(vote.ReceiveTime),
		TargetedBlockTime: null.TimeFrom(vote.TargetedBlockTime),
		ValidatorID:       null.IntFrom(vote.ValidatorId),
		Validity:          null.StringFrom(vote.Validity),
	}

	// get the target block
	block, err := pg.getBlock(ctx, int(vote.VotingOn))
	if err == nil {
		voteModel.BlockReceiveTime = null.TimeFrom(block.ReceiveTime.Time)
	}

	err = voteModel.Insert(ctx, pg.db, boil.Infer())
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") { // Ignore duplicate entries
			return nil
		}
		return err
	}

	log.Infof("New vote received at %s for %d, Validator Id %d, Hash ...%s",
		vote.ReceiveTime.Format(dbhelper.DateMiliTemplate), vote.VotingOn, vote.ValidatorId, vote.Hash[len(vote.Hash)-23:])
	return nil
}

func (pg *PgDb) Votes(ctx context.Context, offset int, limit int) ([]VoteDto, error) {
	voteSlice, err := models.Votes(qm.OrderBy(fmt.Sprintf("%s DESC", models.BlockColumns.ReceiveTime)), qm.Offset(offset), qm.Limit(limit)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var votes = make([]VoteDto, len(voteSlice))
	for i, vote := range voteSlice {
		votes[i] = pg.voteModelToDto(vote)
	}

	return votes, nil
}

func (pg *PgDb) VotesByBlock(ctx context.Context, blockHash string) ([]VoteDto, error) {
	voteSlice, err := models.Votes(
		models.VoteWhere.BlockHash.EQ(null.StringFrom(blockHash)),
		qm.OrderBy(fmt.Sprintf("%s DESC", models.BlockColumns.ReceiveTime)),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var votes = make([]VoteDto, len(voteSlice))
	for i, vote := range voteSlice {
		votes[i] = pg.voteModelToDto(vote)
	}

	return votes, nil
}

func (pg *PgDb) votesByBlock(ctx context.Context, blockHeight int64) ([]VoteDto, error) {
	voteSlice, err := models.Votes(models.VoteWhere.VotingOn.EQ(null.Int64From(blockHeight)),
		qm.OrderBy(models.BlockColumns.ReceiveTime)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var votes []VoteDto
	for _, vote := range voteSlice {
		votes = append(votes, pg.voteModelToDto(vote))
	}

	return votes, nil
}

func (pg *PgDb) voteModelToDto(vote *models.Vote) VoteDto {
	timeDiff := vote.ReceiveTime.Time.Sub(vote.TargetedBlockTime.Time).Seconds()
	blockReceiveTimeDiff := vote.ReceiveTime.Time.Sub(vote.BlockReceiveTime.Time).Seconds()
	var shortBlockHash string
	if len(vote.BlockHash.String) > 0 {
		shortBlockHash = vote.BlockHash.String[len(vote.BlockHash.String)-8:]
	}

	return VoteDto{
		Hash:                  vote.Hash,
		ReceiveTime:           vote.ReceiveTime.Time.Format(dbhelper.DateTemplate),
		TargetedBlockTimeDiff: fmt.Sprintf("%04.2f", timeDiff),
		BlockReceiveTimeDiff:  fmt.Sprintf("%04.2f", blockReceiveTimeDiff),
		VotingOn:              vote.VotingOn.Int64,
		BlockHash:             vote.BlockHash.String,
		ShortBlockHash:        shortBlockHash,
		ValidatorId:           vote.ValidatorID.Int,
		Validity:              vote.Validity.String,
	}
}

func (pg *PgDb) VotesCount(ctx context.Context) (int64, error) {
	return models.Votes().Count(ctx, pg.db)
}

func (pg *PgDb) VotesBlockReceiveTimeDiffs(ctx context.Context) ([]PropagationChartData, error) {
	voteSlice, err := models.Votes(
		qm.OrderBy(models.VoteColumns.VotingOn)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var chartData = make([]PropagationChartData, len(voteSlice))
	for i, vote := range voteSlice {
		blockReceiveTimeDiff := vote.ReceiveTime.Time.Sub(vote.BlockReceiveTime.Time).Seconds()
		chartData[i] = PropagationChartData{
			BlockHeight: vote.VotingOn.Int64, TimeDifference: blockReceiveTimeDiff,
		}
	}

	return chartData, nil
}

func (pg *PgDb) BlockDelays(ctx context.Context, height int) ([]PropagationChartData, error) {
	blockSlice, err := models.Blocks(
		models.BlockWhere.Height.GT(height),
		qm.OrderBy(models.BlockColumns.Height)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	var chartData = make([]PropagationChartData, len(blockSlice))
	for i, block := range blockSlice {
		blockReceiveTimeDiff := block.ReceiveTime.Time.Sub(block.InternalTimestamp.Time).Seconds()
		chartData[i] = PropagationChartData{
			BlockHeight:    int64(block.Height),
			TimeDifference: blockReceiveTimeDiff,
			BlockTime:      block.InternalTimestamp.Time,
		}
	}

	return chartData, nil
}

func (pg *PgDb) fetchBlockReceiveTimeByHeight(ctx context.Context, height int32) ([]BlockReceiveTime, error) {
	blockSlice, err := models.Blocks(
		models.BlockWhere.Height.GT(int(height)),
		qm.Select(models.BlockColumns.Height, models.BlockColumns.ReceiveTime),
		qm.OrderBy(models.BlockColumns.Height),
	).All(ctx, pg.db)

	if err != nil {
		return nil, err
	}

	var chartData []BlockReceiveTime
	for _, block := range blockSlice {
		chartData = append(chartData, BlockReceiveTime{
			BlockHeight: int64(block.Height),
			ReceiveTime: block.ReceiveTime.Time,
		})
	}

	return chartData, nil
}

// UpdatePropagationData
func (pg PgDb) UpdatePropagationData(ctx context.Context) error {
	log.Info("Updating propagation data")

	if len(pg.syncSources) == 0 {
		log.Info("Please add one or more propagation sources")
		return nil
	}

	for _, source := range pg.syncSources {
		if err := pg.updatePropagationDataForSource(ctx, source); err != nil && err != sql.ErrNoRows {
			return err
		}
		if err := pg.updatePropagationHourlyAvgForSource(ctx, source); err != nil && err != sql.ErrNoRows {
			return err
		}
		if err := pg.updatePropagationDailyAvgForSource(ctx, source); err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	log.Info("Updated propagation data")
	return nil
}

func (pg *PgDb) updatePropagationDataForSource(ctx context.Context, source string) error {

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	log.Infof("Fetching propagation data for %s", source)
	// get the last entry for this source and prepage the propagation records
	lastEntry, err := models.Propagations(
		models.PropagationWhere.Source.EQ(source),
		models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.PropagationColumns.Time)),
	).One(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return err
	}
	var lastHeight int64
	if lastEntry != nil {
		lastHeight = lastEntry.Height
	}

	chartsBlockHeight := int32(lastHeight)
	mainBlockDelays, err := pg.BlockDelays(ctx, int(chartsBlockHeight))
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	localBlockReceiveTime := make(map[int64]float64)
	for _, record := range mainBlockDelays {
		timeDifference, _ := strconv.ParseFloat(fmt.Sprintf("%04.2f", record.TimeDifference), 64)
		localBlockReceiveTime[record.BlockHeight] = timeDifference
	}

	db, err := pg.syncSourceDbProvider(source)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	blockDelays, err := db.BlockDelays(ctx, int(chartsBlockHeight))
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	receiveTimeMap := make(map[int64]float64)
	for _, record := range blockDelays {
		receiveTimeMap[record.BlockHeight], _ = strconv.ParseFloat(fmt.Sprintf("%04.2f", record.TimeDifference), 64)
	}

	for _, rec := range mainBlockDelays {
		var propagation = models.Propagation{
			Height: rec.BlockHeight,
			Time:   rec.BlockTime.Unix(),
			Bin:    string(chart.DefaultBin),
			Source: source,
		}
		if sourceTime, found := receiveTimeMap[rec.BlockHeight]; found {
			propagation.Deviation = localBlockReceiveTime[rec.BlockHeight] - sourceTime
		}
		if err = propagation.Insert(ctx, tx, boil.Infer()); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (pg *PgDb) updatePropagationHourlyAvgForSource(ctx context.Context, source string) error {

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	log.Infof("Updating propagation hourly average for %s", source)
	lastHourEntry, err := models.Propagations(
		models.PropagationWhere.Bin.EQ(string(chart.HourBin)),
		models.PropagationWhere.Source.EQ(source),
		qm.OrderBy(fmt.Sprintf("%s desc", models.PropagationColumns.Time)),
	).One(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return err
	}

	var nextHour time.Time
	if lastHourEntry != nil {
		nextHour = time.Unix(lastHourEntry.Time, 0).Add(chart.AnHour * time.Second).UTC()
	} else {
		lastHourEntry, err = models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Source.EQ(source),
			qm.OrderBy(models.PropagationColumns.Time),
		).One(ctx, tx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		nextHour = time.Unix(lastHourEntry.Time, 0).UTC()
	}
	if time.Now().Before(nextHour) {
		_ = tx.Rollback()
		return nil
	}

	totalCount, err := models.Propagations(
		models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
		models.PropagationWhere.Time.GTE(nextHour.Unix()),
		models.PropagationWhere.Source.EQ(source),
	).Count(ctx, pg.db)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	step := 7 * 24 * time.Hour
	var processed int64
	for processed < totalCount {
		nextEntry, err := models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Time.GTE(nextHour.Unix()),
			qm.OrderBy(models.PropagationColumns.Time),
		).One(ctx, tx)
		if err != nil && err == sql.ErrNoRows {
			break
		} else if err != nil {
			return nil
		}
		nextHour = time.Unix(nextEntry.Time, 0)

		propagations, err := models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Time.GTE(nextHour.Unix()),
			models.PropagationWhere.Time.LT(nextHour.Add(step).Unix()),
			models.PropagationWhere.Source.EQ(source),
			qm.OrderBy(models.PropagationColumns.Time),
		).All(ctx, tx)
		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		dLen := len(propagations)
		var dates, heights = make(chart.ChartUints, dLen), make(chart.ChartUints, dLen)
		var deviations = make(chart.ChartFloats, dLen)
		for i, rec := range propagations {
			dates[i] = uint64(rec.Time)
			heights[i] = uint64(rec.Height)
			deviations[i] = rec.Deviation
		}
		hours, hourHeights, hourIntervals := chart.GenerateHourBin(dates, heights)
		for i, interval := range hourIntervals {
			propagationBin := models.Propagation{
				Time:      int64(hours[i]),
				Height:    int64(hourHeights[i]),
				Bin:       string(chart.HourBin),
				Source:    source,
				Deviation: deviations.Avg(interval[0], interval[1]),
			}
			if err = propagationBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		log.Infof("Processed hourly average of %d to %d of %d propagation records", processed,
			processed+int64(dLen), totalCount)
		processed += int64(dLen)
		nextHour = nextHour.Add(step)
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (pg *PgDb) updatePropagationDailyAvgForSource(ctx context.Context, source string) error {

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	log.Infof("Updating propagation daily average for %s", source)
	lastDayEntry, err := models.Propagations(
		models.PropagationWhere.Bin.EQ(string(chart.DayBin)),
		models.PropagationWhere.Source.EQ(source),
		qm.OrderBy(fmt.Sprintf("%s desc", models.PropagationColumns.Time)),
	).One(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return err
	}

	var nextDay time.Time
	if lastDayEntry != nil {
		nextDay = time.Unix(lastDayEntry.Time, 0).Add(chart.ADay * time.Second).UTC()
	} else {
		lastDayEntry, err = models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Source.EQ(source),
			qm.OrderBy(models.PropagationColumns.Time),
		).One(ctx, tx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		nextDay = time.Unix(lastDayEntry.Time, 0).UTC()
	}
	if time.Now().Before(nextDay) {
		_ = tx.Rollback()
		return nil
	}

	totalCount, err := models.Propagations(
		models.PropagationWhere.Time.GTE(nextDay.Unix()),
		models.PropagationWhere.Source.EQ(source),
	).Count(ctx, pg.db)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	var processed int64
	step := 30 * 24 * time.Hour
	for processed < totalCount {
		nextEntry, err := models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Time.GTE(nextDay.Unix()),
			qm.OrderBy(models.PropagationColumns.Time),
		).One(ctx, tx)
		if err != nil && err == sql.ErrNoRows {
			break
		} else if err != nil {
			return nil
		}
		nextDay = time.Unix(nextEntry.Time, 0)
		propagations, err := models.Propagations(
			models.PropagationWhere.Bin.EQ(string(chart.DefaultBin)),
			models.PropagationWhere.Time.GTE(nextDay.Unix()),
			models.PropagationWhere.Time.LT(nextDay.Add(step).Unix()),
			models.PropagationWhere.Source.EQ(source),
			qm.OrderBy(models.PropagationColumns.Time),
		).All(ctx, tx)
		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		dLen := len(propagations)
		var dates, heights = make(chart.ChartUints, dLen), make(chart.ChartUints, dLen)
		var deviations = make(chart.ChartFloats, dLen)
		for i, rec := range propagations {
			dates[i] = uint64(rec.Time)
			heights[i] = uint64(rec.Height)
			deviations[i] = rec.Deviation
		}
		days, dayHeight, dayIntervals := chart.GenerateDayBin(dates, heights)
		for i, interval := range dayIntervals {
			propagationBin := models.Propagation{
				Time:      int64(days[i]),
				Height:    int64(dayHeight[i]),
				Bin:       string(chart.DayBin),
				Source:    source,
				Deviation: deviations.Avg(interval[0], interval[1]),
			}
			if err = propagationBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		log.Infof("Processed daily average of %d to %d of %d propagation records", processed,
			processed+int64(dLen), totalCount)
		processed += int64(dLen)
		nextDay = nextDay.Add(step)
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// UpdateBlockBinData
func (pg *PgDb) UpdateBlockBinData(ctx context.Context) error {
	log.Info("Updating block bin data")
	if err := pg.updateBlockHourlyAvgData(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}
	if err := pg.updateBlockDailyAvgData(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func (pg *PgDb) updateBlockHourlyAvgData(ctx context.Context) error {
	lastEntry, err := models.BlockBins(
		models.BlockBinWhere.Bin.EQ(string(chart.HourBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.BlockBinColumns.Height)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextHour = time.Time{}
	var lastHeight int64
	if lastEntry != nil {
		lastHeight = lastEntry.Height
		nextHour = time.Unix(lastEntry.InternalTimestamp, 0).Add(chart.AnHour * time.Second).UTC()
	}

	if time.Now().Before(nextHour) {
		return nil
	}

	totalCount, err := models.Blocks(
		models.BlockWhere.Height.GT(int(lastHeight)),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	const pageSize = 1000
	var processed int64
	for processed < totalCount {
		log.Infof("Processing hourly average deviation of %d to %.0f of %d blocks", processed+1,
			math.Min(float64(processed+pageSize), float64(totalCount)), totalCount)
		blockSlice, err := models.Blocks(
			models.BlockWhere.Height.GT(int(lastHeight)), // lastHeight is updated below to ensure appropriate pagination
			qm.OrderBy(models.BlockColumns.Height),
			qm.Limit(pageSize),
		).All(ctx, pg.db)
		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		if err == sql.ErrNoRows {
			break
		}
		var dates, heights chart.ChartUints
		var diffs chart.ChartFloats
		for _, block := range blockSlice {
			dates = append(dates, uint64(block.InternalTimestamp.Time.Unix()))
			heights = append(heights, uint64(block.Height))
			blockReceiveTimeDiff := block.ReceiveTime.Time.Sub(block.InternalTimestamp.Time).Seconds()
			diffs = append(diffs, blockReceiveTimeDiff)
		}
		hours, hourHeights, hourIntervals := chart.GenerateHourBin(dates, heights)
		for i, interval := range hourIntervals {
			if int64(hours[i]) < nextHour.Unix() {
				continue
			}
			blockBin := models.BlockBin{
				InternalTimestamp: int64(hours[i]),
				Height:            int64(hourHeights[i]),
				Bin:               string(chart.HourBin),
				ReceiveTimeDiff:   diffs.Avg(interval[0], interval[1]),
			}
			if err = blockBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
			lastHeight = blockBin.Height
		}
		processed += int64(len(blockSlice))
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (pg *PgDb) updateBlockDailyAvgData(ctx context.Context) error {
	lastEntry, err := models.BlockBins(
		models.BlockBinWhere.Bin.EQ(string(chart.DayBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.BlockBinColumns.Height)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextDay = time.Time{}
	var lastHeight int64
	if lastEntry != nil {
		lastHeight = lastEntry.Height
		nextDay = time.Unix(lastEntry.InternalTimestamp, 0).Add(chart.ADay * time.Second).UTC()
	}

	if time.Now().Before(nextDay) {
		return nil
	}

	totalCount, err := models.Blocks(
		models.BlockWhere.Height.GT(int(lastHeight)),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	const pageSize = 1000
	var processed int64
	for processed < totalCount {
		log.Infof("Processing daily average deviation of %d to %.0f of %d blocks", processed+1,
			math.Min(float64(processed+pageSize), float64(totalCount)), totalCount)
		blockSlice, err := models.Blocks(
			models.BlockWhere.Height.GT(int(lastHeight)), // lastHeight is updated below to ensure appropriate pagination
			qm.OrderBy(models.BlockColumns.Height),
			qm.Limit(pageSize),
		).All(ctx, pg.db)
		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		if err == sql.ErrNoRows {
			break
		}
		var dates, heights chart.ChartUints
		var diffs chart.ChartFloats
		for _, block := range blockSlice {
			dates = append(dates, uint64(block.InternalTimestamp.Time.Unix()))
			heights = append(heights, uint64(block.Height))
			blockReceiveTimeDiff := block.ReceiveTime.Time.Sub(block.InternalTimestamp.Time).Seconds()
			diffs = append(diffs, blockReceiveTimeDiff)
		}
		days, dayHeights, dayIntervals := chart.GenerateDayBin(dates, heights)
		for i, interval := range dayIntervals {
			if int64(days[i]) < nextDay.Unix() {
				continue
			}
			blockBin := models.BlockBin{
				InternalTimestamp: int64(days[i]),
				Height:            int64(dayHeights[i]),
				Bin:               string(chart.DayBin),
				ReceiveTimeDiff:   diffs.Avg(interval[0], interval[1]),
			}
			if err = blockBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
			lastHeight = blockBin.Height
		}
		processed += int64(len(blockSlice))
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// UpdateVoteTimeDeviationData
func (pg *PgDb) UpdateVoteTimeDeviationData(ctx context.Context) error {
	log.Info("Updating vote time deviation data")
	if err := pg.updateVoteTimeDeviationHourlyAvgData(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}
	if err := pg.updateVoteTimeDeviationDailyAvgData(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func (pg *PgDb) updateVoteTimeDeviationHourlyAvgData(ctx context.Context) error {
	lastEntry, err := models.VoteReceiveTimeDeviations(
		models.VoteReceiveTimeDeviationWhere.Bin.EQ(string(chart.HourBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.VoteReceiveTimeDeviationColumns.BlockTime)),
		// The retrival process below ensure that all votes for the block is processed in one circle
		// qm.OrderBy(fmt.Sprintf("%s desc", models.VoteReceiveTimeDeviationColumns.Hash)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextHour time.Time
	if lastEntry != nil {
		nextHour = time.Unix(lastEntry.BlockTime, 0).Add(chart.AnHour * time.Second).UTC()
	} else {
		firstBlock, err := models.Blocks(
			qm.OrderBy(models.BlockColumns.Height),
		).One(ctx, pg.db)
		if err != nil && err != sql.ErrNoRows {
			return nil
		}
		if firstBlock == nil {
			return nil
		}
		nextHour = firstBlock.ReceiveTime.Time
	}

	if time.Now().Before(nextHour) {
		return nil
	}

	totalCount, err := models.Votes(
		models.VoteWhere.TargetedBlockTime.GTE(null.TimeFrom(nextHour)),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	if totalCount == 0 {
		return nil
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	const step = 7 * 24 * time.Hour
	var processed int64
	for processed < totalCount {

		voteSlice, err := models.Votes(
			// lastHeight is updated below to ensure appropriate pagination
			models.VoteWhere.TargetedBlockTime.GTE(null.TimeFrom(nextHour)),
			// Using block height to coordinate pagination to ensure the processing of all votes
			models.VoteWhere.TargetedBlockTime.LT(null.TimeFrom(nextHour.Add(step))),
			qm.OrderBy(models.VoteColumns.VotingOn),
		).All(ctx, tx)

		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		if err == sql.ErrNoRows {
			return nil
		}

		var dates, heights chart.ChartUints
		var diffs chart.ChartFloats
		for _, vote := range voteSlice {
			dates = append(dates, uint64(vote.TargetedBlockTime.Time.Unix()))
			heights = append(heights, uint64(vote.VotingOn.Int64))
			blockReceiveTimeDiff := vote.ReceiveTime.Time.Sub(vote.BlockReceiveTime.Time).Seconds()
			diffs = append(diffs, blockReceiveTimeDiff)
		}
		hours, hourHeights, hourIntervals := chart.GenerateHourBin(dates, heights)
		for i, interval := range hourIntervals {
			if int64(hours[i]) < nextHour.Unix() {
				continue
			}
			m := models.VoteReceiveTimeDeviation{
				BlockTime:             int64(hours[i]),
				BlockHeight:           int64(hourHeights[i]),
				Bin:                   string(chart.HourBin),
				ReceiveTimeDifference: diffs.Avg(interval[0], interval[1]),
			}
			if err = m.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}

		log.Infof("Processed hourly average vote receive time deviation of %d to %d of %d votes", processed,
			processed+int64(len(voteSlice)), totalCount)
		processed += int64(len(voteSlice))
		nextHour = nextHour.Add(step)
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (pg *PgDb) updateVoteTimeDeviationDailyAvgData(ctx context.Context) error {
	lastEntry, err := models.VoteReceiveTimeDeviations(
		models.VoteReceiveTimeDeviationWhere.Bin.EQ(string(chart.DayBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.VoteReceiveTimeDeviationColumns.BlockTime)),
		// The retrival process below ensure that all votes for the block is processed in one circle
		// qm.OrderBy(fmt.Sprintf("%s desc", models.VoteReceiveTimeDeviationColumns.Hash)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextDay time.Time
	if lastEntry != nil {
		nextDay = time.Unix(lastEntry.BlockTime, 0).Add(chart.ADay * time.Second).UTC()
	} else {
		firstBlock, err := models.Blocks(
			qm.OrderBy(models.BlockColumns.Height),
		).One(ctx, pg.db)
		if err != nil && err != sql.ErrNoRows {
			return nil
		}
		if firstBlock == nil {
			return nil
		}
		nextDay = firstBlock.ReceiveTime.Time
	}

	if time.Now().Before(nextDay) {
		return nil
	}

	totalCount, err := models.Votes(
		models.VoteWhere.TargetedBlockTime.GTE(null.TimeFrom(nextDay)),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	if totalCount == 0 {
		return nil
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	const step = 14 * 24 * time.Hour // 14 days
	var processed int64
	for processed < totalCount {

		voteSlice, err := models.Votes(
			// lastHeight is updated below to ensure appropriate pagination
			models.VoteWhere.TargetedBlockTime.GTE(null.TimeFrom(nextDay)),
			// Using block height to coordinate pagination to ensure the processing of all votes
			models.VoteWhere.TargetedBlockTime.LT(null.TimeFrom(nextDay.Add(step))),
			qm.OrderBy(models.VoteColumns.VotingOn),
		).All(ctx, tx)

		if err != nil && err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
		if err == sql.ErrNoRows {
			return nil
		}

		var dates, heights chart.ChartUints
		var diffs chart.ChartFloats
		for _, vote := range voteSlice {
			dates = append(dates, uint64(vote.TargetedBlockTime.Time.Unix()))
			heights = append(heights, uint64(vote.VotingOn.Int64))
			blockReceiveTimeDiff := vote.ReceiveTime.Time.Sub(vote.BlockReceiveTime.Time).Seconds()
			diffs = append(diffs, blockReceiveTimeDiff)
		}
		days, dayHeights, dayIntervals := chart.GenerateDayBin(dates, heights)
		for i, interval := range dayIntervals {
			if int64(days[i]) < nextDay.Unix() {
				continue
			}
			m := models.VoteReceiveTimeDeviation{
				BlockTime:             int64(days[i]),
				BlockHeight:           int64(dayHeights[i]),
				Bin:                   string(chart.DayBin),
				ReceiveTimeDifference: diffs.Avg(interval[0], interval[1]),
			}
			if err = m.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}

		log.Infof("Processed daily average vote receive time deviation of %d to %d of %d votes", processed,
			processed+int64(len(voteSlice)), totalCount)
		processed += int64(len(voteSlice))
		nextDay = nextDay.Add(step)
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (pg *PgDb) SourceDeviations(ctx context.Context, source, bin string) (records []SourceDeviation, err error) {
	data, err := models.Propagations(
		qm.Select(models.PropagationColumns.Height, models.PropagationColumns.Time, models.PropagationColumns.Deviation),
		models.PropagationWhere.Source.EQ(source),
		models.PropagationWhere.Bin.EQ(bin),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		records = append(records, SourceDeviation{
			Height:    d.Height,
			Time:      d.Time,
			Deviation: d.Deviation,
		})
	}
	return
}

func (pg *PgDb) BlockBinData(ctx context.Context, bin string) (records []BlockBinDto, err error) {
	blocks, err := models.BlockBins(
		models.BlockBinWhere.Bin.EQ(bin),
		qm.OrderBy(models.BlockBinColumns.InternalTimestamp),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		records = append(records, BlockBinDto{
			Height:            b.Height,
			ReceiveTimeDiff:   b.ReceiveTimeDiff,
			InternalTimestamp: b.InternalTimestamp,
		})
	}
	return
}

func (pg *PgDb) VoteReceiveTimeDeviations(ctx context.Context, bin string) (result []VoteReceiveTimeDeviation, err error) {
	records, err := models.VoteReceiveTimeDeviations(
		models.VoteReceiveTimeDeviationWhere.Bin.EQ(bin),
		qm.OrderBy(models.VoteReceiveTimeDeviationColumns.BlockTime),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		result = append(result, VoteReceiveTimeDeviation{
			BlockHeight:           r.BlockHeight,
			BlockTime:             r.BlockTime,
			ReceiveTimeDifference: r.ReceiveTimeDifference,
		})
	}
	return
}
