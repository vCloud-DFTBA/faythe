// Copyright (c) 2019 Tuan-Dat Vu<tuandatk25a@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"crypto"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

type Silence struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Pattern      string         `json:"pattern"`
	RegexPattern *regexp.Regexp `json:"-"`
	TTL          string         `json:"ttl"`
	Tags         []string       `json:"tags"`
	Description  string         `json:"description"`
	CreatedAt    time.Time      `json:"created_at"`
	ExpiredAt    time.Time      `json:"expired_at"`
	CreatedBy    string         `json:"created_by"`
}

func (s *Silence) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("silence name cannot be empty")
	}

	if s.Pattern == "" {
		return fmt.Errorf("silence pattern cannot be empty")
	}

	regex, err := regexp.Compile(s.Pattern)
	if err != nil {
		return err
	}
	s.RegexPattern = regex

	if s.TTL == "" && !s.CreatedAt.IsZero() && !s.ExpiredAt.IsZero() {
		durationHours := int(math.RoundToEven(s.ExpiredAt.Sub(s.CreatedAt).Hours()))
		s.TTL = strconv.Itoa(durationHours) + "h"
	} else if s.TTL != "" {
		t, err := common.ParseDuration(s.TTL)
		if err != nil {
			return err
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now()
		}
		s.ExpiredAt = s.CreatedAt.Add(t)
	} else {
		return fmt.Errorf("silence ttl, created_at and expired_at cannot be both empty")
	}

	s.ID = common.Hash(fmt.Sprintf("%s-%s", s.Pattern, s.ExpiredAt.String()), crypto.MD5)

	return nil
}
