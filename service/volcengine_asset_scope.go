package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

const defaultVolcProjectName = "default"

type volcAssetGroupBinding struct {
	GroupID     string
	ChannelID   int
	ProjectName string
}

func normalizeVolcProjectName(projectName string) string {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		return defaultVolcProjectName
	}
	return projectName
}

func getDefaultVolcEngineChannel() (*model.Channel, error) {
	channel := &model.Channel{}
	err := model.DB.
		Where("type = ? AND status = ?", constant.ChannelTypeVolcEngine, common.ChannelStatusEnabled).
		Order("id ASC").
		First(channel).Error
	if err != nil {
		return nil, fmt.Errorf("no volcengine channel configured, please contact admin: %w", err)
	}
	return channel, nil
}

func resolveMappedVolcEngineChannel(binding volcAssetGroupBinding, requestedProjectName string) (*model.Channel, string, error) {
	if binding.ChannelID == 0 {
		return nil, "", fmt.Errorf("asset group %s has no channel mapping", binding.GroupID)
	}

	projectName := normalizeVolcProjectName(binding.ProjectName)
	if requested := strings.TrimSpace(requestedProjectName); requested != "" && requested != projectName {
		return nil, "", fmt.Errorf(
			"project_name %q does not match asset group %s project %q",
			requested,
			binding.GroupID,
			projectName,
		)
	}

	channel, err := model.GetChannelById(binding.ChannelID, true)
	if err != nil {
		return nil, "", fmt.Errorf("get channel %d for asset group %s failed: %w", binding.ChannelID, binding.GroupID, err)
	}
	return channel, projectName, nil
}

func resolveVolcEngineChannelForProject(bindings []volcAssetGroupBinding, requestedProjectName string) (*model.Channel, string, error) {
	projectName := normalizeVolcProjectName(requestedProjectName)
	channelID := 0
	fallbackChannelID := 0
	fallbackChannelAmbiguous := false
	for _, binding := range bindings {
		if binding.ChannelID == 0 {
			return nil, "", fmt.Errorf("asset group %s has no channel mapping", binding.GroupID)
		}
		if fallbackChannelID == 0 {
			fallbackChannelID = binding.ChannelID
		} else if fallbackChannelID != binding.ChannelID {
			fallbackChannelAmbiguous = true
		}
		if normalizeVolcProjectName(binding.ProjectName) != projectName {
			continue
		}
		if channelID != 0 && channelID != binding.ChannelID {
			return nil, "", fmt.Errorf("project %q is mapped to multiple volcengine channels", projectName)
		}
		channelID = binding.ChannelID
	}

	if channelID == 0 {
		if fallbackChannelAmbiguous {
			return nil, "", fmt.Errorf("project %q has no mapping and user mappings span multiple volcengine channels", projectName)
		}
		if fallbackChannelID == 0 {
			channel, err := getDefaultVolcEngineChannel()
			return channel, projectName, err
		}
		channelID = fallbackChannelID
	}

	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, "", fmt.Errorf("get channel %d for project %q failed: %w", channelID, projectName, err)
	}
	return channel, projectName, nil
}

func resolveVolcAssetGroupScope(
	bindings []volcAssetGroupBinding,
	requestedProjectName string,
	requestedGroupIDs []string,
) (*model.Channel, string, []string, error) {
	if len(bindings) == 0 {
		return nil, "", nil, fmt.Errorf("user has no asset group mapping")
	}

	projectName := strings.TrimSpace(requestedProjectName)
	if projectName == "" {
		projects := make(map[string]struct{})
		for _, binding := range bindings {
			projects[normalizeVolcProjectName(binding.ProjectName)] = struct{}{}
		}
		if len(projects) > 1 {
			return nil, "", nil, fmt.Errorf("project_name is required when asset groups span multiple projects")
		}
		for name := range projects {
			projectName = name
		}
	} else {
		projectName = normalizeVolcProjectName(projectName)
	}

	channelID := 0
	allowedGroupIDs := make(map[string]struct{})
	allGroupIDs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if normalizeVolcProjectName(binding.ProjectName) != projectName {
			continue
		}
		if binding.ChannelID == 0 {
			return nil, "", nil, fmt.Errorf("asset group %s has no channel mapping", binding.GroupID)
		}
		if channelID != 0 && channelID != binding.ChannelID {
			return nil, "", nil, fmt.Errorf("project %q is mapped to multiple volcengine channels", projectName)
		}
		channelID = binding.ChannelID
		allowedGroupIDs[binding.GroupID] = struct{}{}
		allGroupIDs = append(allGroupIDs, binding.GroupID)
	}
	if channelID == 0 {
		return nil, "", nil, fmt.Errorf("user has no asset group mapping in project %q", projectName)
	}

	groupIDs := allGroupIDs
	if len(requestedGroupIDs) > 0 {
		groupIDs = make([]string, 0, len(requestedGroupIDs))
		seen := make(map[string]struct{}, len(requestedGroupIDs))
		for _, groupID := range requestedGroupIDs {
			groupID = strings.TrimSpace(groupID)
			if groupID == "" {
				return nil, "", nil, fmt.Errorf("group_id must not be empty")
			}
			if _, ok := allowedGroupIDs[groupID]; !ok {
				return nil, "", nil, fmt.Errorf("asset group %s does not belong to current user in project %q", groupID, projectName)
			}
			if _, ok := seen[groupID]; ok {
				continue
			}
			seen[groupID] = struct{}{}
			groupIDs = append(groupIDs, groupID)
		}
	}

	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, "", nil, fmt.Errorf("get channel %d for project %q failed: %w", channelID, projectName, err)
	}
	return channel, projectName, groupIDs, nil
}

func buildCreateAssetRequest(groupID, assetURL, assetType, name, projectName string) map[string]any {
	reqBody := map[string]any{
		"GroupId":     groupID,
		"URL":         assetURL,
		"AssetType":   assetType,
		"ProjectName": projectName,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	return reqBody
}
