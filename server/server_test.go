// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package server

import (
	"testing"

	"github.com/security-onion-solutions/securityonion-soc/config"
	"github.com/security-onion-solutions/securityonion-soc/licensing"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(tester *testing.T) {
	licensing.Test("foo", 0, 0, "", "")
	cfg := &config.ServerConfig{}
	srv := NewServer(cfg, "")
	assert.NotNil(tester, srv.Host)
	assert.NotNil(tester, srv.stoppedChan)
	assert.Equal(tester, licensing.LICENSE_STATUS_ACTIVE, licensing.GetStatus())
}

func TestNewServer_SocUrlExceeded(tester *testing.T) {
	licensing.Test("foo", 0, 0, "foo", "")
	cfg := &config.ServerConfig{}
	srv := NewServer(cfg, "")
	assert.NotNil(tester, srv.Host)
	assert.NotNil(tester, srv.stoppedChan)
	assert.Equal(tester, licensing.LICENSE_STATUS_EXCEEDED, licensing.GetStatus())
}

func TestDeveloperAuthorization(tester *testing.T) {
	cfg := &config.ServerConfig{}
	srv := NewServer(cfg, "")
	cfg.DeveloperEnabled = true

	authErr := srv.CheckAuthorized(nil, "read", "users")
	assert.NoError(tester, authErr)
}

func TestMissingAuthorization(tester *testing.T) {
	cfg := &config.ServerConfig{}
	srv := NewServer(cfg, "")

	authErr := srv.CheckAuthorized(nil, "read", "users")
	assert.Error(tester, authErr)
	assert.Equal(tester, "Missing Authorizer module", authErr.Error())
}

func TestFailedAuthorization(tester *testing.T) {
	srv := NewFakeUnauthorizedServer()

	authErr := srv.CheckAuthorized(nil, "read", "users")
	assert.Error(tester, authErr)
	assert.Contains(tester, authErr.Error(), "not authorized to perform operation 'read' on target 'users'")
}
