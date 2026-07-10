package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"github.com/volcengine/volcengine-go-sdk/volcengine/universal"
)

const arkBaseURL = "https://ark.cn-beijing.volces.com"
const arkAPIVersion = "2024-01-01"

type dtoChannelSetting struct {
	Proxy string `json:"proxy"`
}

// CallArkAPI 调用火山引擎 Ark API（使用官方 SDK）
func CallArkAPI(ctx context.Context, channel *model.Channel, action string, reqBody map[string]any) (map[string]any, error) {
	accessKey, secretKey, err := ParseVolcengineAssetAuth(channel.Key)
	if err != nil {
		return nil, fmt.Errorf("parse volcengine auth failed: %w", err)
	}

	config := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
		WithRegion("cn-beijing")

	// 支持 HTTP 代理
	if channel.Setting != nil {
		var setting dtoChannelSetting
		if err := common.UnmarshalJsonStr(*channel.Setting, &setting); err == nil && setting.Proxy != "" {
			proxyURL, parseErr := url.Parse(setting.Proxy)
			if parseErr == nil {
				config.HTTPClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
			}
		}
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("create session failed: %w", err)
	}

	client := universal.New(sess)
	logger.LogDebug(
		ctx,
		"volcengine asset request action=%s channel_id=%d project_name=%v group_id=%v resource_id=%v",
		action,
		channel.Id,
		reqBody["ProjectName"],
		reqBody["GroupId"],
		reqBody["Id"],
	)

	resp, err := client.DoCall(
		universal.RequestUniversal{
			ServiceName: "ark",
			Action:      action,
			Version:     arkAPIVersion,
			HttpMethod:  universal.POST,
			ContentType: universal.ApplicationJSON,
		},
		&reqBody,
	)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("response is nil")
	}

	respBytes, _ := common.Marshal(resp)
	var result map[string]any
	_ = common.Unmarshal(respBytes, &result)
	return result, nil
}

// ParseVolcengineAssetAuth 从 channel.Key 解析 AK/SK（格式: access_key|secret_key）
func ParseVolcengineAssetAuth(key string) (accessKey, secretKey string, err error) {
	if key == "" {
		return "", "", errors.New("volcengine channel key is empty")
	}
	parts := strings.Split(key, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid volcengine key format, expected: access_key|secret_key")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func getUserAssetBindings(userId int) ([]volcAssetGroupBinding, error) {
	mappings, err := model.GetUserAssetGroupMappings(userId)
	if err != nil {
		return nil, fmt.Errorf("query user asset group mappings failed: %w", err)
	}
	bindings := make([]volcAssetGroupBinding, 0, len(mappings))
	for _, mapping := range mappings {
		bindings = append(bindings, volcAssetGroupBinding{
			GroupID:     mapping.GroupId,
			ChannelID:   mapping.ChannelId,
			ProjectName: mapping.VolcProjectName,
		})
	}
	return bindings, nil
}

func resolveChannelForProject(userId int, requestedProjectName string) (*model.Channel, string, error) {
	bindings, err := getUserAssetBindings(userId)
	if err != nil {
		return nil, "", err
	}
	return resolveVolcEngineChannelForProject(bindings, requestedProjectName)
}

func resolveChannelForGroup(userId int, groupID, requestedProjectName string) (*model.Channel, string, error) {
	mapping, err := model.GetAssetGroupMapping(userId, groupID)
	if err != nil {
		return nil, "", fmt.Errorf("query asset group mapping failed: %w", err)
	}
	if mapping == nil {
		return nil, "", fmt.Errorf("asset group %s does not belong to current user", groupID)
	}
	return resolveMappedVolcEngineChannel(volcAssetGroupBinding{
		GroupID:     mapping.GroupId,
		ChannelID:   mapping.ChannelId,
		ProjectName: mapping.VolcProjectName,
	}, requestedProjectName)
}

func resolveChannelForUserAssetProject(userId int, requestedProjectName string) (*model.Channel, string, error) {
	bindings, err := getUserAssetBindings(userId)
	if err != nil {
		return nil, "", err
	}
	channel, projectName, _, err := resolveVolcAssetGroupScope(bindings, requestedProjectName, nil)
	return channel, projectName, err
}

// CreateUserAssetMapping 在用户完成真人认证后创建映射
func CreateUserAssetMapping(userId int, groupId string, channelId int, projectName string) (*model.AssetGroupMapping, error) {
	existing, err := model.GetAssetGroupMapping(userId, groupId)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.ChannelId != channelId || normalizeVolcProjectName(existing.VolcProjectName) != normalizeVolcProjectName(projectName) {
			return nil, fmt.Errorf("asset group %s mapping conflicts with channel or project", groupId)
		}
		return existing, nil // 该 group_id 已存在，直接返回
	}

	mapping := &model.AssetGroupMapping{
		UserId:          userId,
		GroupId:         groupId,
		ChannelId:       channelId,
		VolcProjectName: normalizeVolcProjectName(projectName),
	}
	if err := model.InsertAssetGroupMapping(mapping); err != nil {
		return nil, fmt.Errorf("insert asset group mapping failed: %w", err)
	}
	return mapping, nil
}

// GetGroupIDsForUser 获取用户拥有的所有 group IDs，用于自动过滤
func GetGroupIDsForUser(userId int) ([]string, error) {
	mappings, err := model.GetUserAssetGroupMappings(userId)
	if err != nil {
		return nil, err
	}
	if len(mappings) == 0 {
		return nil, fmt.Errorf("user %d has not completed liveness verification yet", userId)
	}
	groupIDs := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		groupIDs = append(groupIDs, mapping.GroupId)
	}
	return groupIDs, nil
}

// ---- 业务方法 ----

// CreateVisualValidateSession 创建真人认证会话
func CreateVisualValidateSession(ctx context.Context, userId int, callbackURL, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForProject(userId, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"CallbackURL": callbackURL,
		"ProjectName": projectName,
	}
	return CallArkAPI(ctx, channel, "CreateVisualValidateSession", reqBody)
}

// GetVisualValidateResult 获取认证结果并自动创建映射
func GetVisualValidateResult(ctx context.Context, userId int, bytedToken, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForProject(userId, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"BytedToken":  bytedToken,
		"ProjectName": projectName,
	}
	resp, err := CallArkAPI(ctx, channel, "GetVisualValidateResult", reqBody)
	if err != nil {
		return nil, err
	}

	// 从响应结果中提取 GroupId，创建映射
	if result, ok := resp["Result"].(map[string]any); ok {
		if groupId, ok := result["GroupId"].(string); groupId != "" && ok {
			if _, err := CreateUserAssetMapping(userId, groupId, channel.Id, projectName); err != nil {
				return nil, fmt.Errorf("liveness verification created group %s but failed to persist its mapping: %w", groupId, err)
			}
		}
	}

	return resp, nil
}

// CreateAsset 创建素材
func CreateAsset(ctx context.Context, userId int, groupID, assetURL, assetType, name, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForGroup(userId, groupID, requestedProjectName)
	if err != nil {
		return nil, err
	}
	return CallArkAPI(ctx, channel, "CreateAsset", buildCreateAssetRequest(groupID, assetURL, assetType, name, projectName))
}

type ListAssetsInput struct {
	GroupIDs    []string
	GroupType   string
	Statuses    []string
	Name        string
	ProjectName string
	PageNumber  int
	PageSize    int
	SortBy      string
	SortOrder   string
}

// ListAssets 查询素材列表（自动按用户隔离）
func ListAssets(ctx context.Context, userId int, req *ListAssetsInput) (map[string]any, error) {
	bindings, err := getUserAssetBindings(userId)
	if err != nil {
		return nil, err
	}
	channel, projectName, userGroupIDs, err := resolveVolcAssetGroupScope(bindings, req.ProjectName, req.GroupIDs)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Filter": map[string]any{
			"GroupIds": userGroupIDs,
		},
	}
	filter := reqBody["Filter"].(map[string]any)
	if req.GroupType != "" {
		filter["GroupType"] = req.GroupType
	} else {
		// s1 路径默认为 LivenessFace
		filter["GroupType"] = "LivenessFace"
	}
	if len(req.Statuses) > 0 {
		filter["Statuses"] = req.Statuses
	}
	if req.Name != "" {
		filter["Name"] = req.Name
	}
	reqBody["ProjectName"] = projectName
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	if req.SortBy != "" {
		reqBody["SortBy"] = req.SortBy
	}
	if req.SortOrder != "" {
		reqBody["SortOrder"] = req.SortOrder
	}
	return CallArkAPI(ctx, channel, "ListAssets", reqBody)
}

// GetAsset 获取单个素材
func GetAsset(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForUserAssetProject(userId, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAsset", reqBody)
}

// UpdateAsset 更新素材
func UpdateAsset(ctx context.Context, userId int, id, name, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForUserAssetProject(userId, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "UpdateAsset", reqBody)
}

// DeleteAsset 删除素材
func DeleteAsset(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForUserAssetProject(userId, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "DeleteAsset", reqBody)
}

type ListAssetGroupsInput struct {
	Name        string
	GroupIDs    []string
	GroupType   string
	ProjectName string
	PageNumber  int
	PageSize    int
	SortBy      string
	SortOrder   string
}

// ListAssetGroups 查询素材组列表（自动按用户隔离）
func ListAssetGroups(ctx context.Context, userId int, req *ListAssetGroupsInput) (map[string]any, error) {
	bindings, err := getUserAssetBindings(userId)
	if err != nil {
		return nil, err
	}
	channel, projectName, userGroupIDs, err := resolveVolcAssetGroupScope(bindings, req.ProjectName, req.GroupIDs)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Filter": map[string]any{
			"GroupIds": userGroupIDs,
		},
	}
	filter := reqBody["Filter"].(map[string]any)
	if req.GroupType != "" {
		filter["GroupType"] = req.GroupType
	} else {
		// s1 路径默认为 LivenessFace
		filter["GroupType"] = "LivenessFace"
	}
	if req.Name != "" {
		filter["Name"] = req.Name
	}
	reqBody["ProjectName"] = projectName
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	if req.SortBy != "" {
		reqBody["SortBy"] = req.SortBy
	}
	if req.SortOrder != "" {
		reqBody["SortOrder"] = req.SortOrder
	}
	return CallArkAPI(ctx, channel, "ListAssetGroups", reqBody)
}

// GetAssetGroup 获取素材组
func GetAssetGroup(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForGroup(userId, id, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAssetGroup", reqBody)
}

// UpdateAssetGroup 更新素材组
func UpdateAssetGroup(ctx context.Context, userId int, id, name, title, description, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForGroup(userId, id, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	if title != "" {
		reqBody["Title"] = title
	}
	if description != "" {
		reqBody["Description"] = description
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "UpdateAssetGroup", reqBody)
}

// DeleteAssetGroup 删除素材组
func DeleteAssetGroup(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveChannelForGroup(userId, id, requestedProjectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	resp, err := CallArkAPI(ctx, channel, "DeleteAssetGroup", reqBody)
	if err != nil {
		return nil, err
	}

	// 删除成功后，清理本地映射
	if err := model.DeleteAssetGroupMapping(userId, id); err != nil {
		return nil, fmt.Errorf("failed to delete asset group mapping: %w", err)
	}

	return resp, nil
}
