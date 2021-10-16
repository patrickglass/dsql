// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/mattn/go-isatty"
	"github.com/prometheus/client_golang/prometheus"
	buildVersion "github.com/prometheus/common/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Specification struct {
	Debug       bool
	Port        int `default:"8080"`
	MetricsPort int `default:"8080"`
}

func init() {
	// export build information as dsql_build_info via prometheus
	prometheus.MustRegister(buildVersion.NewCollector("dsql"))
}

// ConfigureGlobalLogger will configure zerolog globally. It will
// enable pretty printing for interactive terminals and json for production.
func configureGlobalLogger() {
	// for tty terminal enable pretty logs
	if isatty.IsTerminal(os.Stdout.Fd()) && runtime.GOOS != "windows" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		// UNIX Time is faster and smaller than most timestamps
		// If you set zerolog.TimeFieldFormat to an empty string,
		// logs will write with UNIX time.
		zerolog.TimeFieldFormat = ""
	}
}

func main() {
	var s Specification

	// Configure logger based on terminal type
	configureGlobalLogger()

	log.Info().
		Str("version", buildVersion.Version).
		Str("revision", buildVersion.Revision).
		Str("branch", buildVersion.Branch).
		Str("build_user", buildVersion.BuildUser).
		Str("build_date", buildVersion.BuildDate).
		Str("go_version", buildVersion.GoVersion).
		Msg("dsql application starting")

	err := envconfig.Process("dsql", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not process environment variables")
	}

	if s.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Debug().
		Bool("Debug", s.Debug).
		Int("Port", s.Port).
		Int("MetricsPort", s.MetricsPort).
		Msg("dsql configuration")

	fmt.Println("Hello. Welcome to DSQL")
}
