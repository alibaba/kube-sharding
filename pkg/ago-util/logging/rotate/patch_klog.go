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

package rotate

import (
	"flag"
	"os"
	"strconv"

	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

// Patch klog to use this library to write logs

var (
	program = filepath.Base(os.Args[0])
	// KloggerName is the klog logger name in rotate.LoggerFactory
	KloggerName = "klog-internal"
	maxBackups  int
)

// PatchKlogFlags add extend flags for klog, call after klog.InitFlags
func PatchKlogFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	flagset.IntVar(&maxBackups, "log_file_backups", 10, "Log file max backups")
	setDefaultFlagValues(flagset)
}

// PatchForKlog init logger from klog flags and replace the klog ouptut.
// Call this after flag parse
func PatchForKlog(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	logger := createLogFromFlags(flagset)
	klog.SetOutput(logger)
	factory.add(KloggerName, logger)
}

// SetupKlogByLogger set klog by a logger in factory
func SetupKlogByLogger(name string, args ...string) {
	wc := Must(name)
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	klog.InitFlags(fs)
	setDefaultFlagValues(fs)
	fs.Parse(args)
	klog.SetOutput(wc)
}

// called before parse, user flags can overwrite this
func setDefaultFlagValues(flagset *flag.FlagSet) {
	// turn off stderr
	flagset.Set("stderrthreshold", "FATAL")
	// because we write logs to custom file, set one output.
	flagset.Set("one_output", "true")
	// don't write stderr only
	flagset.Set("logtostderr", "false")
}

func createLogFromFlags(flagset *flag.FlagSet) *lumberjack.Logger {
	logger := &lumberjack.Logger{
		MaxSize:    500,
		MaxBackups: maxBackups,
		LocalTime:  true,
	}
	f := flagset.Lookup("log_dir")
	if f.Value.String() != "" {
		logger.Filename = filepath.Join(f.Value.String(), program+".log")
	}
	f = flagset.Lookup("log_file")
	if f.Value.String() != "" {
		logger.Filename = f.Value.String()
	}
	f = flagset.Lookup("log_file_max_size")
	if f.Value.String() != "" {
		i, err := strconv.Atoi(f.Value.String())
		if err == nil && i > 0 {
			logger.MaxSize = i
		}
	}
	return logger
}
