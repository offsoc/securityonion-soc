// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package web

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type BasePreprocessor struct {
}

func NewBasePreprocessor() *BasePreprocessor {
	return &BasePreprocessor{}
}

func (Processor *BasePreprocessor) PreprocessPriority() int {
	return 0
}

func (processor *BasePreprocessor) Preprocess(ctx context.Context, req *http.Request) (context.Context, int, error) {
	uuid := uuid.New().String()
	ctx = context.WithValue(ctx, ContextKeyRequestId, uuid)
	return ctx, 0, nil
}
