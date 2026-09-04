package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChannelBaseURLIsBoundsSafe(t *testing.T) {
	assert.Empty(t, GetChannelBaseURL(ChannelTypeTaskPlugin))
	assert.Empty(t, GetChannelBaseURL(9999))
}

func TestCanonicalChannelTypeIDs(t *testing.T) {
	assert.Equal(t, 45, ChannelTypeVolcEngine)
	assert.Equal(t, 58, ChannelTypeAdvancedCustom)
	assert.Equal(t, 59, ChannelTypeSub2API)
	assert.Equal(t, 60, ChannelTypeNewAPI)
	assert.Equal(t, 61, ChannelTypeTaskPlugin)
}
