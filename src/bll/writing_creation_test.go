package bll

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCreationInput(t *testing.T) {
	assert := assert.New(t)

	data, err := hex.DecodeString("a4636769644c00000000000000004d5bfcb8657469746c656e6669727374206372656174696f6e67636f6e74656e745859a2647479706563646f6367636f6e74656e7481a3647479706569706172616772617068656174747273a16269646631323334353667636f6e74656e7481a264746578746b48656c6c6f20776f726c6464747970656474657874686c616e677561676563656e67")
	require.NoError(t, err)

	var obj CreateCreationInput
	err = cbor.Unmarshal(data, &obj)
	require.NoError(t, err)
	assert.NoError(obj.Validate())

	str := `{"gid":"0000000000000jarvis0","id":"0000000000000jarvis0","updated_at":123,"status":0}`
	var input UpdateCreationStatusInput
	err = json.Unmarshal([]byte(str), &input)
	fmt.Println(input)
	require.NoError(t, err)
	assert.NoError(input.Validate())
}
