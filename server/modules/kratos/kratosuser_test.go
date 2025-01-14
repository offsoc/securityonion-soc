// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package kratos

import (
	"testing"
	"time"

	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/stretchr/testify/assert"
)

func TestCopyFromUser(tester *testing.T) {
	kratosUser := &KratosUser{}
	user := model.NewUser()
	user.Email = "my@email"
	user.FirstName = "myFirstname"
	user.LastName = "myLastname"
	user.Note = "myNote"
	user.Status = "locked"
	kratosUser.copyFromUser(user)
	assert.Equal(tester, user.Email, kratosUser.Traits.Email)
	assert.Equal(tester, user.FirstName, kratosUser.Traits.FirstName)
	assert.Equal(tester, user.LastName, kratosUser.Traits.LastName)
	assert.Equal(tester, user.Note, kratosUser.Traits.Note)
	assert.Equal(tester, "inactive", kratosUser.State)
	assert.Equal(tester, user.Email, kratosUser.Addresses[0].Value)
}

func TestCopyFromUserActive(tester *testing.T) {
	kratosUser := &KratosUser{}
	user := model.NewUser()
	user.Status = ""
	kratosUser.copyFromUser(user)
	assert.Equal(tester, "active", kratosUser.State)
}

func TestCopyToUser(tester *testing.T) {
	kratosUser := NewKratosUser("myEmail", "myFirst", "myLast", "note", "inactive")
	kratosUser.Credentials = make(map[string]*KratosCredential)
	kratosUser.Credentials["totp"] = &KratosCredential{Type: "totp"}
	kratosUser.Credentials["webauthn"] = &KratosCredential{Type: "webauthn"}
	oidcIds := make([]string, 1)
	oidcIds[0] = "test"
	kratosUser.Credentials["oidc"] = &KratosCredential{Type: "oidc", Identifiers: oidcIds}
	kratosUser.Credentials["password"] = &KratosCredential{Type: "password"}
	user := model.NewUser()
	kratosUser.copyToUser(user)
	assert.Equal(tester, kratosUser.Traits.Email, user.Email)
	assert.Equal(tester, kratosUser.Traits.FirstName, user.FirstName)
	assert.Equal(tester, kratosUser.Traits.LastName, user.LastName)
	assert.Equal(tester, kratosUser.Traits.Note, user.Note)
	assert.Equal(tester, kratosUser.Addresses[0].Value, user.Email)
	assert.Equal(tester, "locked", user.Status)
	assert.Equal(tester, "enabled", user.TotpStatus)
	assert.Equal(tester, "enabled", user.OidcStatus)
	assert.Equal(tester, "enabled", user.WebauthnStatus)
	assert.Equal(tester, false, user.PasswordChanged)

	kratosUser.Credentials = make(map[string]*KratosCredential)
	kratosUser.Credentials["password"] = &KratosCredential{
		Type:       "password",
		CreateDate: time.Now(),
		UpdateDate: time.Now().Add(time.Minute),
	}
	user = model.NewUser()
	kratosUser.copyToUser(user)
	assert.Equal(tester, "disabled", user.TotpStatus)
	assert.Equal(tester, true, user.PasswordChanged)
}

func TestCopyToUserActive(tester *testing.T) {
	kratosUser := NewKratosUser("myEmail", "myFirst", "myLast", "myNote", "active")
	user := model.NewUser()
	kratosUser.copyToUser(user)
	assert.Equal(tester, "", user.Status)
}
