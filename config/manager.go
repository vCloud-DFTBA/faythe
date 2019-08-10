package config

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Manager is a configuration manager. It maintains
// a global configuration, fetchs values to populate.
type Manager struct {
	configPath        string
	configPermissions os.FileMode

	config         *Config
	onConfigChange func(fsnotify.Event)
	mtx            sync.RWMutex
	log            *log.Logger
}

var m *Manager

func init() {
	m = NewManager()
}

// NewManager returns an initialized Manager instance
func NewManager() *Manager {
	m := new(Manager)
	m.configPermissions = os.FileMode(0644)
	m.config = &DefaultConfig
	m.log = log.New(os.Stdout, "config: ", log.LstdFlags)
	return m
}

// SetConfigPath sets the path of config file.
func SetConfigPath(in string) {
	m.SetConfigPath(in)
}

// SetConfigPath sets the path of config file.
func (m *Manager) SetConfigPath(in string) {
	if in != "" {
		m.configPath = in
	}
}

// SetConfigPermissions sets the permissions for the config file.
func SetConfigPermissions(perm os.FileMode) {
	m.SetConfigPermissions(perm)
}

// SetLog sets the manager's log instance.
func SetLog(log *log.Logger) {
	m.SetLog(log)
}

// SetLog sets the manager's log instance.
func (m *Manager) SetLog(log *log.Logger) {
	m.log = log
}

// SetConfigPermissions sets the permissions for the config file.
func (m *Manager) SetConfigPermissions(perm os.FileMode) {
	m.configPermissions = perm.Perm()
}

// Load parases the YAML input into a Config
func Load(in string) error {
	return m.Load(in)
}

// Load parses the YAML input s into a Config
func (m *Manager) Load(in string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	// Force reinit config
	m.config = new(Config)
	err := yaml.UnmarshalStrict([]byte(in), m.config)
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
	err = Load(string(content))
	if err != nil {
		return errors.Wrapf(err, "parsing YAML file %s", in)
	}
	return nil
}

// Get returns Config instance.
func Get() *Config {
	return m.Get()
}

// Get returns Config instance.
func (m *Manager) Get() *Config {
	m.mtx.RLock()
	conf := m.config
	m.mtx.RUnlock()
	return conf
}

// ShowString returns string represent of Config.
func ShowString() {
	m.ShowString()
}

// ShowString returns string represent of Config.
func (m *Manager) ShowString() {
	m.log.Println(m.config.String())
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
			m.log.Fatal(err)
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
						m.log.Printf("Config file %s is changeing...\n", m.configPath)
						err := m.LoadFile(m.configPath)
						if err != nil {
							m.log.Printf("error reading config file: %v\n", err)
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
						m.log.Printf("watcher error: %v\n", err)
					}
					eventsWG.Done()
					return
				}
			}
		}()
		watcher.Add(m.configPath)
		initWG.Done()
		eventsWG.Wait()
	}()
	initWG.Wait()
}

// Set updates the value of configuration. newConf should be a copy
// of m.config instance.
func Set(newConf *Config) {
	m.Set(newConf)
}

// Set updates the value of configuration. newConf should be a copy
// of m.config instance.
func (m *Manager) Set(newConf *Config) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.config = newConf
}

// Write writes the current configuration to a file.
func Write() error {
	return m.Write()
}

// Write writes the current configuration to a file.
func (m *Manager) Write() error {
	m.log.Println("Attempting to write configuration to file")

	raw, _ := yaml.Marshal(m.Get())
	err := ioutil.WriteFile(m.configPath, raw, m.configPermissions)
	if err != nil {
		return err
	}
	m.log.Println("Configuration file was updated")
	return nil
}
