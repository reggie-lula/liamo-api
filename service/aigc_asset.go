package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/model"
)

func getUserAIGCAssetBindings(userId int) ([]volcAssetGroupBinding, error) {
	mappings, err := model.GetUserAIGCAssetGroupMappings(userId)
	if err != nil {
		return nil, fmt.Errorf("query user AIGC asset group mappings failed: %w", err)
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

func resolveAIGCChannelForProject(userId int, requestedProjectName string) (*model.Channel, string, error) {
	bindings, err := getUserAIGCAssetBindings(userId)
	if err != nil {
		return nil, "", err
	}
	return resolveVolcEngineChannelForProject(bindings, requestedProjectName)
}

func resolveAIGCChannelForGroup(userId int, groupID, requestedProjectName string) (*model.Channel, string, error) {
	mapping, err := model.GetAIGCAssetGroupMapping(userId, groupID)
	if err != nil {
		return nil, "", fmt.Errorf("query AIGC asset group mapping failed: %w", err)
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

func resolveAIGCChannelForUserAssetProject(userId int, requestedProjectName string) (*model.Channel, string, error) {
	bindings, err := getUserAIGCAssetBindings(userId)
	if err != nil {
		return nil, "", err
	}
	channel, projectName, _, err := resolveVolcAssetGroupScope(bindings, requestedProjectName, nil)
	return channel, projectName, err
}

// CreateAIGCAssetGroup 创建素材资产组合
func CreateAIGCAssetGroup(ctx context.Context, userId int, name, description, projectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForProject(userId, projectName)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Name":        name,
		"Description": description,
		"GroupType":   "AIGC",
		"ProjectName": projectName,
	}
	resp, err := CallArkAPI(ctx, channel, "CreateAssetGroup", reqBody)
	if err != nil {
		return nil, err
	}

	result, ok := resp["Result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("CreateAssetGroup response is missing Result")
	}
	groupId, ok := result["Id"].(string)
	if !ok || groupId == "" {
		return nil, fmt.Errorf("CreateAssetGroup response is missing Result.Id")
	}
	if _, err := CreateAIGCAssetMapping(userId, groupId, channel.Id, projectName); err != nil {
		return nil, fmt.Errorf("CreateAssetGroup created group %s but failed to persist its mapping: %w", groupId, err)
	}

	return resp, nil
}

// CreateAIGCAssetMapping 在用户创建素材组后创建映射
func CreateAIGCAssetMapping(userId int, groupId string, channelId int, projectName string) (*model.AIGCAssetGroupMapping, error) {
	existing, err := model.GetAIGCAssetGroupMapping(userId, groupId)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.ChannelId != channelId || normalizeVolcProjectName(existing.VolcProjectName) != normalizeVolcProjectName(projectName) {
			return nil, fmt.Errorf("AIGC asset group %s mapping conflicts with channel or project", groupId)
		}
		return existing, nil // 该 group_id 已存在，直接返回
	}

	mapping := &model.AIGCAssetGroupMapping{
		UserId:          userId,
		GroupId:         groupId,
		ChannelId:       channelId,
		VolcProjectName: normalizeVolcProjectName(projectName),
	}
	if err := model.InsertAIGCAssetGroupMapping(mapping); err != nil {
		return nil, fmt.Errorf("insert AIGC asset group mapping failed: %w", err)
	}
	return mapping, nil
}

// CreateAIGCAsset 创建素材
func CreateAIGCAsset(ctx context.Context, userId int, groupID, assetURL, assetType, name, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForGroup(userId, groupID, requestedProjectName)
	if err != nil {
		return nil, err
	}
	return CallArkAPI(ctx, channel, "CreateAsset", buildCreateAssetRequest(groupID, assetURL, assetType, name, projectName))
}

type ListAIGCAssetsInput struct {
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

// ListAIGCAssets 查询素材列表（自动按用户隔离）
func ListAIGCAssets(ctx context.Context, userId int, req *ListAIGCAssetsInput) (map[string]any, error) {
	bindings, err := getUserAIGCAssetBindings(userId)
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

// GetAIGCAsset 获取单个素材
func GetAIGCAsset(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUserAssetProject(userId, requestedProjectName)
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

// UpdateAIGCAsset 更新素材
func UpdateAIGCAsset(ctx context.Context, userId int, id, name, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUserAssetProject(userId, requestedProjectName)
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

// DeleteAIGCAsset 删除素材
func DeleteAIGCAsset(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUserAssetProject(userId, requestedProjectName)
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

type ListAIGCAssetGroupsInput struct {
	Name        string
	GroupIDs    []string
	GroupType   string
	ProjectName string
	PageNumber  int
	PageSize    int
}

// ListAIGCAssetGroups 查询素材组列表（自动按用户隔离）
func ListAIGCAssetGroups(ctx context.Context, userId int, req *ListAIGCAssetGroupsInput) (map[string]any, error) {
	bindings, err := getUserAIGCAssetBindings(userId)
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
	if req.Name != "" {
		filter["Name"] = req.Name
	}
	if req.GroupType != "" {
		filter["GroupType"] = req.GroupType
	}
	reqBody["ProjectName"] = projectName
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	return CallArkAPI(ctx, channel, "ListAssetGroups", reqBody)
}

// GetAIGCAssetGroup 获取素材组
func GetAIGCAssetGroup(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForGroup(userId, id, requestedProjectName)
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

// UpdateAIGCAssetGroup 更新素材组
func UpdateAIGCAssetGroup(ctx context.Context, userId int, id, name, title, description, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForGroup(userId, id, requestedProjectName)
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

// DeleteAIGCAssetGroup 删除素材组
func DeleteAIGCAssetGroup(ctx context.Context, userId int, id, requestedProjectName string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForGroup(userId, id, requestedProjectName)
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
	if err := model.DeleteAIGCAssetGroupMapping(userId, id); err != nil {
		return nil, fmt.Errorf("failed to delete asset group mapping: %w", err)
	}

	return resp, nil
}
