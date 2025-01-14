// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package generichttp

import (
	"strings"
	"text/template"

	"github.com/security-onion-solutions/securityonion-soc/model"
)

func convertCaseToReader(source string, socCase *model.Case) (*strings.Reader, error) {
	builder := new(strings.Builder)
	parsedTemplate, err := template.New("case").Parse(source)
	if err == nil {
		err = parsedTemplate.Execute(builder, socCase)
	}
	return strings.NewReader(builder.String()), err
}
