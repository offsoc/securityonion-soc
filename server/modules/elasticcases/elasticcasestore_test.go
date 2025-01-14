// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package elasticcases

import (
	"context"
	"testing"

	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/security-onion-solutions/securityonion-soc/server"
	"github.com/stretchr/testify/assert"
)

func TestCreateUnauthorized(tester *testing.T) {
	casestore := NewElasticCasestore(server.NewFakeUnauthorizedServer())
	casestore.Init("some/url", "someusername", "somepassword", true)
	socCase := model.NewCase()
	newCase, err := casestore.Create(context.Background(), socCase)
	assert.Error(tester, err)
	assert.Nil(tester, newCase)
}

func TestCreate(tester *testing.T) {
	casestore := NewElasticCasestore(server.NewFakeAuthorizedServer(nil))
	casestore.Init("some/url", "someusername", "somepassword", true)
	caseResponse := `
    {
      "id": "a123",
      "title": "my title"
    }`
	casestore.client.MockStringResponse(caseResponse, 200, nil)
	socCase := model.NewCase()
	newCase, err := casestore.Create(context.Background(), socCase)
	assert.NoError(tester, err)

	assert.Equal(tester, "my title", newCase.Title)
	assert.Equal(tester, "a123", newCase.Id)
}
