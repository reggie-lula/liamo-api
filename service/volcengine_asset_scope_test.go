package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedVolcEngineAssetChannel(t *testing.T, id int, name string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     id,
		Type:   constant.ChannelTypeVolcEngine,
		Key:    "access-key|secret-key",
		Status: common.ChannelStatusEnabled,
		Name:   name,
		Group:  "default",
	}).Error)
}

func TestAIGCAssetRequestAcceptsProjectName(t *testing.T) {
	var req dto.CreateAIGCAssetReq
	require.NoError(t, common.Unmarshal([]byte(`{
		"group_id":"group-insta",
		"url":"https://example.com/image.png",
		"asset_type":"Image",
		"project_name":"INSTA-TV"
	}`), &req))
	assert.Equal(t, "INSTA-TV", req.ProjectName)
}

func TestLivenessAssetRequestsAcceptProjectName(t *testing.T) {
	var sessionReq dto.CreateVisualValidateSessionReq
	require.NoError(t, common.Unmarshal([]byte(`{
		"callback_url":"https://example.com/callback",
		"project_name":"INSTA-TV"
	}`), &sessionReq))
	assert.Equal(t, "INSTA-TV", sessionReq.ProjectName)

	var assetReq dto.CreateAssetReq
	require.NoError(t, common.Unmarshal([]byte(`{
		"group_id":"group-insta",
		"url":"https://example.com/image.png",
		"asset_type":"Image",
		"project_name":"INSTA-TV"
	}`), &assetReq))
	assert.Equal(t, "INSTA-TV", assetReq.ProjectName)
}

func TestResolveAIGCChannelForGroupUsesMatchingMapping(t *testing.T) {
	truncate(t)
	seedVolcEngineAssetChannel(t, 9101, "default-project-channel")
	seedVolcEngineAssetChannel(t, 9102, "insta-project-channel")

	require.NoError(t, model.DB.Create(&model.AIGCAssetGroupMapping{
		UserId:          9201,
		GroupId:         "group-default",
		ChannelId:       9101,
		VolcProjectName: "default",
	}).Error)
	require.NoError(t, model.DB.Create(&model.AIGCAssetGroupMapping{
		UserId:          9201,
		GroupId:         "group-insta",
		ChannelId:       9102,
		VolcProjectName: "INSTA-TV",
	}).Error)

	channel, projectName, err := resolveAIGCChannelForGroup(9201, "group-insta", "INSTA-TV")
	require.NoError(t, err)
	assert.Equal(t, 9102, channel.Id)
	assert.Equal(t, "INSTA-TV", projectName)

	_, _, err = resolveAIGCChannelForGroup(9201, "group-insta", "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestResolveVolcAssetGroupScopeRequiresProjectForMultipleProjects(t *testing.T) {
	truncate(t)
	seedVolcEngineAssetChannel(t, 9301, "shared-channel")

	bindings := []volcAssetGroupBinding{
		{GroupID: "group-default", ChannelID: 9301, ProjectName: "default"},
		{GroupID: "group-insta", ChannelID: 9301, ProjectName: "INSTA-TV"},
	}

	_, _, _, err := resolveVolcAssetGroupScope(bindings, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project_name is required")

	channel, projectName, groupIDs, err := resolveVolcAssetGroupScope(bindings, "INSTA-TV", nil)
	require.NoError(t, err)
	assert.Equal(t, 9301, channel.Id)
	assert.Equal(t, "INSTA-TV", projectName)
	assert.Equal(t, []string{"group-insta"}, groupIDs)
}

func TestResolveVolcEngineChannelForNewProjectKeepsUserChannel(t *testing.T) {
	truncate(t)
	seedVolcEngineAssetChannel(t, 9351, "user-channel")

	channel, projectName, err := resolveVolcEngineChannelForProject(
		[]volcAssetGroupBinding{
			{GroupID: "group-default", ChannelID: 9351, ProjectName: "default"},
		},
		"INSTA-TV",
	)
	require.NoError(t, err)
	assert.Equal(t, 9351, channel.Id)
	assert.Equal(t, "INSTA-TV", projectName)
}

func TestBuildCreateAssetRequestIncludesProjectName(t *testing.T) {
	reqBody := buildCreateAssetRequest(
		"group-insta",
		"https://example.com/image.png",
		"Image",
		"portrait",
		"INSTA-TV",
	)

	assert.Equal(t, "group-insta", reqBody["GroupId"])
	assert.Equal(t, "INSTA-TV", reqBody["ProjectName"])
	assert.Equal(t, "portrait", reqBody["Name"])
}

func TestCreateUserAssetMappingKeepsMultipleProjects(t *testing.T) {
	truncate(t)
	seedVolcEngineAssetChannel(t, 9401, "shared-channel")

	_, err := CreateUserAssetMapping(9402, "group-default", 9401, "default")
	require.NoError(t, err)
	_, err = CreateUserAssetMapping(9402, "group-insta", 9401, "INSTA-TV")
	require.NoError(t, err)

	mappings, err := model.GetUserAssetGroupMappings(9402)
	require.NoError(t, err)
	require.Len(t, mappings, 2)
	assert.Equal(t, "default", mappings[0].VolcProjectName)
	assert.Equal(t, "INSTA-TV", mappings[1].VolcProjectName)
}
