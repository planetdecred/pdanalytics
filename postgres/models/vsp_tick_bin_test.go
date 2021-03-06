// Code generated by SQLBoiler 4.5.0 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/volatiletech/randomize"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/strmangle"
)

var (
	// Relationships sometimes use the reflection helper queries.Equal/queries.Assign
	// so force a package dependency in case they don't.
	_ = queries.Equal
)

func testVSPTickBins(t *testing.T) {
	t.Parallel()

	query := VSPTickBins()

	if query.Query == nil {
		t.Error("expected a query, got nothing")
	}
}

func testVSPTickBinsDelete(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := o.Delete(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testVSPTickBinsQueryDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := VSPTickBins().DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testVSPTickBinsSliceDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := VSPTickBinSlice{o}

	if rowsAff, err := slice.DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testVSPTickBinsExists(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	e, err := VSPTickBinExists(ctx, tx, o.VSPID, o.Time, o.Bin)
	if err != nil {
		t.Errorf("Unable to check if VSPTickBin exists: %s", err)
	}
	if !e {
		t.Errorf("Expected VSPTickBinExists to return true, but got false.")
	}
}

func testVSPTickBinsFind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	vspTickBinFound, err := FindVSPTickBin(ctx, tx, o.VSPID, o.Time, o.Bin)
	if err != nil {
		t.Error(err)
	}

	if vspTickBinFound == nil {
		t.Error("want a record, got nil")
	}
}

func testVSPTickBinsBind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = VSPTickBins().Bind(ctx, tx, o); err != nil {
		t.Error(err)
	}
}

func testVSPTickBinsOne(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if x, err := VSPTickBins().One(ctx, tx); err != nil {
		t.Error(err)
	} else if x == nil {
		t.Error("expected to get a non nil record")
	}
}

func testVSPTickBinsAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	vspTickBinOne := &VSPTickBin{}
	vspTickBinTwo := &VSPTickBin{}
	if err = randomize.Struct(seed, vspTickBinOne, vspTickBinDBTypes, false, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}
	if err = randomize.Struct(seed, vspTickBinTwo, vspTickBinDBTypes, false, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = vspTickBinOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = vspTickBinTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := VSPTickBins().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 2 {
		t.Error("want 2 records, got:", len(slice))
	}
}

func testVSPTickBinsCount(t *testing.T) {
	t.Parallel()

	var err error
	seed := randomize.NewSeed()
	vspTickBinOne := &VSPTickBin{}
	vspTickBinTwo := &VSPTickBin{}
	if err = randomize.Struct(seed, vspTickBinOne, vspTickBinDBTypes, false, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}
	if err = randomize.Struct(seed, vspTickBinTwo, vspTickBinDBTypes, false, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = vspTickBinOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = vspTickBinTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 2 {
		t.Error("want 2 records, got:", count)
	}
}

func testVSPTickBinsInsert(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testVSPTickBinsInsertWhitelist(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Whitelist(vspTickBinColumnsWithoutDefault...)); err != nil {
		t.Error(err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testVSPTickBinToOneVSPUsingVSP(t *testing.T) {
	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var local VSPTickBin
	var foreign VSP

	seed := randomize.NewSeed()
	if err := randomize.Struct(seed, &local, vspTickBinDBTypes, false, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}
	if err := randomize.Struct(seed, &foreign, vspDBTypes, false, vspColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSP struct: %s", err)
	}

	if err := foreign.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	local.VSPID = foreign.ID
	if err := local.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	check, err := local.VSP().One(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	if check.ID != foreign.ID {
		t.Errorf("want: %v, got %v", foreign.ID, check.ID)
	}

	slice := VSPTickBinSlice{&local}
	if err = local.L.LoadVSP(ctx, tx, false, (*[]*VSPTickBin)(&slice), nil); err != nil {
		t.Fatal(err)
	}
	if local.R.VSP == nil {
		t.Error("struct should have been eager loaded")
	}

	local.R.VSP = nil
	if err = local.L.LoadVSP(ctx, tx, true, &local, nil); err != nil {
		t.Fatal(err)
	}
	if local.R.VSP == nil {
		t.Error("struct should have been eager loaded")
	}
}

func testVSPTickBinToOneSetOpVSPUsingVSP(t *testing.T) {
	var err error

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var a VSPTickBin
	var b, c VSP

	seed := randomize.NewSeed()
	if err = randomize.Struct(seed, &a, vspTickBinDBTypes, false, strmangle.SetComplement(vspTickBinPrimaryKeyColumns, vspTickBinColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &b, vspDBTypes, false, strmangle.SetComplement(vspPrimaryKeyColumns, vspColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &c, vspDBTypes, false, strmangle.SetComplement(vspPrimaryKeyColumns, vspColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}

	if err := a.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}
	if err = b.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	for i, x := range []*VSP{&b, &c} {
		err = a.SetVSP(ctx, tx, i != 0, x)
		if err != nil {
			t.Fatal(err)
		}

		if a.R.VSP != x {
			t.Error("relationship struct not set to correct value")
		}

		if x.R.VSPTickBins[0] != &a {
			t.Error("failed to append to foreign relationship struct")
		}
		if a.VSPID != x.ID {
			t.Error("foreign key was wrong value", a.VSPID)
		}

		if exists, err := VSPTickBinExists(ctx, tx, a.VSPID, a.Time, a.Bin); err != nil {
			t.Fatal(err)
		} else if !exists {
			t.Error("want 'a' to exist")
		}

	}
}

func testVSPTickBinsReload(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = o.Reload(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testVSPTickBinsReloadAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := VSPTickBinSlice{o}

	if err = slice.ReloadAll(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testVSPTickBinsSelect(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := VSPTickBins().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 1 {
		t.Error("want one record, got:", len(slice))
	}
}

var (
	vspTickBinDBTypes = map[string]string{`VSPID`: `integer`, `Bin`: `character varying`, `Immature`: `integer`, `Live`: `integer`, `Voted`: `integer`, `Missed`: `integer`, `PoolFees`: `double precision`, `ProportionLive`: `double precision`, `ProportionMissed`: `double precision`, `UserCount`: `integer`, `UsersActive`: `integer`, `Time`: `bigint`}
	_                 = bytes.MinRead
)

func testVSPTickBinsUpdate(t *testing.T) {
	t.Parallel()

	if 0 == len(vspTickBinPrimaryKeyColumns) {
		t.Skip("Skipping table with no primary key columns")
	}
	if len(vspTickBinAllColumns) == len(vspTickBinPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	if rowsAff, err := o.Update(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only affect one row but affected", rowsAff)
	}
}

func testVSPTickBinsSliceUpdateAll(t *testing.T) {
	t.Parallel()

	if len(vspTickBinAllColumns) == len(vspTickBinPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &VSPTickBin{}
	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, vspTickBinDBTypes, true, vspTickBinPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	// Remove Primary keys and unique columns from what we plan to update
	var fields []string
	if strmangle.StringSliceMatch(vspTickBinAllColumns, vspTickBinPrimaryKeyColumns) {
		fields = vspTickBinAllColumns
	} else {
		fields = strmangle.SetComplement(
			vspTickBinAllColumns,
			vspTickBinPrimaryKeyColumns,
		)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	typ := reflect.TypeOf(o).Elem()
	n := typ.NumField()

	updateMap := M{}
	for _, col := range fields {
		for i := 0; i < n; i++ {
			f := typ.Field(i)
			if f.Tag.Get("boil") == col {
				updateMap[col] = value.Field(i).Interface()
			}
		}
	}

	slice := VSPTickBinSlice{o}
	if rowsAff, err := slice.UpdateAll(ctx, tx, updateMap); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("wanted one record updated but got", rowsAff)
	}
}

func testVSPTickBinsUpsert(t *testing.T) {
	t.Parallel()

	if len(vspTickBinAllColumns) == len(vspTickBinPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	// Attempt the INSERT side of an UPSERT
	o := VSPTickBin{}
	if err = randomize.Struct(seed, &o, vspTickBinDBTypes, true); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert VSPTickBin: %s", err)
	}

	count, err := VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}

	// Attempt the UPDATE side of an UPSERT
	if err = randomize.Struct(seed, &o, vspTickBinDBTypes, false, vspTickBinPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize VSPTickBin struct: %s", err)
	}

	if err = o.Upsert(ctx, tx, true, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert VSPTickBin: %s", err)
	}

	count, err = VSPTickBins().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}
}
