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
	"fmt"
	"net/http"
	"os"

	"github.com/rajatjindal/prom-to-kairosdb/config"
	"github.com/rajatjindal/prom-to-kairosdb/kairosdb"
	"github.com/rajatjindal/prom-to-kairosdb/server"
	"github.com/spf13/cobra"
)

var cfgFile string
var cfg *config.Config

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
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME)")
	server.RegisterPrometheusMetrics()
	kairosdb.RegisterPrometheusMetrics()
}

func Main() {
	var err error
	cfg, err = config.ParseCfgFile(cfgFile)
	if err != nil {
		panic(err)
	}

	client := kairosdb.NewClient(cfg)
	serve(cfg.Server.Port, *client)
}

func serve(addr string, client kairosdb.Client) error {
	serverobj := &server.Server{
		Client: client,
	}

	http.Handle("/write", serverobj)

	err := http.ListenAndServe(addr, nil)
	fmt.Println(err)
	return err
}
