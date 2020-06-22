// etcdadapter will simulate the table structure of Relational DB in ETCD which is a kv-based storage.
// Under a basic path, we will build a key for each policy, and the value is the Json format string for each Casbin Rule.

package etcdadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	client "go.etcd.io/etcd/clientv3"
	clientnamespace "go.etcd.io/etcd/clientv3/namespace"
)

const (
	REQUESTTIMEOUT = 5 * time.Second

	// PLACEHOLDER represent the NULL value in the Casbin Rule.
	PLACEHOLDER = "_"

	// DEFAULT_KEY is the root path in ETCD, if not provided.
	DEFAULT_KEY = "casbin_policy"
)

type CasbinRule struct {
	Key   string `json:"key"`
	PType string `json:"ptype"`
	V0    string `json:"v0"`
	V1    string `json:"v1"`
	V2    string `json:"v2"`
	V3    string `json:"v3"`
	V4    string `json:"v4"`
	V5    string `json:"v5"`
}

// Adapter represents the ETCD adapter for policy storage.
type Adapter struct {
	etcdCfg client.Config
	key     string

	// etcd connection client
	conn *client.Client
}

func NewAdapter(etcdCfg client.Config, namespace string, key string) *Adapter {
	return newAdapter(etcdCfg, namespace, key)
}

func newAdapter(etcdCfg client.Config, namespace string, key string) *Adapter {
	if key == "" {
		key = DEFAULT_KEY
	}
	a := &Adapter{
		etcdCfg: etcdCfg,
		key:     key,
	}
	a.connect(namespace)

	// Call the destructor when the object is released.
	runtime.SetFinalizer(a, finalizer)

	return a
}

func (a *Adapter) connect(namespace string) {
	connection, err := client.New(a.etcdCfg)
	if err != nil {
		panic(err)
	}
	if namespace != "" {
		connection.Watcher = clientnamespace.NewWatcher(connection.Watcher, namespace)
		connection.Lease = clientnamespace.NewLease(connection.Lease, namespace)
		connection.KV = clientnamespace.NewKV(connection.KV, namespace)
	}
	a.conn = connection
}

// finalizer is the destructor for Adapter.
func finalizer(a *Adapter) {
	a.conn.Close()
}

func (a *Adapter) close() {
	a.conn.Close()
}

// LoadPolicy loads all of policys from ETCD
func (a *Adapter) LoadPolicy(model model.Model) error {
	var rule CasbinRule
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	getResp, err := a.conn.Get(ctx, a.getRootKey(), client.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range getResp.Kvs {
		err = json.Unmarshal(kv.Value, &rule)
		if err != nil {
			return err
		}
		a.loadPolicy(rule, model)
	}
	return nil
}

func (a *Adapter) getRootKey() string {
	return fmt.Sprintf("/%s", a.key)
}

func (a *Adapter) loadPolicy(rule CasbinRule, model model.Model) {
	lineText := rule.PType
	if rule.V0 != "" {
		lineText += ", " + rule.V0
	}
	if rule.V1 != "" {
		lineText += ", " + rule.V1
	}
	if rule.V2 != "" {
		lineText += ", " + rule.V2
	}
	if rule.V3 != "" {
		lineText += ", " + rule.V3
	}
	if rule.V4 != "" {
		lineText += ", " + rule.V4
	}
	if rule.V5 != "" {
		lineText += ", " + rule.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

// This will rewrite all of policies in ETCD with the current data in Casbin
func (a *Adapter) SavePolicy(model model.Model) error {
	// clean old rule data
	a.destroy()

	var rules []CasbinRule

	for ptype, ast := range model["p"] {
		for _, line := range ast.Policy {
			rules = append(rules, a.convertRule(ptype, line))
		}
	}

	for ptype, ast := range model["g"] {
		for _, line := range ast.Policy {
			rules = append(rules, a.convertRule(ptype, line))
		}
	}

	return a.savePolicy(rules)
}

// destroy or clean all of policy
func (a *Adapter) destroy() error {
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	_, err := a.conn.Delete(ctx, a.getRootKey(), client.WithPrefix())
	return err
}

func (a *Adapter) convertRule(ptype string, line []string) (rule CasbinRule) {
	rule = CasbinRule{}
	rule.PType = ptype
	policys := []string{ptype}
	length := len(line)

	if len(line) > 0 {
		rule.V0 = line[0]
		policys = append(policys, line[0])
	}
	if len(line) > 1 {
		rule.V1 = line[1]
		policys = append(policys, line[1])
	}
	if len(line) > 2 {
		rule.V2 = line[2]
		policys = append(policys, line[2])
	}
	if len(line) > 3 {
		rule.V3 = line[3]
		policys = append(policys, line[3])
	}
	if len(line) > 4 {
		rule.V4 = line[4]
		policys = append(policys, line[4])
	}
	if len(line) > 5 {
		rule.V5 = line[5]
		policys = append(policys, line[5])
	}

	for i := 0; i < 6-length; i++ {
		policys = append(policys, PLACEHOLDER)
	}

	rule.Key = strings.Join(policys, "::")

	return rule
}

func (a *Adapter) savePolicy(rules []CasbinRule) error {
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	for _, rule := range rules {
		ruleData, _ := json.Marshal(rule)
		_, err := a.conn.Put(ctx, a.constructPath(rule.Key), string(ruleData))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) constructPath(key string) string {
	return fmt.Sprintf("/%s/%s", a.key, key)
}

// AddPolicy adds a policy rule to the storage.
// Part of the Auto-Save feature.
func (a *Adapter) AddPolicy(sec string, ptype string, line []string) error {
	rule := a.convertRule(ptype, line)
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	ruleData, _ := json.Marshal(rule)
	_, err := a.conn.Put(ctx, a.constructPath(rule.Key), string(ruleData))
	return err
}

// AddPolicies adds policy rules to the storage.
// This is part of the Auto-Save feature.
func (a *Adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.AddPolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

// RemovePolicy removes a policy rule from the storage.
// Part of the Auto-Save feature.
func (a *Adapter) RemovePolicy(sec string, ptype string, line []string) error {
	rule := a.convertRule(ptype, line)
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	_, err := a.conn.Delete(ctx, a.constructPath(rule.Key))
	return err
}

// RemovePolicies removes policy rules from the storage.
// This is part of the Auto-Save feature.
func (a *Adapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.RemovePolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// Part of the Auto-Save feature.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	rule := CasbinRule{}

	rule.PType = ptype
	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		rule.V0 = fieldValues[0-fieldIndex]
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		rule.V1 = fieldValues[1-fieldIndex]
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		rule.V2 = fieldValues[2-fieldIndex]
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		rule.V3 = fieldValues[3-fieldIndex]
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		rule.V4 = fieldValues[4-fieldIndex]
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		rule.V5 = fieldValues[5-fieldIndex]
	}

	filter := a.constructFilter(rule)

	return a.removeFilteredPolicy(filter)
}

func (a *Adapter) constructFilter(rule CasbinRule) string {
	var filter string
	if rule.PType != "" {
		filter = fmt.Sprintf("/%s/%s", a.key, rule.PType)
	} else {
		filter = fmt.Sprintf("/%s/.*", a.key)
	}

	if rule.V0 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V0)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	if rule.V1 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V1)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	if rule.V2 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V2)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	if rule.V3 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V3)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	if rule.V4 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V4)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	if rule.V5 != "" {
		filter = fmt.Sprintf("%s::%s", filter, rule.V5)
	} else {
		filter = fmt.Sprintf("%s::.*", filter)
	}

	return filter
}

func (a *Adapter) removeFilteredPolicy(filter string) error {
	ctx, cancel := context.WithTimeout(context.Background(), REQUESTTIMEOUT)
	defer cancel()
	// get all policy key
	getResp, err := a.conn.Get(ctx, a.constructPath(""), client.WithPrefix(), client.WithKeysOnly())
	if err != nil {
		return err
	}
	var filteredKeys []string
	for _, kv := range getResp.Kvs {
		matched, err := regexp.MatchString(filter, string(kv.Key))
		if err != nil {
			return err
		}
		if matched {
			filteredKeys = append(filteredKeys, string(kv.Key))
		}
	}

	for _, key := range filteredKeys {
		_, err := a.conn.Delete(ctx, key)
		if err != nil {
			return err
		}
	}
	return nil
}
