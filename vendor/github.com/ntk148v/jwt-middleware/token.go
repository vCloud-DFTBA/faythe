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

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
)

type Token struct {
	privateKey    interface{}
	publicKey     interface{}
	signingMethod jwt.SigningMethod
	store         Store
	options       Options
}

type TokenInfo struct {
	Id       string
	IssuedAt float64
	Data     map[string]interface{}
}

// Claims holds the claims encoded in a JWT
type Claims struct {
	// Standard claims are the standard jwt claims from the ietf standard
	// https://tools.ietf.org/html/rfc7519
	*jwt.StandardClaims
	Data map[string]interface{} `json:"data,omitempty"`
}

// NewToken constructs a new Token instance
func NewToken(o Options, s Store) (*Token, error) {
	signingMethod := jwt.GetSigningMethod(o.SigningMethod)

	var (
		privateKey interface{}
		publicKey  interface{}
		err        error
	)

	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		if o.HMACKey == nil {
			return nil, ErrNoHMACKey
		}
		privateKey = o.HMACKey
		publicKey = o.HMACKey
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		if o.PrivateKeyLocation == "" || o.PublicKeyLocation == "" {
			return nil, ErrNoRSAKey
		}
		privateKey, publicKey, err = getRSAKeys(o.PrivateKeyLocation, o.PublicKeyLocation)
		if err != nil {
			return nil, err
		}
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		if o.PrivateKeyLocation == "" || o.PublicKeyLocation == "" {
			return nil, ErrNoECKey
		}
		privateKey, publicKey, err = getECKeys(o.PrivateKeyLocation, o.PublicKeyLocation)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidSigningMethod
	}
	if o.UserProperty == "" {
		o.UserProperty = "user"
	}
	return &Token{
		privateKey:    privateKey,
		publicKey:     publicKey,
		signingMethod: signingMethod,
		store:         s,
		options:       o,
	}, nil
}

// GetToken extracts a token string from the header.
func (t *Token) GetToken(req *http.Request) (string, error) {
	if t.options.IsBearerToken {
		return request.AuthorizationHeaderExtractor.ExtractToken(req)
	}
	header := req.Header.Get(t.options.Header)
	return header, nil
}

// GenerateToken generate a JWT.
// Please don't add sensitive data such as password to payload of JWT.
func (t *Token) GenerateToken(data map[string]interface{}) (string, error) {
	id, err := generateRandString(32)
	if err != nil {
		return "", fmt.Errorf("JWT: unable to generate JWT id, %s", err)
	}
	claims := Claims{
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(t.options.TTL).Unix(),
			IssuedAt:  time.Now().Unix(),
			Id:        id,
		},
		Data: data,
	}
	unsigned := jwt.NewWithClaims(t.signingMethod, claims)
	return unsigned.SignedString(t.privateKey)
}

func (t *Token) CheckToken(tokenString string) (map[string]interface{}, error) {
	tokenInfo, err := t.validateJWT(tokenString)
	if err != nil {
		return nil, err
	}
	// When there is no storage, we should like to return information from token.
	if t.store == nil {
		return tokenInfo.Data, nil
	}
	return t.store.Check(tokenInfo.Id, tokenInfo.IssuedAt)
}

// ValidateJWT validates whether a jwt string is valid.
// If so, it returns data included in the token and nil error.
func (t *Token) ValidateToken(tokenString string) (*TokenInfo, error) {
	return t.ValidateToken(tokenString)
}

// RevokeToken revokes a token which is no longer in use.
// This case often happens when a user logs out.
// or an authorization ends.
func (t *Token) RevokeToken(id string) error {
	if t.store == nil {
		return ErrNoStore
	}
	return t.store.Revoke(id)
}

// RefreshToken regenerate the token after check the given token
// string is valid.
func (t *Token) RefreshToken(tokenString string) (string, error) {
	tokenInfo, err := t.validateJWT(tokenString)
	if err != nil {
		return "", err
	}
	return t.GenerateToken(tokenInfo.Data)
}

func (t *Token) validateJWT(tokenString string) (*TokenInfo, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm is what you expect
		if token.Method.Alg() != t.signingMethod.Alg() {
			return nil, ErrUnexpectedSigningMethod
		}
		return t.publicKey, nil
	})
	if err != nil {
		if e, ok := err.(*jwt.ValidationError); ok {
			switch {
			case e.Errors&jwt.ValidationErrorMalformed != 0:
				// Token is malformed
				return nil, ErrTokenMalformed
			case e.Errors&jwt.ValidationErrorExpired != 0:
				// Token is expired
				return nil, ErrTokenExpired
			case e.Errors&jwt.ValidationErrorNotValidYet != 0:
				// Token is not active yet
				return nil, ErrTokenNotActive
			case e.Inner != nil:
				// report e.Inner
				return nil, e.Inner
			}
		}
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["jti"] == nil || claims["iat"] == nil {
		return nil, ErrInvalidToken
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return nil, ErrGetTokenId
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, ErrGetIssuedTime
	}

	if claims["data"] == nil {
		return nil, nil
	}

	data, ok := claims["data"].(map[string]interface{})
	if !ok {
		return nil, ErrGetData
	}
	return &TokenInfo{
		Id:       jti,
		IssuedAt: iat,
		Data:     data,
	}, nil
}

func getKeyContent(keyLocation string) ([]byte, error) {
	keyContent, err := ioutil.ReadFile(keyLocation)
	if err != nil {
		return nil, fmt.Errorf("JWT: failed to load a key from %s, %s", keyLocation, err)
	}
	return keyContent, nil
}

func getRSAKeys(privateKeyLocation, publicKeyLocation string) (interface{}, interface{}, error) {
	privateKeyContent, err := getKeyContent(privateKeyLocation)
	if err != nil {
		return nil, nil, err
	}
	publicKeyContent, err := getKeyContent(publicKeyLocation)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyContent)
	if err != nil {
		return nil, nil, fmt.Errorf("JWT: failed to generate a private RSA key, %s", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyContent)
	if err != nil {
		return nil, nil, fmt.Errorf("JWT: failed to generate a public RSA key, %s", err)
	}
	return privateKey, publicKey, nil
}

func getECKeys(privateKeyLocation, publicKeyLocation string) (interface{}, interface{}, error) {
	privateKeyContent, err := getKeyContent(privateKeyLocation)
	if err != nil {
		return nil, nil, err
	}
	publicKeyContent, err := getKeyContent(publicKeyLocation)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(privateKeyContent)
	if err != nil {
		return nil, nil, fmt.Errorf("JWT: failed to generate a private EC key, %s", err)
	}
	publicKey, err := jwt.ParseECPublicKeyFromPEM(publicKeyContent)
	if err != nil {
		return nil, nil, fmt.Errorf("JWT: failed to generate a public EC key, %s", err)
	}
	return privateKey, publicKey, nil
}

func generateRandString(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return base64.URLEncoding.EncodeToString(b), err
}
