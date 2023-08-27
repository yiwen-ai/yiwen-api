package bll

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(true, SupportModel(DefaultModel))
	assert.Equal(true, SupportModel("gpt4"))
	assert.Equal(false, SupportModel("davinci"))

	assert.Equal(float64(1.0), Pricing(DefaultModel))
	assert.Equal(float64(10.0), Pricing("gpt4"))
	assert.Equal(float64(0), Pricing("davinci"))

	assert.Equal(int64(1), CostWEN(DefaultModel, 100))
	assert.Equal(int64(2), CostWEN(DefaultModel, 1600))
	assert.Equal(int64(3), CostWEN(DefaultModel, 2100))
	assert.Equal(int64(1), CostWEN("gpt4", 100))
	assert.Equal(int64(15), CostWEN("gpt4", 1500))
}
