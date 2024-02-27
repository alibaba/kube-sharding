/*
Copyright 2024 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)
	dbImpl, mock, err := sqlmock.New()
	defer dbImpl.Close()
	assert.Nil(err)
	dbStorage := &DB{
		client:    db.NewDBMock(dbImpl),
		tableName: "testTable",
	}
	Register()

	keyCol, valueCol := KeyColName, ValueColName
	selectWhereClause := "SELECT " + keyCol + ", " + valueCol + " FROM testTable WHERE " + keyCol + " = ?"
	// get
	{
		rows := sqlmock.NewRows([]string{keyCol, valueCol}).AddRow("testKey", "testValue")
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").WillReturnRows(rows)
		kvPair, err := dbStorage.Get("testKey", nil)
		assert.Nil(err)
		assert.NotNil(kvPair)
		assert.Equal("testKey", kvPair.Key)
		assert.Equal([]byte("testValue"), kvPair.Value)
	}

	// get not found
	{
		mock.ExpectQuery(selectWhereClause).WithArgs("not_exist_key").WillReturnError(sql.ErrNoRows)
		kvPair, err := dbStorage.Get("not_exist_key", nil)
		assert.NotNil(err)
		assert.Nil(kvPair)
	}

	// get db inner error
	{
		rows := sqlmock.NewRows([]string{keyCol, valueCol}).AddRow("testKey", "testValue")
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").WillReturnRows(rows)
		kvPair, err := dbStorage.Get("not_exist_key", nil)
		assert.NotNil(err)
		assert.Nil(kvPair)
	}

	// exist and not exist
	{
		rows := sqlmock.NewRows([]string{keyCol, valueCol}).AddRow("testKey", "testValue")
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").
			WillReturnRows(rows).
			WithArgs("not_exist_key").
			WillReturnError(sql.ErrNoRows)

		exist, err := dbStorage.Exists("testKey", nil)
		assert.Nil(err)
		assert.True(exist)

		exist, err = dbStorage.Exists("not_exist_key", nil)
		assert.Nil(err)
		assert.False(exist)
	}

	// List prefix
	{
		rows := sqlmock.NewRows([]string{keyCol, valueCol}).
			AddRow("testKey", "testValue").
			AddRow("test", "testValue1")
		q := mock.ExpectQuery("SELECT (.+) FROM testTable WHERE (.+) LIKE")
		q.WithArgs("test%").WillReturnRows(rows)
		kvs, err := dbStorage.List("test", nil)
		assert.Nil(err)
		assert.Equal(2, len(kvs))

		q.WithArgs("not_exist_prefix%").WillReturnError(sql.ErrNoRows)
		kvs, err = dbStorage.List("not_exist_prefix", nil)
		assert.NotNil(err)
		assert.Equal(0, len(kvs))
	}

	// Put
	{
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").WillReturnError(sql.ErrNoRows)
		mock.ExpectPrepare("INSERT INTO (.+) VALUES").ExpectExec().WithArgs("testKey", "testValue").WillReturnResult(sqlmock.NewResult(1, 1))
		err := dbStorage.Put("testKey", []byte("testValue"), nil)
		assert.Nil(err)

		// update
		rows := sqlmock.NewRows([]string{"key", "value"}).AddRow("testKey", "testValue")
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").WillReturnRows(rows)
		mock.ExpectPrepare("UPDATE (.+) SET (.+) WHERE (.+)").ExpectExec().WithArgs("testValue1", "testKey").WillReturnResult(sqlmock.NewResult(1, 1))
		err = dbStorage.Put("testKey", []byte("testValue1"), nil)
		assert.Nil(err)

		// mock insert duplicate
		mock.ExpectQuery(selectWhereClause).WithArgs("testKey").WillReturnError(sql.ErrNoRows)
		mock.ExpectPrepare("INSERT INTO (.+) VALUES").ExpectExec().WithArgs("testKey", "testValue").WillReturnError(fmt.Errorf("duplicate unique key"))
		err = dbStorage.Put("testKey", []byte("testValue"), nil)
		assert.NotNil(err)
	}

	// Delete
	{
		mock.ExpectPrepare("DELETE FROM (.+) WHERE (.+)").ExpectExec().WithArgs("testKey").WillReturnResult(sqlmock.NewResult(1, 1))
		err := dbStorage.Delete("testKey")
		assert.Nil(err)

		mock.ExpectPrepare("DELETE FROM (.+) WHERE (.+) LIKE").ExpectExec().WithArgs("testKey%").WillReturnResult(sqlmock.NewResult(1, 1))
		err = dbStorage.DeleteTree("testKey")
		assert.Nil(err)
	}

}

func TestSupplyProtocol(t *testing.T) {
	assert := assert.New(t)
	e := "user:pwd@host:port/dbName?k=v"
	r, err := supplyDefaultProtocol(e)
	assert.Nil(err)
	assert.Equal("user:pwd@tcp(host:port)/dbName?k=v", r)

	e = "user:pwd@host:port"
	r, err = supplyDefaultProtocol(e)
	assert.NotNil(err)

	e = "user:pwdhost:port"
	r, err = supplyDefaultProtocol(e)
	assert.NotNil(err)

	e = "user:pwd@tcp(host:port)/dbName"
	r, err = supplyDefaultProtocol(e)
	assert.Nil(err)
	assert.Equal(e, r)
}
