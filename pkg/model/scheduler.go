package model

import (
	"crypto"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

// Scheduler represents a Scheduler object
type Scheduler struct {
	Description    string                     `json:"description"`
	Actions        map[string]ActionInterface `json:"ractions"`
	ActionsRaw     map[string]json.RawMessage `json:"actions"`
	Tags           []string                   `json:"tags"`
	Active         bool                       `json:"active"`
	ID             string                     `json:"id,omitempty"`
	CloudID        string                     `json:"cloudid"`
	CreatedBy      string                     `json:"created_by"`
	FromDate       string                     `json:"from_date"`
	ToDate         string                     `json:"to_date"`
	FromCronSlices string                     `json:"from_cron_slices"`
	ToCronSlices   string                     `json:"to_cron_slices"`
	FromDateTime   time.Time                  `json:"from_date_time,omitempty"`
	ToDateTime     time.Time                  `json:"to_date_time,omitempty"`
	FromNextExec   time.Time                  `json:"from_next_exec,omitempty"`
	ToNextExec     time.Time                  `json:"to_next_exec,omitempty"`
}

// Validate returns nil if all fields of the Scheduler have valid values.
func (s *Scheduler) Validate() error {
	if s.ActionsRaw != nil {
		s.Actions = make(map[string]ActionInterface, len(s.ActionsRaw))
		for k, v := range s.ActionsRaw {
			a := Action{}
			if err := json.Unmarshal(v, &a); err != nil {
				return err
			}
			switch strings.ToLower(a.Type) {
			case "http":
				ah := &ActionHTTP{}
				if err := json.Unmarshal(v, ah); err != nil {
					return err
				}
				s.Actions[k] = ah
			default:
				return fmt.Errorf("type %s is not supported", a.Type)
			}
			if err := s.Actions[k].Validate(); err != nil {
				return err
			}
		}
	}

	fromSchedule, err := cron.ParseStandard(s.FromCronSlices)

	if err != nil {
		return err
	}

	toSchedule, err := cron.ParseStandard(s.ToCronSlices)

	if err != nil {
		return err
	}

	s.FromDateTime, err = time.Parse("2006-01-02 15:04:05-07:00", s.FromDate)

	if err != nil {
		return err
	}

	s.ToDateTime, err = time.Parse("2006-01-02 15:04:05-07:00", s.ToDate)

	if err != nil {
		return err
	}

	s.FromNextExec = fromSchedule.Next(time.Now().UTC())
	s.ToNextExec = toSchedule.Next(time.Now().UTC())

	s.ID = common.Hash(strings.Join(s.Tags, "."), crypto.MD5)

	return nil
}

func (s *Scheduler) ForwardFromNextExec() {
	now := time.Now().UTC()
	fromSchedule, _ := cron.ParseStandard(s.FromCronSlices)
	// FromNextExec still in the past
	if s.FromNextExec.Before(now) {
		s.FromNextExec = fromSchedule.Next(now)
	}
}

func (s *Scheduler) ForwardToNextExec() {
	now := time.Now().UTC()
	toSchedule, _ := cron.ParseStandard(s.ToCronSlices)

	// ToNextExec still in the past
	if s.ToNextExec.Before(now) {
		s.ToNextExec = toSchedule.Next(now)
	}
}

func (s *Scheduler) IsExpired() bool {
	now := time.Now().UTC()
	return s.ToDateTime.Before(now)
}

func (s *Scheduler) IsActive() bool {
	now := time.Now().UTC()
	return s.FromDateTime.Before(now) && s.Active
}
