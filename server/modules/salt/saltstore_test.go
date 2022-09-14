// Copyright Jason Ertel (github.com/jertel).
// Copyright Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package salt

import (
	"context"
	"errors"
	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/security-onion-solutions/securityonion-soc/server"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

const TMP_SALTSTACK_PATH = "/tmp/gotest-soc-saltstore"
const TMP_REQUEST_PIPE = "/tmp/gotest-soc-salt-req.pipe"
const TEST_SETTINGS_COUNT = 20

func Cleanup() {
	exec.Command("rm", "-fr", TMP_SALTSTACK_PATH).Run()
	exec.Command("rm", "-fr", TMP_REQUEST_PIPE).Run()
}

func NewTestSalt() *Saltstore {
	Cleanup()
	exec.Command("mkdir", "-p", TMP_SALTSTACK_PATH).Run()
	exec.Command("cp", "-fr", "./test_resources/saltstack", TMP_SALTSTACK_PATH).Run()

	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)
	salt.Init(123, TMP_SALTSTACK_PATH+"/saltstack", "test_pipe_req", "test_pipe_resp", false)
	return salt
}

func NewTestSaltPipe(respPipe string) *Saltstore {
	Cleanup()
	exec.Command("touch", TMP_REQUEST_PIPE).Run()

	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)
	salt.Init(123, TMP_SALTSTACK_PATH+"/saltstack", TMP_REQUEST_PIPE, "./test_resources/pipe/"+respPipe, false)
	return salt
}

func ReadReqPipe() string {
	contents, _ := os.ReadFile(TMP_REQUEST_PIPE)
	return string(contents)
}

func TestSaltstoreInit(tester *testing.T) {
	salt := NewSaltstore(nil)
	salt.Init(123, "saltstack/path", "salt/control/pipe.req", "salt/control/pipe.resp", false)
	assert.Equal(tester, 123, salt.timeoutMs)
	assert.Equal(tester, "saltstack/path", salt.saltstackDir)
	assert.Equal(tester, "salt/control/pipe.req", salt.saltPipeReq)
	assert.Equal(tester, "salt/control/pipe.resp", salt.saltPipeResp)
}

func TestGetMembersFromJson(tester *testing.T) {
	json := `
		{
	    "minions": {
        "minion_id": "fingerprint"
      }
		}
	`

	// Error supplied
	members, err := getMembersFromJson(errors.New("something bad"), []byte(json))
	assert.Error(tester, err)
	assert.Nil(tester, members)

	// Parse error
	members, err = getMembersFromJson(nil, []byte("{ds"))
	assert.EqualError(tester, err, "invalid character 'd' looking for beginning of object key string")
	assert.Nil(tester, members)

	// Good parse
	members, err = getMembersFromJson(nil, []byte(json))
	assert.NoError(tester, err)
	assert.Len(tester, members, 1)
	assert.Equal(tester, "minion_id", members[0].Id)
	assert.Equal(tester, "id", members[0].Role)
	assert.Equal(tester, "fingerprint", members[0].Fingerprint)
}

func TestGetMembers_BadPipePath(tester *testing.T) {
	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)
	_, err := salt.GetMembers(context.Background())
	assert.EqualError(tester, err, "open : no such file or directory")
}

func TestGetMembersUnauthorized(tester *testing.T) {
	srv := server.NewFakeUnauthorizedServer()
	salt := NewSaltstore(srv)
	_, err := salt.GetMembers(context.Background())
	assert.ErrorContains(tester, err, "Subject 'fake-subject' is not authorized to perform operation 'read' on target 'grid'")
}

func TestGetMembers(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSaltPipe("list_minions.resp")
	members, err := salt.GetMembers(context.Background())
	assert.NoError(tester, err)
	assert.Equal(tester, 15, len(members))

	request := ReadReqPipe()
	assert.Equal(tester, "list-minions", request)
}

func TestGetMembers_Failure(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSaltPipe("false.resp")
	members, err := salt.GetMembers(context.Background())
	assert.EqualError(tester, err, "ERROR_SALT_MANAGE_MEMBER")
	assert.Equal(tester, 0, len(members))

	request := ReadReqPipe()
	assert.Equal(tester, "list-minions", request)
}

func TestManageMemberUnauthorized(tester *testing.T) {
	srv := server.NewFakeUnauthorizedServer()
	salt := NewSaltstore(srv)

	for _, op := range []string{"add", "reject", "delete"} {
		err := salt.ManageMember(context.Background(), op, "foo")
		assert.ErrorContains(tester, err, "Subject 'fake-subject' is not authorized to perform operation 'write' on target 'grid'")
	}
}

func TestManageMember_BadPipePath(tester *testing.T) {
	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)

	for _, op := range []string{"add", "reject", "delete"} {
		err := salt.ManageMember(context.Background(), op, "foo")
		assert.EqualError(tester, err, "open : no such file or directory")
	}
}

func TestManageMember(tester *testing.T) {
	for _, op := range []string{"add", "reject", "delete"} {
		defer Cleanup()
		salt := NewTestSaltPipe("true.resp")
		err := salt.ManageMember(context.Background(), op, "foo")
		assert.NoError(tester, err)

		request := ReadReqPipe()
		assert.Equal(tester, "manage-minion "+op+" foo", request)
	}
}

func TestManageMember_Failure(tester *testing.T) {
	for _, op := range []string{"add", "reject", "delete"} {
		defer Cleanup()
		salt := NewTestSaltPipe("false.resp")
		err := salt.ManageMember(context.Background(), op, "foo")
		assert.EqualError(tester, err, "ERROR_SALT_MANAGE_MEMBER")

		request := ReadReqPipe()
		assert.Equal(tester, "manage-minion "+op+" foo", request)
	}
}

func TestGetSettings_BadSaltstackPath(tester *testing.T) {
	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)
	_, err := salt.GetSettings(context.Background())
	assert.EqualError(tester, err, "lstat /default: no such file or directory")
}

func TestGetSettings(tester *testing.T) {
	defer Cleanup()

	salt := NewTestSalt()
	settings, err := salt.GetSettings(context.Background())
	assert.NoError(tester, err)

	count := 0
	assert.Equal(tester, "myapp.bar", settings[count].Id)
	assert.Equal(tester, "minion-override", settings[count].Value)
	assert.Equal(tester, "normal_import", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.bool", settings[count].Id)
	assert.Equal(tester, "true", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_bool", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_float", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_int", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_list_str", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_map_str", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.empty_lists.list_str", settings[count].Id)
	assert.Equal(tester, "", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.float", settings[count].Id)
	assert.Equal(tester, "3.5", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.foo", settings[count].Id)
	assert.Equal(tester, "minion-born", settings[count].Value)
	assert.Equal(tester, "normal_import", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.foo__txt", settings[count].Id)
	assert.Equal(tester, "old", settings[count].Value)
	assert.Equal(tester, true, settings[count].File)
	count++

	assert.Equal(tester, "myapp.int", settings[count].Id)
	assert.Equal(tester, "123", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	assert.Equal(tester, "([0-9]+){3}", settings[count].Regex)
	assert.Equal(tester, "Invalid!", settings[count].RegexFailureMessage)
	assert.Equal(tester, "test desc", settings[count].Description)
	assert.Equal(tester, true, settings[count].Global)
	assert.Equal(tester, true, settings[count].Readonly)
	count++

	assert.Equal(tester, "myapp.lists.list_bool", settings[count].Id)
	assert.Equal(tester, "true\nfalse", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.lists.list_float", settings[count].Id)
	assert.Equal(tester, "1.24\n2.2", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.lists.list_int", settings[count].Id)
	assert.Equal(tester, "3\n24", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.lists.list_list_str", settings[count].Id)
	assert.Equal(tester, "[\"item1\",\"item2\"]\n[\"item3\",\"item4\"]", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.lists.list_map_str", settings[count].Id)
	assert.Equal(tester, "{\"key1\":\"value1\",\"key2\":\"value2\"}\n{\"key1\":\"value3\",\"key2\":\"value4\"}", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.lists.list_str", settings[count].Id)
	assert.Equal(tester, "foo\nbar", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.str", settings[count].Id)
	assert.Equal(tester, "my_str", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, "myapp.advanced", settings[count].Id)
	assert.Equal(tester, "myapp:\n  global: advanced\n", settings[count].Value)
	assert.Equal(tester, "", settings[count].NodeId)
	count++

	assert.Equal(tester, count, len(settings))
}

func TestUpdateSetting_MissingSettingFile(tester *testing.T) {
	srv := server.NewFakeAuthorizedServer(nil)
	salt := NewSaltstore(srv)
	setting := model.NewSetting("some.setting")
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, "open /local/pillar/some/soc_some.sls: no such file or directory")
}

func findSetting(settings []*model.Setting, id string, nodeId string) *model.Setting {
	for _, setting := range settings {
		if setting.Id == id && setting.NodeId == nodeId {
			return setting
		}
	}
	return nil
}

func TestUpdateSetting_AddGlobal(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Add new setting
	setting := model.NewSetting("myapp.setting")
	setting.Value = "new setting"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure there's an additional setting listed
	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT+1, len(settings))

	new_setting := findSetting(settings, "myapp.setting", "")
	assert.Equal(tester, "new setting", new_setting.Value)
	assert.Equal(tester, "", new_setting.NodeId)
	assert.Equal(tester, false, new_setting.Global)
	assert.Equal(tester, false, new_setting.Node)
}

func TestUpdateSetting_AddToNode(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Add new setting
	setting := model.NewSetting("myapp.setting")
	setting.Value = "new setting"
	setting.NodeId = "normal_import"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure there's an additional setting listed
	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT+1, len(settings))

	new_setting := findSetting(settings, "myapp.setting", "normal_import")
	assert.Equal(tester, "new setting", new_setting.Value)
	assert.Equal(tester, "normal_import", new_setting.NodeId)
	assert.Equal(tester, false, new_setting.Global)
	assert.Equal(tester, false, new_setting.Node)
}

// This isn't currently supported via the UI, but the API supports it.
func TestUpdateSetting_DeleteGlobal(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Delete setting
	setting := model.NewSetting("myapp.str")
	setting.NodeId = ""
	err := salt.UpdateSetting(context.Background(), setting, true)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT-1, len(settings))
	delete_setting := findSetting(settings, "myapp.str", "")
	assert.Nil(tester, delete_setting)
}

func TestUpdateSetting_DeleteFromNode(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Delete setting
	setting := model.NewSetting("myapp.foo")
	setting.NodeId = "normal_import"
	err := salt.UpdateSetting(context.Background(), setting, true)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT-1, len(settings))
	delete_setting := findSetting(settings, "myapp.foo", "normal_import")
	assert.Nil(tester, delete_setting)
}

func TestUpdateSetting_DeleteAdvanced(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Delete setting
	setting := model.NewSetting("myapp.advanced")
	err := salt.UpdateSetting(context.Background(), setting, true)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT, len(settings))
	deleted_setting := findSetting(settings, "myapp.advanced", "")
	assert.Equal(tester, "", deleted_setting.Value)
}

func TestUpdateSetting_UpdateGlobal(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.str")
	setting.NodeId = ""
	setting.Value = "new value"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	assert.Equal(tester, TEST_SETTINGS_COUNT, len(settings))
	updated_setting := findSetting(settings, "myapp.str", "")
	assert.Equal(tester, "new value", updated_setting.Value)
	assert.Equal(tester, "", updated_setting.NodeId)
}

func TestUpdateSetting_UpdateForNode(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.foo")
	setting.NodeId = "normal_import"
	setting.Value = "new value"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.foo", "normal_import")
	assert.Equal(tester, "new value", updated_setting.Value)
	assert.Equal(tester, "normal_import", updated_setting.NodeId)
}

func TestUpdateSetting_UpdateAdvanced(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.advanced")
	setting.Value = "something: new"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.advanced", "")
	assert.Equal(tester, "something: new", updated_setting.Value)
	assert.Equal(tester, "", updated_setting.NodeId)
}

func TestUpdateSetting_UpdateFile(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.foo__txt")
	setting.File = true
	setting.Value = "something"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.foo__txt", "")
	assert.Equal(tester, "anything", updated_setting.Default)
	assert.Equal(tester, "something", updated_setting.Value)
}

func TestUpdateSetting_UpdateAdvancedFailToParse(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.advanced")
	setting.Value = "new advanced"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, "ERROR_MALFORMED_YAML -> yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `new adv...`")
}

///// INT TYPE

func TestUpdateSetting_AlignIntType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.int")
	setting.Value = "44"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.int", "")
	assert.Equal(tester, "44", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignIntType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.int")
	setting.Value = "not an int"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, "strconv.ParseInt: parsing \"not an int\": invalid syntax")
}

func TestUpdateSetting_AlignIntListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_int")
	setting.Value = "44\n2\n1"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_int", "")
	assert.Equal(tester, "44\n2\n1", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignIntListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_int")
	setting.Value = "1\n2\ninvalid"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseInt: parsing "invalid": invalid syntax`)
}

func TestUpdateSetting_AlignEmptyListIntType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Prime the empty list setting with int
	setting := model.NewSetting("myapp.empty_lists.list_int")
	setting.Value = "123\n456"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure we can update it with more ints
	setting = model.NewSetting("myapp.empty_lists.list_int")
	setting.Value = "123\n456\n23"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.empty_lists.list_int", "")
	assert.Equal(tester, "123\n456\n23", updated_setting.Value)

	// Now try to put the wrong type in it
	setting = model.NewSetting("myapp.empty_lists.list_int")
	setting.Value = "cannot set string on int list"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseInt: parsing "cannot set string on int list": invalid syntax`)
}

///// FLOAT TYPE

func TestUpdateSetting_AlignFloatType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.float")
	setting.Value = "44.2"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.float", "")
	assert.Equal(tester, "44.2", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignFloatType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.float")
	setting.Value = "not a float"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, "strconv.ParseFloat: parsing \"not a float\": invalid syntax")
}

func TestUpdateSetting_AlignFloatListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_float")
	setting.Value = "44.3\n2.1\n1.2"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_float", "")
	assert.Equal(tester, "44.3\n2.1\n1.2", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignFloatListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_float")
	setting.Value = "1.2\nnope"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseFloat: parsing "nope": invalid syntax`)
}

func TestUpdateSetting_AlignEmptyListFloatType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Prime the empty list setting with float
	setting := model.NewSetting("myapp.empty_lists.list_float")
	setting.Value = "1.23\n4.56"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure we can update it with more floats
	setting = model.NewSetting("myapp.empty_lists.list_float")
	setting.Value = "1.23\n4.56\n2.3"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.empty_lists.list_float", "")
	assert.Equal(tester, "1.23\n4.56\n2.3", updated_setting.Value)

	// Now try to put the wrong type in it
	setting = model.NewSetting("myapp.empty_lists.list_float")
	setting.Value = "cannot set string on float list"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseFloat: parsing "cannot set string on float list": invalid syntax`)
}

// BOOL TYPE

func TestUpdateSetting_AlignBoolType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.bool")
	setting.Value = "false"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.bool", "")
	assert.Equal(tester, "false", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignBoolType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.bool")
	setting.Value = "not a bool"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, "strconv.ParseBool: parsing \"not a bool\": invalid syntax")
}

func TestUpdateSetting_AlignBoolListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_bool")
	setting.Value = "true\nfalse\ntrue"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_bool", "")
	assert.Equal(tester, "true\nfalse\ntrue", updated_setting.Value)
}

func TestUpdateSetting_FailToAlignBoolListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_bool")
	setting.Value = "true\nfalse\nhi"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseBool: parsing "hi": invalid syntax`)
}

func TestUpdateSetting_AlignEmptyListBoolType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Prime the empty list setting with bools
	setting := model.NewSetting("myapp.empty_lists.list_bool")
	setting.Value = "true\nfalse"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure we can update it with more bools
	setting = model.NewSetting("myapp.empty_lists.list_bool")
	setting.Value = "true\ntrue\nfalse"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.empty_lists.list_bool", "")
	assert.Equal(tester, "true\ntrue\nfalse", updated_setting.Value)

	// Now try to put the wrong type in it
	setting = model.NewSetting("myapp.empty_lists.list_bool")
	setting.Value = "cannot set string on bool list"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `strconv.ParseBool: parsing "cannot set string on bool list": invalid syntax`)
}

// List of Lists TYPE

func TestUpdateSetting_AlignListListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	expected := "[\"item1\",\"item2\"]\n[\"item3\",\"item3\"]\n[\"item5\",\"item6\"]"
	setting := model.NewSetting("myapp.lists.list_list_str")
	setting.Value = expected
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_list_str", "")
	assert.Equal(tester, expected, updated_setting.Value)
}

func TestUpdateSetting_FailToAlignListListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting (can't change list of lists to list of bools)
	setting := model.NewSetting("myapp.lists.list_list_str")
	setting.Value = "true\nfalse"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `json: cannot unmarshal bool into Go value of type []interface {}`)
}

func TestUpdateSetting_AlignEmptyListListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Prime the empty list setting with list of strings
	setting := model.NewSetting("myapp.empty_lists.list_list_str")
	setting.Value = "[\"item1\",\"item2\"]\n[\"item3\",\"item3\"]"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure we can update it with more bools
	expected := "[\"item1\",\"item2\"]\n[\"item3\",\"item3\"]\n[\"item5\",\"item6\"]"
	setting = model.NewSetting("myapp.empty_lists.list_list_str")
	setting.Value = expected
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.empty_lists.list_list_str", "")
	assert.Equal(tester, expected, updated_setting.Value)

	// Now try to put the wrong type in it
	setting = model.NewSetting("myapp.empty_lists.list_list_str")
	setting.Value = "cannot set list of strings\non list of lists"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `invalid character 'c' looking for beginning of value`)
}

// List of Maps TYPE

func TestUpdateSetting_AlignMapListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	expected := "{\"key1\":\"value1\",\"key2\":\"value2\"}\n{\"key1\":\"value3\",\"key2\":\"value4\"}\n{\"key1\":\"value5\",\"key2\":\"value6\"}"
	setting := model.NewSetting("myapp.lists.list_map_str")
	setting.Value = expected
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_map_str", "")
	assert.Equal(tester, expected, updated_setting.Value)
}

func TestUpdateSetting_FailToAlignMapListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting (can't change list of maps to list of bools)
	setting := model.NewSetting("myapp.lists.list_map_str")
	setting.Value = "true\nfalse"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `json: cannot unmarshal bool into Go value of type map[string]interface {}`)
}

func TestUpdateSetting_AlignEmptyListMapType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Prime the empty list setting with list of maps
	setting := model.NewSetting("myapp.empty_lists.list_map_str")
	setting.Value = "{\"key1\":\"value1\",\"key2\":\"value2\"}"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	// Ensure we can update it with more bools
	expected := "{\"key1\":\"value1\",\"key2\":\"value2\"}\n{\"key1\":\"value3\",\"key2\":\"value4\"}"
	setting = model.NewSetting("myapp.empty_lists.list_map_str")
	setting.Value = expected
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.empty_lists.list_map_str", "")
	assert.Equal(tester, expected, updated_setting.Value)

	// Now try to put the wrong type in it
	setting = model.NewSetting("myapp.empty_lists.list_map_str")
	setting.Value = "cannot set list of strings\non list of maps"
	err = salt.UpdateSetting(context.Background(), setting, false)
	assert.EqualError(tester, err, `invalid character 'c' looking for beginning of value`)
}

// STRING TYPE

func TestUpdateSetting_AlignNonStringType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.str")
	setting.Value = "123"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.str", "")
	assert.Equal(tester, "123", updated_setting.Value)
}

func TestUpdateSetting_AlignNonStringListType(tester *testing.T) {
	defer Cleanup()
	salt := NewTestSalt()

	// Update setting
	setting := model.NewSetting("myapp.lists.list_str")
	setting.Value = "123\n456"
	err := salt.UpdateSetting(context.Background(), setting, false)
	assert.NoError(tester, err)

	settings, get_err := salt.GetSettings(context.Background())
	assert.NoError(tester, get_err)
	updated_setting := findSetting(settings, "myapp.lists.list_str", "")
	assert.Equal(tester, "123\n456", updated_setting.Value)
}