// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package model

import (
	"time"
)

type User struct {
	Id              string    `json:"id"`
	CreateTime      time.Time `json:"createTime"`
	UpdateTime      time.Time `json:"updateTime"`
	Email           string    `json:"email"`
	FirstName       string    `json:"firstName"`
	LastName        string    `json:"lastName"`
	TotpStatus      string    `json:"totpStatus"`
	OidcStatus      string    `json:"oidcStatus"`
	WebauthnStatus  string    `json:"webauthnStatus"`
	Note            string    `json:"note"`
	Roles           []string  `json:"roles"`
	Status          string    `json:"status"`
	SearchUsername  string    `json:"searchUsername"`
	Password        string    `json:"password"`
	PasswordChanged bool      `json:"passwordChanged"`
}

func NewUser() *User {
	return &User{
		CreateTime:     time.Now(),
		Email:          "",
		FirstName:      "",
		LastName:       "",
		Note:           "",
		Status:         "",
		SearchUsername: "",
		Password:       "",
	}
}

func (user *User) String() string {
	return user.Id
}
