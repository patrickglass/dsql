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

package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"os/exec"

	"github.com/jackc/pgproto3/v2"
	"github.com/rs/zerolog/log"
)

type Option func(*Server)

type Server struct {
	listener  net.Listener
	tlsConfig *tls.Config
	address   string
}

func New(opts ...Option) (*Server, error) {
	s := Server{
		address: ":5432",
	}
	for _, opt := range opts {
		opt(&s)
	}
	return &s, nil
}

// WithAddress sets the listener address
func WithAddress(address string) Option {
	return func(s *Server) {
		s.address = address
	}
}

// WithPort will set the listener address to any interface on the specified port
func WithPort(port int) Option {
	return func(s *Server) {
		s.address = fmt.Sprintf(":%d", port)
	}
}

func WithTLSCert(s *Server, cert tls.Certificate) Option {
	return func(s *Server) {
		s.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}
}

func (s *Server) Listen() error {
	var err error
	var ln net.Listener

	if s.tlsConfig != nil {
		ln, err = tls.Listen("tcp", s.address, s.tlsConfig)
	} else {
		ln, err = net.Listen("tcp", s.address)
	}
	if err != nil {
		return err
	}
	s.listener = ln

	// Handle connections forever
	s.HandleConnections()

	// NOTE: this only get called if listener dies
	return nil
}

func (s *Server) HandleConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// handle error
			log.Error().Err(err).Msg("connection failure")
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Info().Str("address", remoteAddr).Msg("accepted connection")

	b := NewDataQueryBackend(conn, func() ([]byte, error) {
		return exec.Command("sh", "-c", "fortune | cowsay -f elephant").CombinedOutput()
	})
	go func() {
		err := b.Run()
		if err != nil {
			log.Error().Err(err)
		}
		log.Info().Str("address", remoteAddr).Msgf("connection closed")
	}()
}

// DataQueryBackend
type DataQueryBackend struct {
	backend   *pgproto3.Backend
	conn      net.Conn
	responder func() ([]byte, error)
}

func NewDataQueryBackend(conn net.Conn, responder func() ([]byte, error)) *DataQueryBackend {
	backend := pgproto3.NewBackend(pgproto3.NewChunkReader(conn), conn)

	connHandler := &DataQueryBackend{
		backend:   backend,
		conn:      conn,
		responder: responder,
	}

	return connHandler
}

func (b *DataQueryBackend) Run() error {
	defer b.Close()

	err := b.handleStartup()
	if err != nil {
		return err
	}

	for {
		msg, err := b.backend.Receive()
		if err != nil {
			return fmt.Errorf("error receiving message: %w", err)
		}

		switch msg.(type) {
		case *pgproto3.Query:
			response, err := b.responder()
			if err != nil {
				return fmt.Errorf("error generating query response: %w", err)
			}

			buf := (&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{
				{
					Name:                 []byte("fortune"),
					TableOID:             0,
					TableAttributeNumber: 0,
					DataTypeOID:          25,
					DataTypeSize:         -1,
					TypeModifier:         -1,
					Format:               0,
				},
			}}).Encode(nil)
			buf = (&pgproto3.DataRow{Values: [][]byte{response}}).Encode(buf)
			buf = (&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")}).Encode(buf)
			buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf)
			_, err = b.conn.Write(buf)
			if err != nil {
				return fmt.Errorf("error writing query response: %w", err)
			}
		case *pgproto3.Terminate:
			return nil
		default:
			return fmt.Errorf("received message other than Query from client: %#v", msg)
		}
	}
}

func (p *DataQueryBackend) handleStartup() error {
	startupMessage, err := p.backend.ReceiveStartupMessage()
	if err != nil {
		return fmt.Errorf("error receiving startup message: %w", err)
	}

	switch startupMessage.(type) {
	case *pgproto3.StartupMessage:
		// Do not require auth
		buf := (&pgproto3.AuthenticationOk{}).Encode(nil)
		// Indicate backend is Idle and able to accept queries
		buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf)
		_, err = p.conn.Write(buf)
		if err != nil {
			return fmt.Errorf("error sending ready for query: %w", err)
		}
	case *pgproto3.SSLRequest:
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny SSL request: %w", err)
		}
		return p.handleStartup()
	default:
		return fmt.Errorf("unknown startup message: %#v", startupMessage)
	}

	return nil
}

func (p *DataQueryBackend) Close() error {
	return p.conn.Close()
}
