// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package syntax

import (
	"errors"
	"strings"
)

func Validate(value string, syntax string) error {
	var err error

	if (strings.Contains(value, "{#") && strings.Contains(value, "#}")) ||
		(strings.Contains(value, "{{") && strings.Contains(value, "}}")) ||
		(strings.Contains(value, "{%") && strings.Contains(value, "%}")) {
		err = errors.New("ERROR_JINJA_NOT_SUPPORTED")
	} else {
		switch strings.ToLower(syntax) {
		case "yaml", "yml":
			err = ValidateYaml(value)
		case "json", "suricata":
			err = ValidateJson(value)
		}
	}

	return err
}
