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
	"fmt"
	"net"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	glog "k8s.io/klog"
)

type logger int

func (l logger) Print(v ...interface{}) {
	glog.Info(v...)
}

//InitDbGormConnection ...
func InitDbGormConnection(connStr string, poolSize int, dns utils.DNSClient) (*gorm.DB, error) {
	//"Mysqlconn":"user:pwd@tcp(url:3306)/feeds?charset=utf8mb4&autocommit=true&parseTime=true&loc=Local",
	cfg, err := mysql.ParseDSN(connStr)
	if nil != err {
		glog.Error(err)
		return nil, err
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	var l logger
	if dns != nil {
		l.Print("init dns type")
		err = initDNSClient(cfg.Addr, cfg.Timeout, dns)
		if nil != err {
			return nil, err
		}
	}

	db, err := gorm.Open("mysql", connStr)
	if err != nil {
		l.Print("init db err : ", connStr, err)
		return nil, err
	}

	db.DB().SetMaxOpenConns(poolSize)
	db.DB().SetMaxIdleConns(poolSize)
	db.LogMode(true)
	db.SetLogger(l)
	db.SingularTable(true)

	err = db.DB().Ping()
	if err != nil {
		l.Print("ping db error ", err)
		return nil, err
	}
	l.Print("init db pool OK")
	return db, nil
}

//ExternalProto ...
const ExternalProto = "ext"

func initDNSClient(addr string, timeout time.Duration, dns utils.DNSClient) error {
	dialer, err := getDialer(addr, timeout, dns)
	if nil != err {
		glog.Error(err)
		return err
	}
	mysql.RegisterDial(ExternalProto, dialer)
	return nil
}

func getDialer(addr string, timeout time.Duration, dns utils.DNSClient) (func(addr string) (net.Conn, error), error) {
	if dns == nil {
		return nil, fmt.Errorf("don,t support this dns is nil, %s", addr)
	}

	domain, port, err := net.SplitHostPort(addr)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if err := dns.Subscribe(domain); err != nil {
		glog.Error(err)
		return nil, err
	}

	return func(addr string) (net.Conn, error) {
		ip, err := dns.Query(domain)
		if err != nil {
			return nil, err
		}
		nd := net.Dialer{Timeout: timeout}
		conn, err := nd.Dial(`tcp`, net.JoinHostPort(ip, port))
		if err != nil {
			glog.Warningf("make connection to %s:%v failed", ip, port)
			return nil, &net.OpError{Op: "dial", Net: `tcp`, Addr: nil, Err: err}
		}
		return conn, nil
	}, nil
}
