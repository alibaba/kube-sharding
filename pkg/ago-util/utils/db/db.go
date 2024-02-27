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
	"context"
	"database/sql"
	"fmt"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	glog "k8s.io/klog"
)

var (
	ctx context.Context
)

// DB ...
type DB struct {
	db *sql.DB
}

// NewDB ...
func NewDB() *DB {
	return &DB{}
}

// NewDBMock ...
func NewDBMock(impl *sql.DB) *DB {
	return &DB{db: impl}
}

// Config ...
type Config struct {
	User    string
	Passwd  string
	Net     string
	Addr    string
	DBName  string
	Timeout time.Duration
	Params  map[string]string
}

func buildDSN(config *Config) string {
	conf := mysql.NewConfig()
	conf.User = config.User
	conf.Passwd = config.Passwd
	conf.Net = config.Net
	conf.Addr = config.Addr
	conf.DBName = config.DBName
	conf.Timeout = config.Timeout
	conf.Params = make(map[string]string)
	for k, v := range config.Params {
		config.Params[k] = v
	}
	conf.Params["charset"] = "utf8"
	if conf.Net == "" {
		conf.Net = "tcp"
	}
	return conf.FormatDSN()
}

// Open ...
func (d *DB) Open(config *Config) error {
	dsn := buildDSN(config)
	return d.OpenWithDSN(dsn)
}

// OpenWithDSN open db connection with dsn string
func (d *DB) OpenWithDSN(dsn string) error {
	// TODO driverName should be a func arg
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		glog.Errorf("open mysql [%s] failed: %v", dsn, err)
		return err
	}

	// what is the best practice ?
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(10)

	err = db.Ping()
	if err != nil {
		glog.Errorf("connect db [%s] failed: %v", dsn, err)
	} else {
		glog.Infof("initial conn to db [%s] success", dsn)
		d.db = db
	}
	return err
}

// Close ...
func (d *DB) Close() {
	glog.Info("close db")
	if d.db != nil {
		d.db.Close()
	}
}

// Mock ...
func (d *DB) Mock(db *sql.DB) {
	d.db = db
}

// SetImpl if necessary, such as tddlx
func (d *DB) SetImpl(db *sql.DB) {
	d.db = db
}

// Exec ...
func (d *DB) Exec(sql string, value ...interface{}) error {
	stmt, err := d.db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(value...)
	glog.Infof("db exec [%s] args: %v, error: %v", sql, value, err)
	return err
}

// ExecWithTX exec sql in transaction
func (d *DB) ExecWithTX(iLevel sql.IsolationLevel, sqlStr string, args ...interface{}) error {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{Isolation: iLevel})
	if err != nil {
		return fmt.Errorf("begin transaction err:%v,  sql:%s, isolation level:%v", err, sqlStr, iLevel)
	}
	_, execErr := tx.Exec(sqlStr, args)
	if execErr != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec error: %v, sql:%s, isolation level:%v", execErr, sqlStr, iLevel)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %v, sql:%s, isolation level:%v", err, sqlStr, iLevel)
	}
	return nil
}

// Row ...
type Row interface {
	Scan(args ...interface{}) error
}

// QueryRow ...
func (d *DB) QueryRow(mapper func(row Row) (interface{}, error), sql string, args ...interface{}) (interface{}, error) {
	row := d.db.QueryRow(sql, args...)
	return mapper(row)
}

// Query ...
func (d *DB) Query(mapper func(row Row) (interface{}, error), sql string, args ...interface{}) ([]interface{}, error) {
	rows, err := d.db.Query(sql, args...)
	if err != nil {
		glog.Errorf("query rows failed [%s] [%v]", sql, err)
		return nil, err
	}
	defer rows.Close()
	rets := make([]interface{}, 0, 16)
	for rows.Next() {
		ret, err := mapper(rows)
		if err != nil {
			return nil, err
		}
		rets = append(rets, ret)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("query rows failed [%s] [%v]", sql, err)
	}
	return rets, err
}

// QueryMap ...
func (d *DB) QueryMap(conv func(i interface{}) interface{}, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(sql, args...)
	if err != nil {
		glog.Errorf("query rows failed [%s] [%v]", sql, err)
		return nil, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	rets := make([]map[string]interface{}, 0)

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if conv == nil || *val == nil {
				m[colName] = *val
			} else {
				m[colName] = conv(*val)
			}
		}
		rets = append(rets, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("query rows failed [%s] [%v]", sql, err)
	}
	return rets, err
}
