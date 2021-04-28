// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/planetdecred/pdanalytics/app/helpers"
	"github.com/planetdecred/pdanalytics/postgres/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const dateTemplate = "2006-01-02 15:04"
const dateMiliTemplate = "2006-01-02 15:04:05.99"

type insertable interface {
	Insert(context.Context, boil.ContextExecutor, boil.Columns) error
}

type upsertable interface {
	Upsert(ctx context.Context, db boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error
}

func UnixTimeToString(t int64) string {
	return helpers.UnixTime(t).Format(dateTemplate)
}

func RoundValue(input float64) string {
	value := input * 100
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func (pg *PgDb) tryInsert(ctx context.Context, txr boil.Transactor, data insertable) error {
	err := data.Insert(ctx, pg.db, boil.Infer())
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return err
		}
		errT := txr.Rollback()
		if errT != nil {
			return errT
		}
		return err
	}
	return nil
}

func (pg *PgDb) tryUpsert(ctx context.Context, txr boil.Transactor, data upsertable) error {
	err := data.Upsert(ctx, pg.db, true, nil, boil.Infer(), boil.Infer())
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return err
		}
		errT := txr.Rollback()
		if errT != nil {
			return errT
		}
		return err
	}
	return nil
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unique constraint")
}

func (pg *PgDb) LastEntry(ctx context.Context, tableName string, receiver interface{}) error {
	var columnName string
	switch tableName {
	case models.TableNames.Exchange:
		columnName = models.ExchangeColumns.ID
	case models.TableNames.ExchangeTick:
		columnName = models.ExchangeTickColumns.ID
	case models.TableNames.Mempool:
		columnName = models.MempoolColumns.Time
	case models.TableNames.Block:
		columnName = models.BlockColumns.Height
	case models.TableNames.Vote:
		columnName = models.VoteColumns.ReceiveTime
	// case models.TableNames.PowData:
	// 	columnName = models.PowDatumColumns.Time
	// case models.TableNames.VSP:
	// 	columnName = models.VSPColumns.ID
	// case models.TableNames.VSPTick:
	// 	columnName = models.VSPTickColumns.ID
	case models.TableNames.Reddit:
		columnName = models.RedditColumns.Date
	case models.TableNames.Twitter:
		columnName = models.TwitterColumns.Date
	case models.TableNames.Github:
		columnName = models.GithubColumns.Date
	case models.TableNames.Youtube:
		columnName = models.YoutubeColumns.Date
	case models.TableNames.NetworkSnapshot:
		columnName = models.NetworkSnapshotColumns.Timestamp
	}

	rows := pg.db.QueryRow(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s DESC LIMIT 1", columnName, tableName, columnName))
	err := rows.Scan(receiver)
	return err

}
