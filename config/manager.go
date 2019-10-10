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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Manager is a configuration manager. It maintains
// a global configuration, fetchs values to populate.
type Manager struct {
	configPath        string
	configPermissions os.FileMode
	Config            *Config
	onConfigChange    func(fsnotify.Event)
	mtx               sync.RWMutex
	logger            log.Logger
}

// NewManager returns initialized Manager instance
func NewManager() *Manager {
	return &Manager{
		configPermissions: os.FileMode(0644),
		Config:            &DefaultConfig,
	}
}

var m *Manager

func init() {
	m = NewManager()
}

// SetConfigPath sets the path of config file
func SetConfigPath(in string) {
	m.SetConfigPath(in)
}

// SetConfigPath sets the path of config file
func (m *Manager) SetConfigPath(in string) {
	if in != "" {
		m.configPath = in
	}
}

// SetConfigPermissions sets the permissions for the config file.
func SetConfigPermissions(perm os.FileMode) {
	m.SetConfigPermissions(perm)
}

// SetConfigPermissions sets the permissions for the config file.
func (m *Manager) SetConfigPermissions(perm os.FileMode) {
	m.configPermissions = perm.Perm()
}

// SetLogger sets the manager's log instance.
func SetLogger(l log.Logger) {
	m.SetLogger(l)
}

// SetLogger sets the manager's log instance.
func (m *Manager) SetLogger(l log.Logger) {
	m.logger = l
}

// Load parses the YAML input s into a Config
func Load(in string) error {
	return m.Load(in)
}

// Load parses the YAML input s into a Config
func (m *Manager) Load(in string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	// Force reinit config
	m.Config = new(Config)
	err := yaml.UnmarshalStrict([]byte(in), m.Config)
	if err != nil {
		return err
	}
	return nil
}

// LoadFile parses the given YAMl file into a Config.
func LoadFile(in string) error {
	return m.LoadFile(in)
}

// LoadFile parses the given YAMl file into a Config.
func (m *Manager) LoadFile(in string) error {
	content, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}
	m.SetConfigPath(in)
	err = m.Load(string(content))
	if err != nil {
		return errors.Wrapf(err, "parsing YAML file %s", m.configPath)
	}
	return nil
}

// Set sets logger and loads Config from the given file path.
func Set(fp string, l log.Logger) error {
	return m.Set(fp, l)
}

// Set sets logger and loads Config from the given file path.
func (m *Manager) Set(fp string, l log.Logger) error {
	m.SetLogger(l)
	return m.LoadFile(fp)
}

// Show returns the represent of Config.
// For debug purpose only
func Show() {
	m.Show()
}

// Show returns the represent of Config.
// For debug purpose only
func (m *Manager) Show() {
	fmt.Println(m.Config.String())
}

// OnConfigChange defines the function be called when config file was change.
func OnConfigChange(run func(in fsnotify.Event)) {
	m.OnConfigChange(run)
}

// OnConfigChange defines the function be called when config file was change.
func (m *Manager) OnConfigChange(run func(in fsnotify.Event)) {
	m.onConfigChange = run
}

// WatchConfig detects file changes and reloads config.
func WatchConfig() {
	m.WatchConfig()
}

// WatchConfig detects file changes and reloads config.
func (m *Manager) WatchConfig() {
	initWG := sync.WaitGroup{}
	initWG.Add(1)
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			level.Error(m.logger).Log("msg", "Error creating config file watcher", "err", err)
		}
		defer watcher.Close()

		eventsWG := sync.WaitGroup{}
		eventsWG.Add(1)
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok { // 'Event' channel is closed
						eventsWG.Done()
						return
					}
					const writeOrCreateMask = fsnotify.Write | fsnotify.Create
					if filepath.Clean(event.Name) == m.configPath &&
						event.Op&writeOrCreateMask != 0 {
						level.Info(m.logger).Log("msg", "Config file is changing...")
						err := m.LoadFile(m.configPath)
						if err != nil {
							level.Error(m.logger).Log("msg", "Error reading config file", "err", err)
						}
						if m.onConfigChange != nil {
							m.onConfigChange(event)
						}
					} else if filepath.Clean(event.Name) == m.configPath &&
						event.Op&fsnotify.Remove != 0 {
						eventsWG.Done()
						return
					}

				case err, ok := <-watcher.Errors:
					if ok { // 'Errors' channel is not closed
						level.Error(m.logger).Log("msg", "Error closing config file watcher", "err", err)
					}
					eventsWG.Done()
					return
				}
			}
		}()
		err = watcher.Add(m.configPath)
		if err != nil {
			level.Error(m.logger).Log("msg", "Error adding config file watcher", "err", err)
		}
		initWG.Done()
		eventsWG.Wait()
	}()
	initWG.Wait()
}

// Get returns Config instance.
func Get() *Config {
	return m.Get()
}

// Get returns Config instance.
func (m *Manager) Get() *Config {
	m.mtx.RLock()
	conf := m.Config
	m.mtx.RUnlock()
	return conf
}

// SetConfig updates the value of configuration. newConf should be a copy
// of m.config instance.
func SetConfig(newConf *Config) {
	m.SetConfig(newConf)
}

// SetConfig updates the value of configuration. newConf should be a copy
// of m.config instance.
func (m *Manager) SetConfig(newConf *Config) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.Config = newConf
}

// Write writes the currnet configuration to a file
func Write() error {
	return m.Write()
}

// Write writes the current configuration to a file.
func (m *Manager) Write() error {
	level.Info(m.logger).Log("msg", "Attempting to write configuration to file")

	raw, _ := yaml.Marshal(m.Get())
	err := ioutil.WriteFile(m.configPath, raw, m.configPermissions)
	if err != nil {
		return err
	}
	level.Info(m.logger).Log("msg", "Configuration file was updated")
	return nil
}
