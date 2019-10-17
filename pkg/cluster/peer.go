// Copyright (c) 2019 Kien Nguyen-Tuan <kiennt2609@gmail.com>
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

package cluster

import (
	"io"
	"os"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/hashicorp/serf/serf"
)

// Peer starts and manage a Serf instance
type Peer struct {
	// Stores the serf configuration
	conf *serf.Config
	// This is underlying Serf we are wrapping
	serf *serf.Serf

	// eventCh is used for Serf to deliver events on
	eventCh chan serf.Event
	// eventHandlers is the registered handlers for events
	eventHandlers     map[EventHandler]struct{}
	eventHandlerList  []EventHandler

	logger log.Logger

	// shutdownCh is used for shutdowns
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
}

// Create creates a new peer, potentially returning an error
func Create(c *serf.Config, l log.Logger, lo io.Writer) (*Peer, error) {
	if l == nil {
		l = log.NewNopLogger()
	}
	if lo == nil {
		lo = os.Stderr
	}

	// Setup the underlying loggers
	c.MemberlistConfig.LogOutput = lo
	c.LogOutput = lo

	// Create a channel to listen for events from Serf
	eventCh := make(chan serf.Event, 64)
	c.EventCh = eventCh

	// Setup the peer
	p := &Peer{
		conf: c,
		logger: l,
		eventCh: eventCh,

	}
}
