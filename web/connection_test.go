// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUpdatePingTime(tester *testing.T) {
	conn := NewConnection(nil, nil, "")
	oldPingTime := conn.lastPingTime
	time.Sleep(3 * time.Millisecond)
	conn.UpdatePingTime()
	newPingTime := conn.lastPingTime

	assert.True(tester, newPingTime.Sub(oldPingTime).Milliseconds() >= 3, "expected 3s increase in lastPingTime")
}
