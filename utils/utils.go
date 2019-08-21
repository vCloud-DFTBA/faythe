package utils

import (
	"crypto/sha256"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/alertmanager/template"

	"faythe/config"
)

// Flogger represents a file logger.
type Flogger struct {
	file string
	*log.Logger
}

// SharedValue stores a sharable cached value between requests.
type SharedValue struct {
	lock sync.RWMutex
	Data map[string]interface{}
}

// NewSharedValue returns a new instance SharedValue.
func NewSharedValue() *SharedValue {
	return &SharedValue{Data: make(map[string]interface{})}
}

// Get returns the data of SharedValue by a given key.
func (sv *SharedValue) Get(key string) (interface{}, bool) {
	sv.lock.RLock()
	defer sv.lock.RUnlock()
	d, ok := sv.Data[key]
	return &d, ok
}

// Set inserts/updates the data by a given key.
func (sv *SharedValue) Set(key string, d interface{}) {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	sv.Data[key] = d
}

// Delete removes a data from SharedValue by a given key.
func (sv *SharedValue) Delete(key string) {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	delete(sv.Data, key)
}

func createFlogger(fname string) *Flogger {
	logDir := config.Get().ServerConfig.LogDir
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			panic(err)
		}
	}

	fpath := filepath.Join(logDir, fname)
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		os.Exit(1)
	}

	return &Flogger{
		file:   fname,
		Logger: log.New(f, "handlers: ", log.Lshortfile|log.LstdFlags),
	}
}

// NewFlogger inits a file logger.
func NewFlogger(once *sync.Once, fname string) *Flogger {
	var logger *Flogger
	once.Do(func() {
		logger = createFlogger(fname)
	})
	return logger
}

// Getenv returns default value if environment variable
// doesn't exist.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Hash compute SHA1 hashes of s given input.
func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return string(h.Sum(nil))
}

// LookupAddr performs a reverse lookup for the network address of the form "host:port"/network address/hostname
// returning a name mapping to that address.
func LookupAddr(host string) (string, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil && !strings.Contains("missingPort", err.Error()) {
		return "", errors.Wrap(err, "parse host from hostport failed")
	}
	// Determine whether a given string is an ip or hostname
	addr := net.ParseIP(host)
	if addr == nil {
		return host, nil
	}
	hostname, err := net.LookupAddr(host)
	if err != nil {
		return "", errors.Wrap(err, "lookup adddress failed")
	}
	// Force get the first result, ignore the rest.
	return hostname[0], nil
}

// UpdateExistingAlerts stores a set of existing alerts and updates
// these by removing resolved alerts from list.
func UpdateExistingAlerts(existingAlerts *SharedValue, data *template.Data, logger *Flogger) {
	resolvedAlerts := data.Alerts.Resolved()
	for _, alert := range resolvedAlerts {
		// Generate a simple fingerprint aka signature
		// that represents for Alert.
		av := append(alert.Labels.Values(), alert.StartsAt.String())
		fingerprint := Hash(strings.Join(av, "_"))
		// Remove Alert if it is already resolved.
		if _, ok := existingAlerts.Get(fingerprint); ok {
			logger.Printf("Alert %s/%s was resolved, delete it from existing alerts list.",
				alert.Labels["alertname"],
				alert.Labels["instance"])
			existingAlerts.Delete(fingerprint)
		}
	}
}
