package doubao

import (
	"strings"
	"time"
)

var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
	"doubao-seedance-2-0-mini-260615",
}

var ChannelName = "doubao-video"

// videoPriceKey 价格表的键：输出分辨率档（is1080p/is4k 均为 false 即 480p/720p 基准档）、输入是否含视频。
type videoPriceKey struct {
	is1080p  bool
	is4k     bool
	hasVideo bool
}

// videoPriceTable 各模型在不同 (输出分辨率档, 是否含视频输入) 下的单价（元/百万 token）。
// 其中零值键 {480p/720p, 不含视频} 为基准价，等于管理员应配置的 ModelRatio；
// 计费时取 实际单价/基准价 作为 OtherRatio。
var videoPriceTable = map[string]map[videoPriceKey]float64{
	"doubao-seedance-2-0-260128": {
		{hasVideo: false}:                46.0,
		{hasVideo: true}:                 28.0,
		{is1080p: true, hasVideo: false}: 51.0,
		{is1080p: true, hasVideo: true}:  31.0,
		{is4k: true, hasVideo: false}:    26.0,
		{is4k: true, hasVideo: true}:     16.0,
	},
	"doubao-seedance-2-0-fast-260128": {
		{hasVideo: false}: 37.0,
		{hasVideo: true}:  22.0,
	},
	"doubao-seedance-2-0-mini-260615": {
		{hasVideo: false}: 23.0,
		{hasVideo: true}:  14.0,
	},
}

// GetVideoInputRatio 返回指定模型在给定输出分辨率/是否含视频输入下，相对基准价的计费倍率。
// 第二个返回值表示该模型是否配置了价格表；倍率为 1.0 时调用方可忽略该 OtherRatio。
func GetVideoInputRatio(modelName, resolution string, hasVideo bool) (float64, bool) {
	prices, ok := videoPriceTable[modelName]
	base := prices[videoPriceKey{}] // 零值键 = {480p/720p, 不含视频} 基准价
	if !ok || base <= 0 {
		return 0, false
	}
	res := strings.ToLower(strings.TrimSpace(resolution))
	price, ok := prices[videoPriceKey{is1080p: res == "1080p", is4k: res == "4k", hasVideo: hasVideo}]
	if !ok {
		// 未配置的组合（如 fast 无 1080p/4k，上游会自行报错）按基准价计费即可。
		return 1.0, true
	}
	return price / base, true
}

// seedanceTierPriceMap 官方分档价格（¥/百万tokens）。
// key 说明: "low"=480p/720p 无视频, "low_video"=480p/720p 有视频,
//
//	"high"=1080p 无视频, "high_video"=1080p 有视频；
//	"ultra"=4k 无视频, "ultra_video"=4k 有视频。
//
// 预扣和结算都直接使用此价格，公式：quota = tokens / 1,000,000 × tierPrice × QuotaPerUnit × groupRatio。
var seedanceTierPriceMap = map[string]map[string]float64{
	"doubao-seedance-2-0-260128": {
		"low":         46.0, // 480p/720p 无视频
		"low_video":   28.0, // 480p/720p 有视频
		"high":        51.0, // 1080p 无视频
		"high_video":  31.0, // 1080p 有视频
		"ultra":       26.0, // 4k 无视频
		"ultra_video": 16.0, // 4k 有视频
	},
	"doubao-seedance-2-0-fast-260128": {
		"low":       37.0, // 480p/720p 无视频
		"low_video": 22.0, // 480p/720p 有视频
	},
	"doubao-seedance-2-0-mini-260615": {
		"low":       23.0, // 480p/720p 无视频
		"low_video": 14.0, // 480p/720p 有视频
	},
	"doubao-seedance-2-5-260628": {
		"low":        70.0, // 480p/720p 无视频
		"low_video":  42.0, // 480p/720p 有视频
		"high":       77.0, // 1080p 无视频
		"high_video": 46.0, // 1080p 有视频
	},
}

// GetSeedanceTierPrice 根据模型名获取种子舞官方分档价格映射。
func GetSeedanceTierPrice(modelName string) (map[string]float64, bool) {
	m, ok := seedanceTierPriceMap[modelName]
	return m, ok
}

// seedanceLimitedDiscountRule 描述自定义补丁维护的限时折扣。
// 起止日期均按北京时间计算，并且首尾日期都包含在生效范围内。
type seedanceLimitedDiscountRule struct {
	modelName  string
	startDate  time.Time
	endDate    time.Time
	tierKeys   map[string]struct{}
	multiplier float64
}

var seedanceBeijingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

var seedanceLimitedDiscountRules = []seedanceLimitedDiscountRule{
	{
		modelName:  "doubao-seedance-2-0-mini-260615",
		startDate:  time.Date(2026, time.August, 7, 0, 0, 0, 0, seedanceBeijingLocation),
		endDate:    time.Date(2026, time.September, 7, 0, 0, 0, 0, seedanceBeijingLocation),
		multiplier: 0.4,
	},
	{
		modelName:  "doubao-seedance-2-0-fast-260128",
		startDate:  time.Date(2026, time.August, 7, 0, 0, 0, 0, seedanceBeijingLocation),
		endDate:    time.Date(2026, time.September, 7, 0, 0, 0, 0, seedanceBeijingLocation),
		multiplier: 0.75,
	},
	{
		modelName: "doubao-seedance-2-5-260628",
		startDate: time.Date(2026, time.August, 18, 0, 0, 0, 0, seedanceBeijingLocation),
		endDate:   time.Date(2026, time.September, 17, 0, 0, 0, 0, seedanceBeijingLocation),
		tierKeys: map[string]struct{}{
			"high":       {},
			"high_video": {},
		},
		multiplier: 0.72,
	},
}

// seedanceDiscountMultiplier 返回指定模型和分辨率档位在给定时间的限时折扣。
// 空 tierKeys 表示该模型当前已配置的所有档位都适用。
func seedanceDiscountMultiplier(modelName, tierKey string, now time.Time) float64 {
	beijingNow := now.In(seedanceBeijingLocation)
	currentDate := time.Date(
		beijingNow.Year(),
		beijingNow.Month(),
		beijingNow.Day(),
		0,
		0,
		0,
		0,
		seedanceBeijingLocation,
	)

	for _, rule := range seedanceLimitedDiscountRules {
		if rule.modelName != modelName || currentDate.Before(rule.startDate) || currentDate.After(rule.endDate) {
			continue
		}
		if len(rule.tierKeys) > 0 {
			if _, ok := rule.tierKeys[tierKey]; !ok {
				continue
			}
		}
		return rule.multiplier
	}
	return 1.0
}

// GetSeedanceTierPriceWithDiscount 返回计费时应使用的有效分档价格。
// GetSeedanceTierPrice 始终返回基础价，便于保留官方价格和记录；限时折扣只在计费路径中按当前北京时间叠加。
func GetSeedanceTierPriceWithDiscount(modelName, tierKey string, now time.Time) (float64, bool) {
	prices, ok := GetSeedanceTierPrice(modelName)
	if !ok {
		return 0, false
	}
	price, ok := prices[tierKey]
	if !ok {
		return 0, false
	}
	return price * seedanceDiscountMultiplier(modelName, tierKey, now), true
}

// GetSeedanceTierPriceForUsage resolves the effective token price for a
// validated task-plugin usage record. Unsupported high-resolution entries
// retain the legacy fallback to the model's low tier.
func GetSeedanceTierPriceForUsage(modelName, resolution string, hasVideo bool, now time.Time) (string, float64, bool) {
	tier := "low"
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "1080p":
		tier = "high"
	case "4k":
		tier = "ultra"
	}

	key := tier
	if hasVideo {
		key += "_video"
	}
	if price, ok := GetSeedanceTierPriceWithDiscount(modelName, key, now); ok {
		return key, price, true
	}

	lowKey := "low"
	if hasVideo {
		lowKey = "low_video"
	}
	if price, ok := GetSeedanceTierPriceWithDiscount(modelName, lowKey, now); ok {
		return lowKey, price, true
	}
	price, ok := GetSeedanceTierPriceWithDiscount(modelName, "low", now)
	return "low", price, ok
}
