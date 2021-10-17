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
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/patrickglass/dsql/server"

	"github.com/kelseyhightower/envconfig"
	"github.com/mattn/go-isatty"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	buildVersion "github.com/prometheus/common/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Specification struct {
	Debug           bool
	DevelopmentMode bool
	PublicKeyFile   string `default:"./server.pem"`
	PrivateKeyFile  string `default:"./server.key"`
	Port            int    `default:"5432"`
	MetricsPort     int    `default:"5480"`
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

func StartServer(s Specification) error {
	if s.DevelopmentMode && s.PrivateKeyFile == "" && s.PublicKeyFile == "" {
		log.Fatal().Msg("DSQL_PRIVATEKEY and DSQL_PUBLICKEY must be set unless in Development mode")
	}

	// Start the prometheus server
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", s.MetricsPort), nil)
	log.Info().Int("port", s.MetricsPort).Msg("prometheus metrics started")

	// cert, err := tls.LoadX509KeyPair(s.PublicKeyFile, s.PrivateKeyFile)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("could not parse certificates as PEM files")
	// }
	// cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	// listener, err := tls.Listen("tcp", s.Port, cfg)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("could not start listener")
	// }

	sqlServer, err := server.New(
		server.WithPort(s.Port),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not start create server")
	}

	log.Info().Int("port", s.Port).Msg("Starting dsql server")
	log.Info().Msg("Connect to server with: psql -h localhost -w -c 'select 1'")
	sqlServer.Listen()
	return err
}

func cmdServer() error {
	var s Specification

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
		Bool("DevelopmentMode", s.DevelopmentMode).
		Str("PublicKeyFile", s.PublicKeyFile).
		Str("PrivateKeyFile", s.PrivateKeyFile).
		Int("Port", s.Port).
		Int("MetricsPort", s.MetricsPort).
		Msg("dsql configuration")

	fmt.Println("Hello. Welcome to DSQL")
	return StartServer(s)
}

func cmdCertGen() error {
	pub, priv := server.GenKey()
	_, _ = pub, priv
	return nil
}

func main() {
	// Configure logger based on terminal type
	configureGlobalLogger()

	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal().Msg("must specify command as the first argument")
		flag.Usage()
		os.Exit(1)
	}

	switch cmd := flag.Arg(0); cmd {
	case "server":
		cmdServer()
	case "gencert":
		cmdCertGen()
	default:
		log.Fatal().Msgf("invalid command: '%s', must be one of 'server' or `gencert`", cmd)
	}
}
