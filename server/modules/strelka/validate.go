// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package strelka

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/security-onion-solutions/securityonion-soc/util"
)

type parseState int

const (
	parseStateImportsID parseState = iota
	parseStateWatchForHeader
	parseStateInSection
)

type YaraRule struct {
	Imports    []string
	Identifier string
	Meta       Metadata
	Strings    []string
	Condition  string
	Src        string
}

type Metadata struct {
	ID          *string
	Author      *string
	Date        *string
	Version     *string
	Reference   *string
	Description *string
	Rest        map[string]string
}

func (md *Metadata) IsEmpty() bool {
	return md.Author == nil && md.Date == nil && md.Version == nil && md.Reference == nil && md.Description == nil && len(md.Rest) == 0
}

func (md *Metadata) Set(key, value string) {
	key = strings.ToLower(key)

	value = util.Unquote(value)

	switch key {
	case "id":
		md.ID = util.Ptr(value)
	case "author":
		md.Author = util.Ptr(value)
	case "date":
		md.Date = util.Ptr(value)
	case "version":
		md.Version = util.Ptr(value)
	case "reference":
		md.Reference = util.Ptr(value)
	case "description":
		md.Description = util.Ptr(value)
	default:
		if md.Rest == nil {
			md.Rest = make(map[string]string)
		}
		md.Rest[key] = value
	}
}

func (rule *YaraRule) GetID() string {
	if rule.Meta.ID != nil {
		return util.Unquote(*rule.Meta.ID)
	}

	hash := sha256.Sum256([]byte(rule.Identifier))

	hash[6] = 0x40 | (hash[6] & 0x0f)
	hash[8] = 0x80 | (hash[8] & 0x3f)
	id := fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		hash[0], hash[1], hash[2], hash[3], hash[4], hash[5], hash[6], hash[7], hash[8],
		hash[9], hash[10], hash[11], hash[12], hash[13], hash[14], hash[15])

	rule.Meta.ID = util.Ptr(id)

	return id
}

func (r *YaraRule) String() string {
	buffer := bytes.NewBuffer([]byte{})

	// imports
	for _, i := range r.Imports {
		line := fmt.Sprintf("import \"%s\"\n", i)
		buffer.WriteString(line)
	}

	if len(r.Imports) > 0 {
		buffer.WriteString("\n")
	}

	// identifier
	buffer.WriteString(fmt.Sprintf("rule %s {\n", r.Identifier))

	// meta
	if !r.Meta.IsEmpty() {
		buffer.WriteString("\tmeta:\n")

		if r.Meta.Author != nil {
			buffer.WriteString(fmt.Sprintf("\t\tauthor = \"%s\"\n", *r.Meta.Author))
		}

		if r.Meta.Date != nil {
			buffer.WriteString(fmt.Sprintf("\t\tdate = \"%s\"\n", *r.Meta.Date))
		}

		if r.Meta.Version != nil {
			buffer.WriteString(fmt.Sprintf("\t\tversion = \"%s\"\n", *r.Meta.Version))
		}

		if r.Meta.Reference != nil {
			buffer.WriteString(fmt.Sprintf("\t\treference = \"%s\"\n", *r.Meta.Reference))
		}

		if r.Meta.Description != nil {
			buffer.WriteString(fmt.Sprintf("\t\tdescription = \"%s\"\n", *r.Meta.Description))
		}

		keys := []string{}
		for k := range r.Meta.Rest {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			buffer.WriteString(fmt.Sprintf("\t\t%s = \"%s\"\n", k, r.Meta.Rest[k]))
		}

		buffer.WriteString("\n")
	}

	// strings
	if len(r.Strings) > 0 {
		buffer.WriteString("\tstrings:\n")

		for _, s := range r.Strings {
			buffer.WriteString(fmt.Sprintf("\t\t%s\n", s))
		}
	}

	// condition and closing bracket
	buffer.WriteString(fmt.Sprintf("\n\tcondition:\n\t\t%s\n}", r.Condition))

	return buffer.String()
}

func (r *YaraRule) Validate() error {
	missing := []string{}

	if r.Identifier == "" {
		missing = append(missing, "identifier")
	}

	if r.Condition == "" {
		missing = append(missing, "condition")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	return nil
}
