package utils

import (
	"github.com/gotestyourself/gotestyourself/assert"
	"testing"
)

func TestMatchService(t *testing.T) {
	assert.Equal(t, MatchService("redis"), ServiceConfig{
		ServiceImage: "redis",
		ServiceName:  "redis",
	}, "incorrect service match")
}

func TestMatchServiceSanitize(t *testing.T) {
	assert.Equal(t, MatchService("ruby on rails"), ServiceConfig{
		ServiceImage: "rails",
		ServiceName:  "ruby_on_rails",
	}, "incorrect service match")
}

func TestMatchServiceNoMatch(t *testing.T) {
	assert.Equal(t, MatchService("no-such_image"), ServiceConfig{
		ServiceImage: "alpine",
		ServiceName:  "no-such_image",
	}, "incorrect default service config")
}

func TestMatchServiceNoMatchSanitize(t *testing.T) {
	assert.Equal(t, MatchService("aspq[foo[bar]|"), ServiceConfig{
		ServiceImage: "alpine",
		ServiceName:  "aspq_foo_bar__",
	}, "incorrect default service config")
}
