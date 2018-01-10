// Copyright © 2017 Proofpoint Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/proofpoint/prom-to-kairosdb/config"
	"github.com/proofpoint/prom-to-kairosdb/kairosdb"
	"github.com/proofpoint/prom-to-kairosdb/server"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dryRun  bool
	debug   bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "prom-to-kairos",
	Short: "remote storate adapter for KairosDB",

	Run: func(cmd *cobra.Command, args []string) {
		Main()
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Errorf("%s", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME)")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dryrun", false, "if set to true, dont push metrics to downstream")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "if set to true, print debug level logs")

	server.RegisterPrometheusMetrics()
	kairosdb.RegisterPrometheusMetrics()
}

func Main() {
	cfg, err := config.ParseCfgFile(cfgFile)
	if err != nil {
		logrus.Errorf("%s", err)
		os.Exit(-1)
	}

	if debug {
		cfg.Debug = true
		logrus.SetLevel(logrus.DebugLevel)
	}

	client := kairosdb.NewClient(cfg)
	serve(cfg.Server.Port, *client)
}

func serve(addr string, client kairosdb.Client) error {
	serverobj := &server.Server{
		Client: client,
	}

	http.Handle("/write", serverobj)
	http.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(addr, nil)
	logrus.Errorf("%s", err)
	return err
}
