// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package httpserver

import (
	"errors"
	"fmt"
	"time"

	"github.com/cogment/cogment/version"
	"github.com/golang-jwt/jwt/v5"
)

var TokenIssuer = fmt.Sprintf("Cogment v%s", version.Version)

type TokenClaims struct {
	TrialID   string `json:"trial_id"`
	ActorName string `json:"actor_name"`
	jwt.RegisteredClaims
}

func MakeAndSerializeToken(trialID string, actorName string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		TrialID:   trialID,
		ActorName: actorName,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   TokenIssuer,
		},
	})
	return token.SignedString([]byte(secret))
}

func ParseAndVerifyToken(tokenString string, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		issuer, err := token.Claims.GetIssuer()
		if err != nil {
			return nil, err
		}

		if issuer != TokenIssuer {
			return nil, fmt.Errorf("Unexpected token issuer: %v", issuer)
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok {
		return claims, nil
	}
	return nil, errors.New("Unexpected token claims")
}
