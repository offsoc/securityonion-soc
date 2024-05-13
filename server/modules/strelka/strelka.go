// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package strelka

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/security-onion-solutions/securityonion-soc/module"
	"github.com/security-onion-solutions/securityonion-soc/server"
	"github.com/security-onion-solutions/securityonion-soc/server/modules/detections"
	"github.com/security-onion-solutions/securityonion-soc/util"
	"github.com/security-onion-solutions/securityonion-soc/web"

	"github.com/apex/log"
	"github.com/kennygrant/sanitize"
)

const (
	DEFAULT_ALLOW_REGEX                              = ""
	DEFAULT_DENY_REGEX                               = ""
	DEFAULT_COMMUNITY_RULES_IMPORT_FREQUENCY_SECONDS = 86400
	DEFAULT_YARA_RULES_FOLDER                        = "/opt/sensoroni/yara/rules"
	DEFAULT_REPOS_FOLDER                             = "/opt/sensoroni/yara/repos"
	DEFAULT_COMPILE_YARA_PYTHON_SCRIPT_PATH          = "/opt/so/conf/strelka/compile_yara.py"
	DEFAULT_COMPILE_RULES                            = true
	DEFAULT_STATE_FILE_PATH                          = "/opt/sensoroni/fingerprints/strelkaengine.state"
	DEFAULT_AUTO_ENABLED_YARA_RULES                  = "securityonion-yara"
	DEFAULT_COMMUNITY_RULES_IMPORT_ERROR_SECS        = 300
	DEFAULT_FAIL_AFTER_CONSECUTIVE_ERROR_COUNT       = 10
)

var errModuleStopped = fmt.Errorf("strelka module has stopped running")

type IOManager interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, contents []byte, perm fs.FileMode) error
	DeleteFile(path string) error
	ReadDir(path string) ([]os.DirEntry, error)
	MakeRequest(*http.Request) (*http.Response, error)
	ExecCommand(cmd *exec.Cmd) ([]byte, int, time.Duration, error)
}

type StrelkaEngine struct {
	srv                                  *server.Server
	isRunning                            bool
	thread                               *sync.WaitGroup
	interrupt                            chan bool
	interm                               sync.Mutex
	communityRulesImportFrequencySeconds int
	communityRulesImportErrorSeconds     int
	failAfterConsecutiveErrorCount       int
	yaraRulesFolder                      string
	reposFolder                          string
	autoEnabledYaraRules                 []string
	rulesRepos                           []*model.RuleRepo
	compileYaraPythonScriptPath          string
	allowRegex                           *regexp.Regexp
	denyRegex                            *regexp.Regexp
	notify                               bool
	stateFilePath                        string
	IOManager
}

func checkRulesetEnabled(e *StrelkaEngine, det *model.Detection) {
	det.IsEnabled = false

	for _, rule := range e.autoEnabledYaraRules {
		if strings.EqualFold(rule, det.Ruleset) {
			det.IsEnabled = true
			break
		}
	}
}

func NewStrelkaEngine(srv *server.Server) *StrelkaEngine {
	return &StrelkaEngine{
		srv:       srv,
		IOManager: &ResourceManager{},
	}
}

func (e *StrelkaEngine) PrerequisiteModules() []string {
	return nil
}

func (e *StrelkaEngine) Init(config module.ModuleConfig) (err error) {
	e.thread = &sync.WaitGroup{}
	e.interrupt = make(chan bool, 1)

	e.communityRulesImportFrequencySeconds = module.GetIntDefault(config, "communityRulesImportFrequencySeconds", DEFAULT_COMMUNITY_RULES_IMPORT_FREQUENCY_SECONDS)
	e.yaraRulesFolder = module.GetStringDefault(config, "yaraRulesFolder", DEFAULT_YARA_RULES_FOLDER)
	e.reposFolder = module.GetStringDefault(config, "reposFolder", DEFAULT_REPOS_FOLDER)
	e.compileYaraPythonScriptPath = module.GetStringDefault(config, "compileYaraPythonScriptPath", DEFAULT_COMPILE_YARA_PYTHON_SCRIPT_PATH)
	e.autoEnabledYaraRules = module.GetStringArrayDefault(config, "autoEnabledYaraRules", []string{DEFAULT_AUTO_ENABLED_YARA_RULES})
	e.communityRulesImportErrorSeconds = module.GetIntDefault(config, "communityRulesImportErrorSeconds", DEFAULT_COMMUNITY_RULES_IMPORT_ERROR_SECS)
	e.failAfterConsecutiveErrorCount = module.GetIntDefault(config, "failAfterConsecutiveErrorCount", DEFAULT_FAIL_AFTER_CONSECUTIVE_ERROR_COUNT)

	e.rulesRepos, err = model.GetReposDefault(config, "rulesRepos", []*model.RuleRepo{
		{
			Repo:    "https://github.com/Security-Onion-Solutions/securityonion-yara",
			License: "DRL",
		},
	})
	if err != nil {
		return fmt.Errorf("unable to parse Strelka's rulesRepos: %w", err)
	}

	allow := module.GetStringDefault(config, "allowRegex", DEFAULT_ALLOW_REGEX)
	deny := module.GetStringDefault(config, "denyRegex", DEFAULT_DENY_REGEX)

	if allow != "" {
		e.allowRegex, err = regexp.Compile(allow)
		if err != nil {
			return fmt.Errorf("unable to compile Strelka's allowRegex: %w", err)
		}
	}

	if deny != "" {
		var err error
		e.denyRegex, err = regexp.Compile(deny)
		if err != nil {
			return fmt.Errorf("unable to compile Strelka's denyRegex: %w", err)
		}
	}

	e.stateFilePath = module.GetStringDefault(config, "stateFilePath", DEFAULT_STATE_FILE_PATH)

	return nil
}

func (e *StrelkaEngine) Start() error {
	e.srv.DetectionEngines[model.EngineNameStrelka] = e
	e.isRunning = true

	go e.startCommunityRuleImport()

	return nil
}

func (e *StrelkaEngine) Stop() error {
	e.isRunning = false
	e.InterruptSleep(false)
	e.thread.Wait()

	return nil
}

func (e *StrelkaEngine) InterruptSleep(fullUpgrade bool) {
	e.interm.Lock()
	defer e.interm.Unlock()

	e.notify = true

	if len(e.interrupt) == 0 {
		e.interrupt <- fullUpgrade
	}
}

func (e *StrelkaEngine) resetInterrupt() {
	e.interm.Lock()
	defer e.interm.Unlock()

	e.notify = false

	if len(e.interrupt) != 0 {
		<-e.interrupt
	}
}

func (e *StrelkaEngine) IsRunning() bool {
	return e.isRunning
}

func (e *StrelkaEngine) ValidateRule(data string) (string, error) {
	_, err := e.parseYaraRules([]byte(data), false)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (e *StrelkaEngine) ConvertRule(ctx context.Context, detect *model.Detection) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *StrelkaEngine) ExtractDetails(detect *model.Detection) error {
	rules, err := s.parseYaraRules([]byte(detect.Content), false)
	if err != nil {
		return err
	}

	rule := rules[0]

	if rule.Identifier != "" {
		detect.Title = rule.Identifier
	} else {
		detect.Title = "Detection title not yet provided - click here to update this title"
	}

	if rule.Meta.Description != nil {
		detect.Description = *rule.Meta.Description
	}

	detect.Severity = model.SeverityUnknown
	detect.PublicID = rule.GetID()

	return nil
}

func (e *StrelkaEngine) SyncLocalDetections(ctx context.Context, _ []*model.Detection) (errMap map[string]string, err error) {
	return e.syncDetections(ctx)
}

func (e *StrelkaEngine) startCommunityRuleImport() {
	e.thread.Add(1)
	defer func() {
		e.thread.Done()
		e.isRunning = false
	}()

	// |> nil: no import has been completed, it's this way during the first sync
	// so that the timerDur returned by DetermineWaitTime is used. After first sync,
	// the pointer should always have a value
	// |> false: the last sync was not successful, the timer for the next sync should use
	// the shorter communityRulesImportErrorSeconds timer.
	// |> true: the last sync was successful, the timer for the next sync should use
	// the normal communityRulesImportFrequencySeconds timer.
	var lastSyncSuccess *bool

	// publicId of a detection that was written but not read back
	var writeNoRead *string

	templateFound := false

	lastImport, timerDur := detections.DetermineWaitTime(e.IOManager, e.stateFilePath, time.Second*time.Duration(e.communityRulesImportFrequencySeconds))

	for e.isRunning {
		e.resetInterrupt()

		var forceSync bool

		if lastSyncSuccess != nil {
			if *lastSyncSuccess {
				timerDur = time.Second * time.Duration(e.communityRulesImportFrequencySeconds)
			} else {
				timerDur = time.Second * time.Duration(e.communityRulesImportErrorSeconds)
				forceSync = true
			}
		}

		timer := time.NewTimer(timerDur)

		log.WithFields(log.Fields{
			"waitTimeSeconds":   timerDur.Seconds(),
			"forceSync":         forceSync,
			"lastSyncSuccess":   lastSyncSuccess,
			"expectedStartTime": time.Now().Add(timerDur).Format(time.RFC3339),
		}).Info("waiting for next Strelka community rules sync")

		select {
		case <-timer.C:
		case typ := <-e.interrupt:
			forceSync = forceSync || typ
		}

		if !e.isRunning {
			break
		}

		lastSyncSuccess = util.Ptr(false)

		if detections.CheckWriteNoRead(e.srv.Context, e.srv.Detectionstore, writeNoRead) {
			if e.notify {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "error",
				})
			}

			continue
		}

		writeNoRead = nil

		log.WithFields(log.Fields{
			"forceSync": forceSync,
		}).Info("syncing Strelka community rules")

		start := time.Now()

		if !templateFound {
			exists, err := e.srv.Detectionstore.DoesTemplateExist(e.srv.Context, "so-detection")
			if err != nil {
				log.WithError(err).Error("unable to check for detection index template")

				if e.notify {
					e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
						Engine: model.EngineNameStrelka,
						Status: "error",
					})
				}

				continue
			}

			if !exists {
				log.Warn("detection index template does not exist, skipping import")

				if e.notify {
					e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
						Engine: model.EngineNameStrelka,
						Status: "error",
					})
				}

				continue
			}

			templateFound = true
		}

		upToDate := map[string]*model.RuleRepo{}

		allRepos, anythingNew, err := detections.UpdateRepos(&e.isRunning, e.reposFolder, e.rulesRepos)
		if err != nil {
			if strings.Contains(err.Error(), "module stopped") {
				break
			}
		}

		// If no import has been completed, then do a full sync
		if lastImport == nil {
			forceSync = true
		}

		if !anythingNew && !forceSync {
			// no updates, skip
			log.Info("Strelka sync found no changes")

			detections.WriteStateFile(e.IOManager, e.stateFilePath)
			lastSyncSuccess = util.Ptr(true)

			if e.notify {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "success",
				})
			}

			continue
		}

		for k, v := range allRepos {
			if v.WasModified || forceSync {
				upToDate[k] = v.Repo
			}
		}

		communityDetections, err := e.srv.Detectionstore.GetAllCommunitySIDs(e.srv.Context, util.Ptr(model.EngineNameStrelka))
		if err != nil {
			log.WithError(err).Error("Failed to get all community SIDs")

			if e.notify {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "error",
				})
			}

			continue
		}

		et := detections.NewErrorTracker(e.failAfterConsecutiveErrorCount)
		failSync := false

		// parse *.yar files in repos
		for repopath, repo := range upToDate {
			if !e.isRunning {
				return
			}

			baseDir := repopath
			if repo.Folder != nil {
				baseDir = filepath.Join(baseDir, *repo.Folder)
			}

			err = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					log.WithError(err).WithField("path", path).Error("Failed to walk path")
					return nil
				}

				if !e.isRunning {
					return errModuleStopped
				}

				if d.IsDir() {
					return nil
				}

				ext := filepath.Ext(d.Name())
				if strings.ToLower(ext) != ".yar" {
					return nil
				}

				raw, err := e.ReadFile(path)
				if err != nil {
					log.WithError(err).WithField("file", path).Error("failed to read yara rule file")
					return nil
				}

				parsed, err := e.parseYaraRules(raw, true)
				if err != nil {
					log.WithError(err).WithField("file", path).Error("failed to parse yara rule file")
					return nil
				}

				for _, rule := range parsed {
					det := rule.ToDetection(repo.License, filepath.Base(repopath), repo.Community)
					log.WithFields(log.Fields{
						"rule.uuid": det.PublicID,
						"rule.name": det.Title,
					}).Info("Strelka community sync - processing YARA rule")

					comRule, exists := communityDetections[det.PublicID]
					if exists {
						det.IsEnabled = comRule.IsEnabled
						det.Id = comRule.Id
						det.IsEnabled = comRule.IsEnabled
						det.Overrides = comRule.Overrides
						det.CreateTime = comRule.CreateTime
					}

					if exists {
						// pre-existing detection, update it
						log.WithFields(log.Fields{
							"rule.uuid": det.PublicID,
							"rule.name": det.Title,
						}).Info("Updating Yara detection")

						det, err = e.srv.Detectionstore.UpdateDetection(e.srv.Context, det)
						if err != nil && err.Error() == "Object not found" {
							log.WithField("publicId", det.PublicID).Error("unable to read back successful write")

							writeNoRead = util.Ptr(det.PublicID)
							failSync = true

							return err
						}

						eterr := et.AddError(err)
						if eterr != nil {
							return eterr
						}

						if err != nil {
							log.WithError(err).WithField("det", det).Error("Failed to update detection")
							continue
						}
					} else {
						// new detection, create it
						log.WithFields(log.Fields{
							"rule.uuid": det.PublicID,
							"rule.name": det.Title,
						}).Info("Creating new Yara detection")

						checkRulesetEnabled(e, det)

						det, err = e.srv.Detectionstore.CreateDetection(e.srv.Context, det)
						if err != nil && err.Error() == "Object not found" {
							log.WithField("publicId", det.PublicID).Error("unable to read back successful write")

							writeNoRead = util.Ptr(det.PublicID)
							failSync = true

							return err
						}

						eterr := et.AddError(err)
						if eterr != nil {
							failSync = true

							return eterr
						}

						if err != nil {
							log.WithError(err).WithField("det", det).Error("Failed to create detection")
							continue
						}
					}
				}

				return nil
			})
			if err != nil {
				log.WithError(err).WithField("repo", repopath).Error("failed while walking repo")

				if failSync {
					break
				}

				continue
			}
		}

		if failSync {
			continue
		}

		errMap, err := e.syncDetections(e.srv.Context)
		if err != nil {
			if err == errModuleStopped {
				log.Info("incomplete sync of YARA community detections due to module stopping")
				return
			}

			log.WithError(err).Error("unable to sync YARA community detections")

			if e.notify {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "error",
				})
			}

			continue
		}

		detections.WriteStateFile(e.IOManager, e.stateFilePath)
		lastSyncSuccess = util.Ptr(true)

		if e.notify {
			if len(errMap) > 0 {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "partial",
				})
			} else {
				e.srv.Host.Broadcast("detection-sync", "detections", server.SyncStatus{
					Engine: model.EngineNameStrelka,
					Status: "success",
				})
			}
		}

		log.WithFields(log.Fields{
			"errMap": errMap,
			"time":   time.Since(start).Seconds(),
		}).Info("Strelka community rules sync finished")
	}
}

func (e *StrelkaEngine) parseYaraRules(data []byte, filter bool) ([]*YaraRule, error) {
	rules := []*YaraRule{}
	rule := &YaraRule{}

	state := parseStateImportsID

	raw := string(data)
	buffer := bytes.NewBuffer([]byte{})
	last := ' '
	curCommentType := ' '                      // either '/' or '*' if in a comment, ' ' if not in comment
	curHeader := ""                            // meta, strings, condition, or empty if not yet in a section
	curQuotes := ' '                           // either ' or " if in a string, ' ' if not in a string
	fileImports := map[string]*regexp.Regexp{} // every import in the file paired with it's regex

	for i, r := range raw {
		rule.Src += string(r)

		if r == '\r' {
			continue
		}

		if (curCommentType == '*' && last == '*' && r == '/') ||
			(curCommentType == '/' && r == '\n') {
			curCommentType = ' '

			if last == '*' {
				last = r
				continue
			}
		}

		if last == '/' && curQuotes == ' ' && curCommentType == ' ' {
			if r == '/' {
				curCommentType = '/'
				if buffer.Len() != 0 {
					buffer.Truncate(buffer.Len() - 1)
				}
			} else if r == '*' {
				curCommentType = '*'
				if buffer.Len() != 0 {
					buffer.Truncate(buffer.Len() - 1)
				}
			}
		}

		if curCommentType != ' ' {
			// in a comment, skip everything
			last = r
			continue
		}

	reevaluateState:
		switch state {
		case parseStateImportsID:
			switch r {
			case '\n':
				// is this an import?
				buf := buffer.String() // expected: `import "foo"`
				if strings.HasPrefix(buf, "import ") {
					buf = strings.TrimSpace(strings.TrimPrefix(buf, "import "))
					buf = strings.Trim(buf, `"`)

					rule.Imports = append(rule.Imports, buf)
					fileImports[buf] = buildImportChecker(buf)

					buffer.Reset()
				}
			case '{':
				buf := strings.TrimSpace(buffer.String()) // expected: `rule foo {` or `private rule foo\n{`

				if strings.HasPrefix(buf, "private ") {
					rule.IsPrivate = true
					buf = strings.TrimSpace(strings.TrimPrefix(buf, "private"))
				}

				buf = strings.TrimSpace(strings.TrimPrefix(buf, "rule"))

				if strings.Contains(buf, ":") {
					// gets rid of inheritance?
					// rule This : That {...} becomes "This"
					parts := strings.SplitN(buf, ":", 2)
					buf = strings.TrimSpace(parts[0])
				}

				if buf != "" {
					rule.Identifier = buf
				} else {
					return nil, fmt.Errorf("expected rule identifier at %d", i)
				}

				buffer.Reset()

				state = parseStateWatchForHeader
			default:
				buffer.WriteRune(r)
			}
		case parseStateWatchForHeader:
			buf := strings.TrimSpace(buffer.String())
			if r == '\n' && len(buf) != 0 && buf[len(buf)-1] == ':' {
				curHeader = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(buf, ":")))
				buffer.Reset()

				if curHeader != "meta" &&
					curHeader != "strings" &&
					curHeader != "condition" {
					return nil, fmt.Errorf("unexpected header at %d: %s", i, curHeader)
				}

				state = parseStateInSection
			} else {
				buffer.WriteRune(r)
			}
		case parseStateInSection:
			if r == '\n' {
				buf := strings.TrimSpace(buffer.String())
				if len(buf) != 0 && buf[len(buf)-1] == ':' && !strings.HasPrefix(buf, "for ") {
					// found a header, new section
					state = parseStateWatchForHeader
					goto reevaluateState
				} else {
					if buf != "" {
						switch curHeader {
						case "meta":
							parts := strings.SplitN(buf, "=", 2)
							if len(parts) != 2 {
								return nil, fmt.Errorf("invalid meta line at %d: %s", i, buf)
							}

							key := strings.TrimSpace(parts[0])
							value := strings.TrimSpace(parts[1])

							rule.Meta.Set(key, value)
						case "strings":
							rule.Strings = append(rule.Strings, buf)
						case "condition":
							rule.Condition = strings.TrimSpace(rule.Condition + " " + buf)
						}
					}

					buffer.Reset()
				}
			} else if r == '}' && len(strings.TrimSpace(buffer.String())) == 0 && curQuotes != '}' {
				// end of rule
				rule.Src = strings.TrimSpace(rule.Src)
				keep := true

				if filter && e.denyRegex != nil && e.denyRegex.MatchString(rule.Src) {
					log.WithField("identifier", rule.Identifier).Debug("content matched Strelka's denyRegex")
					keep = false
				}

				if filter && e.allowRegex != nil && !e.allowRegex.MatchString(rule.Src) {
					log.WithField("identifier", rule.Identifier).Debug("content didn't match Strelka's allowRegex")
					keep = false
				}

				if keep {
					addMissingImports(rule, fileImports)
					rules = append(rules, rule)
				}

				buffer.Reset()

				state = parseStateImportsID
				curHeader = ""
				curQuotes = ' '
				rule = &YaraRule{}
			} else {
				buffer.WriteRune(r)
				if (r == '\'' || r == '"' || r == '{') && last != '\\' && curQuotes == ' ' {
					// starting a string
					if r == '{' {
						curQuotes = '}'
					} else {
						curQuotes = r
					}
				} else if curQuotes != ' ' && r == curQuotes && last != '\\' {
					// ending a string
					curQuotes = ' '
				}
			}
		}

		if r == '\\' && last == '\\' && curQuotes != ' ' {
			// this is an escaped slash in the middle of a string,
			// so we need to remove the previous slash so it's not
			// mistaken for an escape character in case this is the
			// last character in the string
			last = ' '
		} else {
			last = r
		}
	}

	if state != parseStateImportsID || len(strings.TrimSpace(buffer.String())) != 0 {
		return nil, errors.New("unexpected end of rule")
	}

	return rules, nil
}

func addMissingImports(rule *YaraRule, imports map[string]*regexp.Regexp) {
	newImports := []string{}

	for pkg, finder := range imports {
		hasImport := slices.Contains(rule.Imports, pkg)
		if !hasImport {
			usesImport := finder.MatchString(rule.Src)
			if usesImport {
				rule.Imports = append(rule.Imports, pkg)
				newImports = append(newImports, fmt.Sprintf("import \"%s\"", pkg))
			}
		}
	}

	if len(newImports) != 0 {
		rule.Src = fmt.Sprintf("%s\n\n%s", strings.Join(newImports, "\n"), rule.Src)
	}
}

// buildImportChecker builds a regex looking for the use of a package in an use case
// other than the import statement.
func buildImportChecker(pkg string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`[^"]\b%s\b[^"]`, pkg))
}

func (e *StrelkaEngine) syncDetections(ctx context.Context) (errMap map[string]string, err error) {
	results, err := e.srv.Detectionstore.Query(ctx, `so_detection.engine:strelka AND so_detection.isEnabled:true AND _index:"*:so-detection"`, -1)
	if err != nil {
		return nil, err
	}

	enabledDetections := map[string]*model.Detection{}
	for _, det := range results {
		if !e.isRunning {
			return nil, errModuleStopped
		}

		d := det.(*model.Detection)
		_, exists := enabledDetections[d.PublicID]
		if exists {
			return nil, fmt.Errorf("duplicate detection with public ID %s", d.PublicID)
		}
		enabledDetections[d.PublicID] = d
	}

	// Clear existing .yar files in the directory
	files, err := e.ReadDir(e.yaraRulesFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	deleteByExt := map[string]struct{}{
		".yar":      {},
		".compiled": {},
	}

	for _, file := range files {
		_, ok := deleteByExt[strings.ToLower(filepath.Ext(file.Name()))]
		if ok {
			err := e.DeleteFile(filepath.Join(e.yaraRulesFolder, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to delete existing .yar file %s: %v", file.Name(), err)
			}
		}
	}

	// Process and write new .yar files
	for publicId, det := range enabledDetections {
		name := sanitize.Name(publicId)
		filename := filepath.Join(e.yaraRulesFolder, fmt.Sprintf("%s.yar", name))

		err := e.WriteFile(filename, []byte(det.Content), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write file for detection %s: %v", publicId, err)
		}
	}

	// compile yara rules, call even if no yara rules
	cmd := exec.CommandContext(ctx, "python3", e.compileYaraPythonScriptPath, e.yaraRulesFolder)

	raw, code, dur, err := e.ExecCommand(cmd)

	log.WithFields(log.Fields{
		"command":  cmd.String(),
		"output":   string(raw),
		"code":     code,
		"execTime": dur.Seconds(),
		"error":    err,
	}).Info("yara compilation results")

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (e *StrelkaEngine) DuplicateDetection(ctx context.Context, detection *model.Detection) (*model.Detection, error) {
	rules, err := e.parseYaraRules([]byte(detection.Content), false)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, fmt.Errorf("unable to parse rule")
	}

	rule := rules[0]

	rule.Identifier += "_copy"

	det := rule.ToDetection(model.LicenseUnknown, detections.RULESET_CUSTOM, false)

	err = e.ExtractDetails(det)
	if err != nil {
		return nil, err
	}

	userID := ctx.Value(web.ContextKeyRequestorId).(string)
	user, err := e.srv.Userstore.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	author := strings.Join([]string{user.FirstName, user.LastName}, " ")
	if author == "" {
		author = user.Email
	}

	det.Author = author

	return det, nil
}

// go install go.uber.org/mock/mockgen@latest
//go:generate mockgen -destination mock/mock_iomanager.go -package mock . IOManager

type ResourceManager struct{}

func (_ *ResourceManager) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (_ *ResourceManager) WriteFile(path string, contents []byte, perm fs.FileMode) error {
	return os.WriteFile(path, contents, perm)
}

func (_ *ResourceManager) DeleteFile(path string) error {
	return os.Remove(path)
}

func (_ *ResourceManager) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (_ *ResourceManager) MakeRequest(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

func (_ *ResourceManager) ExecCommand(cmd *exec.Cmd) (output []byte, exitCode int, runtime time.Duration, err error) {
	start := time.Now()
	output, err = cmd.CombinedOutput()
	runtime = time.Since(start)

	exitCode = cmd.ProcessState.ExitCode()

	return output, exitCode, runtime, err
}
