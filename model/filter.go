// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package model

import (
	"time"
)

const PROTOCOL_ICMP = "icmp"
const PROTOCOL_TCP = "tcp"
const PROTOCOL_UDP = "udp"

type Filter struct {
	ImportId   string                 `json:"importId"`
	BeginTime  time.Time              `json:"beginTime"`
	EndTime    time.Time              `json:"endTime"`
	SrcIp      string                 `json:"srcIp"`
	SrcPort    int                    `json:"srcPort"`
	DstIp      string                 `json:"dstIp"`
	DstPort    int                    `json:"dstPort"`
	Protocol   string                 `json:"protocol"`
	Parameters map[string]interface{} `json:"parameters"`
}

func NewFilter() *Filter {
	return &Filter{
		Parameters: make(map[string]interface{}),
	}
}
