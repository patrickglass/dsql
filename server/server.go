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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/jackc/pgproto3/v2"
	"github.com/rs/zerolog/log"
)

type Option func(*Server)

type Server struct {
	listener  net.Listener
	tlsConfig *tls.Config
	address   string
	quit      chan interface{}
	wg        sync.WaitGroup

	backend  string
	tenantID string
}

func New(opts ...Option) (*Server, error) {
	s := Server{
		address: ":5432",
		quit:    make(chan interface{}),
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

func WithQueryServiceBackend(hostname string) Option {
	return func(s *Server) {
		s.backend = hostname
	}
}

func WithTenant(tenant string) Option {
	return func(s *Server) {
		s.tenantID = tenant
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

func (s *Server) Serve() error {
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

listenerLoop:
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				log.Info().Msg("gracefully exiting sql server")
				break listenerLoop
			default:
				log.Error().Err(err).Msg("connection failure")
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			handleConnection(conn, s)
			s.wg.Done()
		}()
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.quit)
	s.listener.Close()
	s.wg.Wait()
	return nil
}

func handleConnection(conn net.Conn, s *Server) {
	remoteAddr := conn.RemoteAddr().String()
	log.Debug().Str("address", remoteAddr).Msg("accepted connection")

	b := NewDataQueryBackend(conn, func(query *pgproto3.Query) (QueryResponse, error) {

		resp := FetchMetrics(query.String, s.backend, s.tenantID)
		return resp, nil
	})

	err := b.Run()
	if err != nil {
		log.Error().Err(err)
	}
	log.Debug().Str("address", remoteAddr).Msgf("connection closed")
}

// DataQueryBackend
type DataQueryBackend struct {
	backend   *pgproto3.Backend
	conn      net.Conn
	responder func(*pgproto3.Query) (QueryResponse, error)
}

func NewDataQueryBackend(conn net.Conn, responder func(*pgproto3.Query) (QueryResponse, error)) *DataQueryBackend {
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

		switch msg := msg.(type) {
		case *pgproto3.Query:
			log.Info().Str("query", msg.String).Msg("sql query")

			// Build response
			response, err := b.responder(msg)
			if err != nil {
				log.Error().Err(err).Str("query", msg.String).Msg("response error")
				return fmt.Errorf("error generating query response: %w", err)
			}

			fields := []pgproto3.FieldDescription{}
			for _, field := range response.Headers {
				fields = append(fields, pgproto3.FieldDescription{
					Name:                 []byte(field),
					TableOID:             0,
					TableAttributeNumber: 0,
					DataTypeOID:          25,
					DataTypeSize:         -1,
					TypeModifier:         -1,
					Format:               0,
				})
			}
			buf := (&pgproto3.RowDescription{Fields: fields}).Encode(nil)

			for _, row := range response.Rows {
				rowData := make([][]byte, len(row))
				for i, col := range row {
					switch value := col.(type) {
					case nil:
						rowData[i] = []byte("NULL")
					case string:
						rowData[i] = []byte(value)
					case int:
						rowData[i] = []byte(fmt.Sprintf("%d", value))
					case float64:
						rowData[i] = []byte(fmt.Sprintf("%f", value))
					default:
						fmt.Printf("Unknown Data Type %T!\n", value)
						rowData[i] = []byte("UNKNOWN")
					}
				}
				buf = append(buf, (&pgproto3.DataRow{Values: rowData}).Encode(nil)...)
			}

			// Comand Tag should be the command which is executed for non selects
			// Insert 0 1
			// Update 1
			// Delete 1
			buf = (&pgproto3.CommandComplete{CommandTag: []byte("SELECT")}).Encode(buf)
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
		log.Debug().Msg("send startup message")
		// Do not require auth
		buf := (&pgproto3.AuthenticationOk{}).Encode(nil)
		// Indicate backend is Idle and able to accept queries
		buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf)
		_, err = p.conn.Write(buf)
		if err != nil {
			return fmt.Errorf("error sending ready for query: %w", err)
		}
	case *pgproto3.SSLRequest:
		log.Debug().Msg("deny request to use SSL")
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny SSL request: %w", err)
		}
		return p.handleStartup()
	default:
		log.Info().Str("message", reflect.TypeOf(startupMessage).String()).Msg("unknown startup message")
		return fmt.Errorf("unknown startup message: %#v", startupMessage)
	}

	return nil
}

func (p *DataQueryBackend) Close() error {
	return p.conn.Close()
}
