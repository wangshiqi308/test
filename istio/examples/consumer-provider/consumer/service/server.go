/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	wg    *sync.WaitGroup
	muxer *mux.Router
}

func NewServer(wg *sync.WaitGroup) *Server {
	s := &Server{
		wg: wg,
	}

	// add one job to wait group
	s.wg.Add(1)
	return s
}

func (s *Server) WithMuxer() {
	s.muxer = mux.NewRouter()
	s.muxer.HandleFunc("/sqrt", s.HandleSqrt).Methods("GET")
	s.muxer.HandleFunc("/healthz/ready", s.HandleHealth).Methods("GET")
}

func (s *Server) Start() {
	s.WithMuxer()
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	fmt.Println("Consumer started!!")
	handler := c.Handler(s.muxer)
	http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", 8080), handler)

	s.wg.Done()
}
