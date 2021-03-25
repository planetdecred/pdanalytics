package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/planetdecred/pdanalytics/mempool"
	"github.com/planetdecred/pdanalytics/postgres/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const (
	// chart data types
	MempoolSize    = "size"
	MempoolFees    = "fees"
	MempoolTxCount = "tx-count"
)

func (pg PgDb) MempoolTableName() string {
	return models.TableNames.Mempool
}

func (pg PgDb) StoreMempool(ctx context.Context, mempoolDto mempool.Mempool) error {
	mempoolModel := mempoolDtoToModel(mempoolDto)
	err := mempoolModel.Insert(ctx, pg.db, boil.Infer())
	if err != nil {
		if !strings.Contains(err.Error(), "unique constraint") { // Ignore duplicate entries
			return err
		}
	}
	//  tx count 76, total size 54205 B, fees 0.00367100
	log.Infof("Added mempool entry at %s, tx count %2d, total size: %6d B, Total Fee: %010.8f",
		mempoolDto.Time.Format(dbhelper.DateTemplate), mempoolDto.NumberOfTransactions, mempoolDto.Size, mempoolDto.TotalFee)
	if err = pg.UpdateMempoolAggregateData(ctx); err != nil {
		return err
	}
	return nil
}

func mempoolDtoToModel(mempoolDto mempool.Mempool) models.Mempool {
	return models.Mempool{
		Time:                 mempoolDto.Time,
		FirstSeenTime:        null.TimeFrom(mempoolDto.FirstSeenTime),
		Size:                 null.IntFrom(int(mempoolDto.Size)),
		NumberOfTransactions: null.IntFrom(mempoolDto.NumberOfTransactions),
		Revocations:          null.IntFrom(mempoolDto.Revocations),
		Tickets:              null.IntFrom(mempoolDto.Tickets),
		Voters:               null.IntFrom(mempoolDto.Voters),
		Total:                null.Float64From(mempoolDto.Total),
		TotalFee:             null.Float64From(mempoolDto.TotalFee),
	}
}

func (pg *PgDb) LastMempoolBlockHeight() (height int64, err error) {
	rows := pg.db.QueryRow(lastMempoolBlockHeight)
	err = rows.Scan(&height)
	return
}

func (pg *PgDb) LastMempoolTime() (entryTime time.Time, err error) {
	rows := pg.db.QueryRow(lastMempoolEntryTime)
	err = rows.Scan(&entryTime)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

func (pg *PgDb) MempoolCount(ctx context.Context) (int64, error) {
	return models.Mempools().Count(ctx, pg.db)
}

func (pg *PgDb) Mempools(ctx context.Context, offtset int, limit int) ([]mempool.Dto, error) {
	mempoolSlice, err := models.Mempools(qm.OrderBy("time DESC"), qm.Offset(offtset), qm.Limit(limit)).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	var result []mempool.Dto
	for _, m := range mempoolSlice {
		result = append(result, mempool.Dto{
			TotalFee:             m.TotalFee.Float64,
			FirstSeenTime:        m.FirstSeenTime.Time.Format(dbhelper.DateTemplate),
			Total:                m.Total.Float64,
			Voters:               m.Voters.Int,
			Tickets:              m.Tickets.Int,
			Revocations:          m.Revocations.Int,
			Time:                 m.Time.Format(dbhelper.DateTemplate),
			Size:                 int32(m.Size.Int),
			NumberOfTransactions: m.NumberOfTransactions.Int,
		})
	}
	return result, nil
}

func (pg PgDb) UpdateMempoolAggregateData(ctx context.Context) error {
	log.Info("Updating mempool bin data")
	if err := pg.updateMempoolHourlyAverage(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}

	if err := pg.updateMempoolDailyAvg(ctx); err != nil && err != sql.ErrNoRows {
		return err
	}

	log.Info("Mempool bin data updated")
	return nil
}

func (pg *PgDb) updateMempoolHourlyAverage(ctx context.Context) error {
	lastHourEntry, err := models.MempoolBins(
		models.MempoolBinWhere.Bin.EQ(string(chart.HourBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.MempoolBinColumns.Time)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextHour = time.Time{}
	if lastHourEntry != nil {
		nextHour = time.Unix(lastHourEntry.Time, 0).Add(chart.AnHour * time.Second).UTC()
	}
	if time.Now().Before(nextHour) {
		return nil
	}

	totalCount, err := models.Mempools(
		models.MempoolWhere.Time.GTE(nextHour),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	var processed int64
	for processed < totalCount {

		// get the first record for the next day to fill gap
		firstMem, err := models.Mempools(
			models.MempoolWhere.Time.GTE(nextHour),
			qm.OrderBy(models.MempoolColumns.Time),
		).One(ctx, pg.db)
		if err != nil {
			return err
		}
		if firstMem != nil {
			nextHour = firstMem.Time
		}

		mempoolSlice, err := models.Mempools(
			models.MempoolWhere.Time.GTE(nextHour),
			models.MempoolWhere.Time.LT(nextHour.Add(7*24*time.Hour)),
			qm.OrderBy(models.MempoolColumns.Time),
		).All(ctx, pg.db)
		if err != nil {
			return err
		}

		dLen := len(mempoolSlice)
		dates, txCounts, sizes := make(chart.ChartUints, dLen), make(chart.ChartUints, dLen), make(chart.ChartUints, dLen)
		fees := make(chart.ChartFloats, dLen)
		for i, m := range mempoolSlice {
			dates[i] = uint64(m.Time.Unix())
			txCounts[i] = uint64(m.NumberOfTransactions.Int)
			sizes[i] = uint64(m.Size.Int)
			fees[i] = m.TotalFee.Float64
		}

		hours, _, hourIntervals := chart.GenerateHourBin(dates, nil)
		for i, interval := range hourIntervals {
			mempoolBin := models.MempoolBin{
				Time:                 int64(hours[i]),
				Bin:                  string(chart.HourBin),
				Size:                 null.IntFrom(int(sizes.Avg(interval[0], interval[1]))),
				TotalFee:             null.Float64From(fees.Avg(interval[0], interval[1])),
				NumberOfTransactions: null.IntFrom(int(txCounts.Avg(interval[0], interval[1]))),
			}
			if err = mempoolBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		nextHour = nextHour.Add(7 * 24 * time.Hour).UTC()
		log.Infof("Processed hourly average of %d to %d of %d mempool record", processed,
			processed+int64(len(mempoolSlice)), totalCount)
		processed += int64(len(mempoolSlice))
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (pg *PgDb) updateMempoolDailyAvg(ctx context.Context) error {
	lastDayEntry, err := models.MempoolBins(
		models.MempoolBinWhere.Bin.EQ(string(chart.DayBin)),
		qm.OrderBy(fmt.Sprintf("%s desc", models.MempoolBinColumns.Time)),
	).One(ctx, pg.db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var nextDay = time.Time{}
	if lastDayEntry != nil {
		nextDay = time.Unix(lastDayEntry.Time, 0).Add(chart.ADay * time.Second).UTC()
	} else {
		firstMem, err := models.Mempools(
			qm.OrderBy(models.MempoolColumns.Time),
		).One(ctx, pg.db)
		if err != nil {
			return err
		}
		nextDay = firstMem.Time
	}
	if time.Now().Before(nextDay) {
		return nil
	}

	totalCount, err := models.Mempools(
		models.MempoolWhere.Time.GTE(nextDay),
	).Count(ctx, pg.db)
	if err != nil {
		return err
	}

	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}

	var processed int64
	for processed < totalCount {

		// get the first record for the next day to fill gap
		firstMem, err := models.Mempools(
			models.MempoolWhere.Time.GTE(nextDay),
			qm.OrderBy(models.MempoolColumns.Time),
		).One(ctx, pg.db)
		if err != nil {
			return err
		}
		if firstMem != nil {
			nextDay = firstMem.Time
		}

		mempoolSlice, err := models.Mempools(
			models.MempoolWhere.Time.GTE(nextDay),
			models.MempoolWhere.Time.LT(nextDay.Add(30*24*time.Hour)),
			qm.OrderBy(models.MempoolColumns.Time),
		).All(ctx, pg.db)
		if err != nil {
			return err
		}

		dLen := len(mempoolSlice)
		dates, txCounts, sizes := make(chart.ChartUints, dLen), make(chart.ChartUints, dLen), make(chart.ChartUints, dLen)
		fees := make(chart.ChartFloats, dLen)
		for i, m := range mempoolSlice {
			dates[i] = uint64(m.Time.Unix())
			txCounts[i] = uint64(m.NumberOfTransactions.Int)
			sizes[i] = uint64(m.Size.Int)
			fees[i] = m.TotalFee.Float64
		}

		days, _, dayIntervals := chart.GenerateDayBin(dates, nil)
		for i, interval := range dayIntervals {
			mempoolBin := models.MempoolBin{
				Time:                 int64(days[i]),
				Bin:                  string(chart.DayBin),
				Size:                 null.IntFrom(int(sizes.Avg(interval[0], interval[1]))),
				TotalFee:             null.Float64From(fees.Avg(interval[0], interval[1])),
				NumberOfTransactions: null.IntFrom(int(txCounts.Avg(interval[0], interval[1]))),
			}
			if err = mempoolBin.Insert(ctx, tx, boil.Infer()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		nextDay = nextDay.Add(30 * 24 * time.Hour)
		log.Infof("Processed daily average of %d to %d of %d mempool record", processed,
			processed+int64(len(mempoolSlice)), totalCount)
		processed += int64(len(mempoolSlice))
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// *****CHARTS GETTER******* //

func (pg *PgDb) FetchEncodeChart(ctx context.Context, dataType, binString string) ([]byte, error) {

	switch dataType {
	case MempoolSize:
		return pg.fetchEncodeMempoolSize(ctx, binString)
	case MempoolFees:
		return pg.fetchEncodeMempoolFee(ctx, binString)

	case MempoolTxCount:
		return pg.fetchEncodeMempoolTxCount(ctx, binString)
	}
	return nil, chart.UnknownChartErr
}

func (pg *PgDb) fetchEncodeMempoolSize(ctx context.Context, binString string) ([]byte, error) {
	if binString == string(chart.DefaultBin) {
		mempoolSlice, err := models.Mempools(
			qm.Select(models.MempoolColumns.Time, models.MempoolColumns.Size),
			qm.OrderBy(models.MempoolColumns.Time),
		).All(ctx, pg.db)
		if err != nil {
			return nil, err
		}
		var time = make(chart.ChartUints, len(mempoolSlice))
		var data = make(chart.ChartUints, len(mempoolSlice))
		for i, m := range mempoolSlice {
			time[i] = uint64(m.Time.UTC().Unix())
			data[i] = uint64(m.Size.Int)
		}
		return chart.Encode(nil, time, data)
	}

	mempoolSlice, err := models.MempoolBins(
		models.MempoolBinWhere.Bin.EQ(binString),
		qm.Select(models.MempoolBinColumns.Time, models.MempoolBinColumns.Size),
		qm.OrderBy(models.MempoolBinColumns.Time),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	var time = make(chart.ChartUints, len(mempoolSlice))
	var data = make(chart.ChartUints, len(mempoolSlice))
	for i, m := range mempoolSlice {
		time[i] = uint64(m.Time)
		data[i] = uint64(m.Size.Int)
	}
	return chart.Encode(nil, time, data)
}

func (pg *PgDb) fetchEncodeMempoolFee(ctx context.Context, binString string) ([]byte, error) {
	if binString == string(chart.DefaultBin) {
		mempoolSlice, err := models.Mempools(
			qm.Select(models.MempoolColumns.Time, models.MempoolColumns.TotalFee),
			qm.OrderBy(models.MempoolColumns.Time),
		).All(ctx, pg.db)
		if err != nil {
			return nil, err
		}
		var time = make(chart.ChartUints, len(mempoolSlice))
		var data = make(chart.ChartFloats, len(mempoolSlice))
		for i, m := range mempoolSlice {
			time[i] = uint64(m.Time.UTC().Unix())
			data[i] = m.TotalFee.Float64
		}
		return chart.Encode(nil, time, data)
	}

	mempoolSlice, err := models.MempoolBins(
		models.MempoolBinWhere.Bin.EQ(binString),
		qm.Select(models.MempoolBinColumns.Time, models.MempoolBinColumns.TotalFee),
		qm.OrderBy(models.MempoolBinColumns.Time),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	var time = make(chart.ChartUints, len(mempoolSlice))
	var data = make(chart.ChartFloats, len(mempoolSlice))
	for i, m := range mempoolSlice {
		time[i] = uint64(m.Time)
		data[i] = m.TotalFee.Float64
	}
	return chart.Encode(nil, time, data)
}

func (pg *PgDb) fetchEncodeMempoolTxCount(ctx context.Context, binString string) ([]byte, error) {
	if binString == string(chart.DefaultBin) {
		mempoolSlice, err := models.Mempools(
			qm.Select(models.MempoolColumns.Time, models.MempoolColumns.NumberOfTransactions),
			qm.OrderBy(models.MempoolColumns.Time),
		).All(ctx, pg.db)
		if err != nil {
			return nil, err
		}
		var time = make(chart.ChartUints, len(mempoolSlice))
		var data = make(chart.ChartUints, len(mempoolSlice))
		for i, m := range mempoolSlice {
			time[i] = uint64(m.Time.UTC().Unix())
			data[i] = uint64(m.NumberOfTransactions.Int)
		}
		return chart.Encode(nil, time, data)
	}

	mempoolSlice, err := models.MempoolBins(
		models.MempoolBinWhere.Bin.EQ(binString),
		qm.Select(models.MempoolBinColumns.Time, models.MempoolBinColumns.NumberOfTransactions),
		qm.OrderBy(models.MempoolBinColumns.Time),
	).All(ctx, pg.db)
	if err != nil {
		return nil, err
	}
	var time = make(chart.ChartUints, len(mempoolSlice))
	var data = make(chart.ChartUints, len(mempoolSlice))
	for i, m := range mempoolSlice {
		time[i] = uint64(m.Time)
		data[i] = uint64(m.NumberOfTransactions.Int)
	}
	return chart.Encode(nil, time, data)
}
