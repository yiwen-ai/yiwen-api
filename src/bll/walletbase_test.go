package bll

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	assert := assert.New(t)

	g35 := GetAIModel("GPT-3.5")
	g4 := GetAIModel("gpt-4")

	assert.Equal(g35, GetAIModel("davinci"))

	assert.Equal(float64(0.6), g35.Price)
	assert.Equal(float64(8.0), g4.Price)

	assert.Equal(int64(1), g35.CostWEN(100))
	assert.Equal(int64(1), g35.CostWEN(1600))
	assert.Equal(int64(2), g35.CostWEN(2100))
	assert.Equal(int64(2), g35.CostWEN(2597))
	assert.Equal(int64(6), g35.CostWEN(10000))
	assert.Equal(int64(1), g4.CostWEN(100))
	assert.Equal(int64(12), g4.CostWEN(1500))
	assert.Equal(int64(21), g4.CostWEN(2597))
	assert.Equal(int64(80), g4.CostWEN(10000))
}
