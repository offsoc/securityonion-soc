// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package elastalert

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/security-onion-solutions/securityonion-soc/module"
	"github.com/security-onion-solutions/securityonion-soc/server"
	"github.com/security-onion-solutions/securityonion-soc/server/modules/elastalert/mock"
	"github.com/security-onion-solutions/securityonion-soc/util"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"
)

func TestElastAlertModule(t *testing.T) {
	srv := &server.Server{
		DetectionEngines: map[model.EngineName]server.DetectionEngine{},
	}
	mod := NewElastAlertEngine(srv)

	assert.Implements(t, (*module.Module)(nil), mod)
	assert.Implements(t, (*server.DetectionEngine)(nil), mod)

	err := mod.Init(nil)
	assert.NoError(t, err)

	err = mod.Start()
	assert.NoError(t, err)

	assert.True(t, mod.IsRunning())

	err = mod.Stop()
	assert.NoError(t, err)

	assert.Equal(t, 1, len(srv.DetectionEngines))
	assert.Same(t, mod, srv.DetectionEngines[model.EngineNameElastAlert])
}

func TestParseSigmaPackages(t *testing.T) {
	t.Parallel()

	table := []struct {
		Name     string
		Input    string
		Expected []string
	}{
		{
			Name:     "Simple Sunny Day Path",
			Input:    "core",
			Expected: []string{"core"},
		},
		{
			Name:     "Multiple Packages",
			Input:    "core+\nemerging_threats",
			Expected: []string{"core+", "emerging_threats_addon"},
		},
		{
			Name:     "Rename (all => all_rules)",
			Input:    "all",
			Expected: []string{"all_rules"},
		},
		{
			Name:     "Rename (emerging_threats_addon => emerging_threats)",
			Input:    "emerging_threats",
			Expected: []string{"emerging_threats_addon"},
		},
		{
			Name:     "Normalize",
			Input:    "CoRe++\n",
			Expected: []string{"core++"},
		},
		{
			Name:     "Account For Nesting Packages",
			Input:    "core\ncore+\ncore++\nall_rules\nemerging_threats",
			Expected: []string{"all_rules"},
		},
	}

	for _, tt := range table {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			engine := ElastAlertEngine{}

			engine.parseSigmaPackages(tt.Input)

			sort.Strings(engine.sigmaRulePackages)
			sort.Strings(tt.Expected)

			assert.Equal(t, tt.Expected, engine.sigmaRulePackages)
		})
	}
}

func TestTimeFrame(t *testing.T) {
	tf := TimeFrame{}

	tf.SetWeeks(1)
	assert.Equal(t, 1, *tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetDays(1)
	assert.Nil(t, tf.Weeks)
	assert.Equal(t, 1, *tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetHours(1)
	assert.Nil(t, tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Equal(t, 1, *tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetMinutes(1)
	assert.Nil(t, tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Equal(t, 1, *tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetSeconds(1)
	assert.Nil(t, tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Equal(t, 1, *tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetMilliseconds(1)
	assert.Nil(t, tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Equal(t, 1, *tf.Milliseconds)
	assert.Nil(t, tf.Schedule)

	tf.SetSchedule("0 0 0 * * *")
	assert.Nil(t, tf.Weeks)
	assert.Nil(t, tf.Days)
	assert.Nil(t, tf.Hours)
	assert.Nil(t, tf.Minutes)
	assert.Nil(t, tf.Seconds)
	assert.Nil(t, tf.Milliseconds)
	assert.Equal(t, "0 0 0 * * *", *tf.Schedule)

	tf.Schedule = nil // everything is now nil

	yml, err := yaml.Marshal(tf)
	assert.NoError(t, err)
	assert.Equal(t, "0\n", string(yml))

	err = yaml.Unmarshal(yml, &tf)
	assert.NoError(t, err)
	assert.Empty(t, tf)

	tf.SetWeeks(1)

	yml, err = yaml.Marshal(tf)
	assert.NoError(t, err)
	assert.Equal(t, "weeks: 1\n", string(yml))

	err = yaml.Unmarshal(yml, &tf)
	assert.NoError(t, err)
	assert.Equal(t, 1, *tf.Weeks)
}

func TestSigmaToElastAlertSunnyDay(t *testing.T) {
	ctrl := gomock.NewController(t)
	mio := mock.NewMockIOManager(ctrl)

	mio.EXPECT().ExecCommand(gomock.Cond(func(x any) bool {
		cmd := x.(*exec.Cmd)

		if !strings.HasSuffix(cmd.Path, "sigma") {
			return false
		}

		if !slices.Contains(cmd.Args, "convert") {
			return false
		}

		if cmd.Stdin == nil {
			return false
		}

		return true
	})).Return([]byte("<eql>"), 0, time.Duration(0), nil)

	engine := ElastAlertEngine{
		IOManager: mio,
	}

	det := &model.Detection{
		Auditable: model.Auditable{
			Id: "00000000-0000-0000-0000-000000000000",
		},
		Content:  "totally good sigma",
		Title:    "Test Detection",
		Severity: model.SeverityHigh,
	}

	wrappedRule, err := engine.sigmaToElastAlert(context.Background(), det)
	assert.NoError(t, err)

	expected := `play_title: Test Detection
play_id: 00000000-0000-0000-0000-000000000000
event.module: elastalert
event.dataset: elastalert.alert
event.severity: 4
rule.category: ""
sigma_level: high
alert:
    - modules.so.playbook-es.PlaybookESAlerter
index: .ds-logs-*
name: Test Detection - 00000000-0000-0000-0000-000000000000
type: any
filter:
    - eql: <eql>
play_url: play_url
kibana_pivot: kibana_pivot
soc_pivot: soc_pivot
`
	assert.YAMLEq(t, expected, wrappedRule)
}

func TestSigmaToElastAlertError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mio := mock.NewMockIOManager(ctrl)

	mio.EXPECT().ExecCommand(gomock.Cond(func(x any) bool {
		cmd := x.(*exec.Cmd)

		if !strings.HasSuffix(cmd.Path, "sigma") {
			return false
		}

		if !slices.Contains(cmd.Args, "convert") {
			return false
		}

		if cmd.Stdin == nil {
			return false
		}

		return true
	})).Return([]byte("Error: something went wrong"), 1, time.Duration(0), errors.New("non-zero return"))

	engine := ElastAlertEngine{
		IOManager: mio,
	}

	det := &model.Detection{
		Auditable: model.Auditable{
			Id: "00000000-0000-0000-0000-000000000000",
		},
		Content:  "totally good sigma",
		Title:    "Test Detection",
		Severity: model.SeverityHigh,
	}

	wrappedRule, err := engine.sigmaToElastAlert(context.Background(), det)
	assert.Empty(t, wrappedRule)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "problem with sigma cli")
}

func TestParseRules(t *testing.T) {
	t.Parallel()

	data := `title: Always Alert
id: 00000000-0000-0000-0000-00000000
status: experimental
description: Always Alerts
author: Corey Ogburn
date: 2023/11/03
modified: 2023/11/03
logsource:
    product: windows
detection:
    filter:
       event.module: "zeek"
    condition: "filter"
level: high
`

	buf := bytes.NewBuffer([]byte{})

	writer := zip.NewWriter(buf)
	aa, err := writer.Create("rules/always_alert.yml")
	assert.NoError(t, err)

	_, err = aa.Write([]byte(data))
	assert.NoError(t, err)

	bad, err := writer.Create("rules/bad.yml")
	assert.NoError(t, err)

	_, err = bad.Write([]byte("bad data"))
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	pkgZips := map[string][]byte{
		"all_rules": buf.Bytes(),
	}

	engine := ElastAlertEngine{}

	expected := &model.Detection{
		PublicID:    "00000000-0000-0000-0000-00000000",
		Title:       "Always Alert",
		Severity:    model.SeverityHigh,
		Content:     data,
		IsCommunity: true,
		Engine:      model.EngineNameElastAlert,
		Language:    model.SigLangSigma,
		Ruleset:     util.Ptr("all_rules"),
	}

	dets, errMap := engine.parseRules(pkgZips)
	assert.NotNil(t, errMap)
	assert.Error(t, errMap["rules/bad.yml"])
	assert.Len(t, dets, 1)
	assert.Equal(t, expected, dets[0])
}

func TestDownloadSigmaPackages(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mio := mock.NewMockIOManager(ctrl)
	body := "data"

	for i := 0; i < 5; i++ {
		// can't use mock's Times(x) because the first response's body
		// closing will result in remaining requests getting 0 data
		mio.EXPECT().MakeRequest(gomock.Any()).Return(&http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
		}, nil)
	}

	mio.EXPECT().MakeRequest(gomock.Any()).Return(&http.Response{
		StatusCode: http.StatusNotFound,
	}, nil)

	pkgs := []string{"core", "core+", "core++", "emerging_threats_addon", "all_rules", "fake"}

	engine := ElastAlertEngine{
		sigmaRulePackages:            pkgs,
		sigmaPackageDownloadTemplate: "localhost:3000/%s.zip",
		IOManager:                    mio,
	}

	pkgZips, errMap := engine.downloadSigmaPackages(context.Background())
	assert.NotNil(t, errMap)
	assert.Error(t, errMap["fake"])
	assert.Len(t, pkgZips, len(pkgs)-1)

	for _, pkg := range pkgs[:len(pkgs)-1] {
		assert.Equal(t, []byte(body), pkgZips[pkg])
	}
}

const (
	SimpleRuleSID = "10000"
	SimpleRule    = `title: Griffon Malware Attack Pattern
	id: bcc6f179-11cd-4111-a9a6-0fab68515cf7
	status: experimental
	description: Detects process execution patterns related to Griffon malware as reported by Kaspersky
	references:
			- https://securelist.com/fin7-5-the-infamous-cybercrime-rig-fin7-continues-its-activities/90703/
	author: Nasreddine Bencherchali (Nextron Systems)
	date: 2023/03/09
	tags:
			- attack.execution
			- detection.emerging_threats
	logsource:
			category: process_creation
			product: windows
	detection:
			selection:
					CommandLine|contains|all:
							- '\local\temp\'
							- '//b /e:jscript'
							- '.txt'
			condition: selection
	falsepositives:
			- Unlikely
	level: critical`
)

type MockDirEntry struct {
	name  string
	isDir bool
	typ   fs.FileMode
}

func (mde *MockDirEntry) Name() string {
	return mde.name
}

func (mde *MockDirEntry) IsDir() bool {
	return mde.isDir
}

func (mde *MockDirEntry) Type() fs.FileMode {
	return mde.typ
}

func (mde *MockDirEntry) ModTime() time.Time {
	return time.Now()
}

func (mde *MockDirEntry) Mode() fs.FileMode {
	return mde.typ
}

func (mde *MockDirEntry) Size() int64 {
	return 100
}

func (mde *MockDirEntry) Sys() any {
	return nil
}

func (mde *MockDirEntry) Info() (fs.FileInfo, error) {
	return mde, nil
}

func TestSyncElastAlert(t *testing.T) {
	t.Parallel()

	table := []struct {
		Name           string
		Detections     []*model.Detection
		InitMock       func(*ElastAlertEngine, *mock.MockIOManager)
		ExpectedErr    error
		ExpectedErrMap map[string]string
	}{
		{
			Name: "Enable New Simple Rule",
			Detections: []*model.Detection{
				{
					PublicID:  SimpleRuleSID,
					Content:   SimpleRule,
					IsEnabled: true,
					Title:     "TEST",
					Auditable: model.Auditable{
						Id: SimpleRuleSID,
					},
					Severity: model.SeverityMedium,
				},
			},
			InitMock: func(mod *ElastAlertEngine, m *mock.MockIOManager) {
				// IndexExistingRules
				m.EXPECT().ReadDir(mod.elastAlertRulesFolder).Return([]fs.DirEntry{}, nil)
				// sigmaToElastAlert
				m.EXPECT().ExecCommand(gomock.Any()).Return([]byte("[sigma rule]"), 0, time.Duration(0), nil)
				// WriteFile when enabling
				m.EXPECT().WriteFile("10000.yml", []byte("play_title: TEST\nplay_id: \"10000\"\nevent.module: elastalert\nevent.dataset: elastalert.alert\nevent.severity: 3\nrule.category: \"\"\nsigma_level: medium\nalert:\n    - modules.so.playbook-es.PlaybookESAlerter\nindex: .ds-logs-*\nname: TEST - 10000\ntype: any\nfilter:\n    - eql: '[sigma rule]'\nplay_url: play_url\nkibana_pivot: kibana_pivot\nsoc_pivot: soc_pivot\n"), fs.FileMode(0644)).Return(nil)
			},
		},
		{
			Name: "Disable Simple Rule",
			Detections: []*model.Detection{
				{
					PublicID:  SimpleRuleSID,
					IsEnabled: false,
				},
			},
			InitMock: func(mod *ElastAlertEngine, m *mock.MockIOManager) {
				// IndexExistingRules
				filename := SimpleRuleSID + ".yml"
				m.EXPECT().ReadDir(mod.elastAlertRulesFolder).Return([]fs.DirEntry{
					&MockDirEntry{
						name: filename,
					},
					&MockDirEntry{
						name:  "ignored_dir",
						isDir: true,
					},
					&MockDirEntry{
						name: "ignored.txt",
					},
				}, nil)
				// DeleteFile when disabling
				m.EXPECT().DeleteFile(filename).Return(nil)
			},
		},
		{
			Name: "Enable Rule w/Override",
			Detections: []*model.Detection{
				{
					PublicID:  SimpleRuleSID,
					Content:   SimpleRule,
					IsEnabled: true,
					Title:     "TEST",
					Auditable: model.Auditable{
						Id: SimpleRuleSID,
					},
					Severity: model.SeverityMedium,
					Overrides: []*model.Override{
						{
							Type:      model.OverrideTypeCustomFilter,
							IsEnabled: false,
							OverrideParameters: model.OverrideParameters{
								CustomFilter: util.Ptr("FALSE"),
							},
						},
						{
							Type:      model.OverrideTypeCustomFilter,
							IsEnabled: true,
							OverrideParameters: model.OverrideParameters{
								CustomFilter: util.Ptr("TRUE"),
							},
						},
					},
				},
			},
			InitMock: func(mod *ElastAlertEngine, m *mock.MockIOManager) {
				// IndexExistingRules
				m.EXPECT().ReadDir(mod.elastAlertRulesFolder).Return([]fs.DirEntry{}, nil)
				// sigmaToElastAlert
				m.EXPECT().ExecCommand(gomock.Any()).Return([]byte("[sigma rule]"), 0, time.Duration(0), nil)
				// WriteFile when enabling
				m.EXPECT().WriteFile("10000.yml", []byte("play_title: TEST\nplay_id: \"10000\"\nevent.module: elastalert\nevent.dataset: elastalert.alert\nevent.severity: 3\nrule.category: \"\"\nsigma_level: medium\nalert:\n    - modules.so.playbook-es.PlaybookESAlerter\nindex: .ds-logs-*\nname: TEST - 10000\ntype: any\nfilter:\n    - eql: ([sigma rule]) and TRUE\nplay_url: play_url\nkibana_pivot: kibana_pivot\nsoc_pivot: soc_pivot\n"), fs.FileMode(0644)).Return(nil)
			},
		},
	}

	ctx := context.Background()

	for _, test := range table {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			mockIO := mock.NewMockIOManager(ctrl)

			mod := NewElastAlertEngine(&server.Server{
				DetectionEngines: map[model.EngineName]server.DetectionEngine{},
			})

			mod.IOManager = mockIO
			mod.srv.DetectionEngines[model.EngineNameElastAlert] = mod

			if test.InitMock != nil {
				test.InitMock(mod, mockIO)
			}

			errMap, err := mod.SyncLocalDetections(ctx, test.Detections)

			assert.Equal(t, test.ExpectedErr, err)
			assert.Equal(t, test.ExpectedErrMap, errMap)
		})
	}
}
