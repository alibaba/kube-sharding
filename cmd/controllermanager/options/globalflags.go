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

package options

import (
	goflag "flag"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"io"

	"github.com/docker/docker/pkg/term"

	apiserverflag "k8s.io/component-base/cli/flag"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
)

// AddGlobalFlags copy all global flags (flag & pflag) to a local flagset.
func AddGlobalFlags(fs *pflag.FlagSet, name string) {
	local := goflag.CommandLine
	local.VisitAll(func(fl *goflag.Flag) {
		fl.Name = normalize(fl.Name)
		fs.AddGoFlag(fl)
	})
	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", name))
}

// AddFlags copy all global flags (flag & pflag) to a local flagset.
func AddFlags(fs *pflag.FlagSet, lfs *goflag.FlagSet) {
	lfs.VisitAll(func(fl *goflag.Flag) {
		fl.Name = normalize(fl.Name)
		fs.AddGoFlag(fl)
	})
}

func normalize(s string) string {
	return strings.Replace(s, "_", "-", -1)
}

func terminalSize(w io.Writer) (int, int, error) {
	outFd, isTerminal := term.GetFdInfo(w)
	if !isTerminal {
		return 0, 0, fmt.Errorf("given writer is no terminal")
	}
	winsize, err := term.GetWinsize(outFd)
	if err != nil {
		return 0, 0, err
	}
	return int(winsize.Width), int(winsize.Height), nil
}

// PrintNamedFlagsetsSections PrintNamedFlagsetsSections
func PrintNamedFlagsetsSections(namedFlagSets apiserverflag.NamedFlagSets, cmd *cobra.Command, name string) {
	fs := cmd.Flags()
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}
	cmd.InitDefaultVersionFlag()
	namedFlagSets.FlagSet("global").AddFlag(cmd.Flags().Lookup("version"))
	AddGlobalFlags(namedFlagSets.FlagSet("global"), name)
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := terminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
}
