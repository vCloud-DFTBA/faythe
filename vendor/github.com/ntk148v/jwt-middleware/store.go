// Copyright (c) 2020 Kien Nguyen-Tuan <kiennt2609@gmail.com>
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

package jwt

type Store interface {
	// Check checks whether a token has been revoked
	// If not, it will return some user data and nil.
	Check(tokenId string, issuedAt float64) (data map[string]interface{}, err error)
	// Revoke revokes a token which is no longer in use.
	// This case often happens when a user logs out.
	// or an authorization ends.
	Revoke(tokenId string) error
}
