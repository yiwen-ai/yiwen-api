package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	t.Run("NormalizeFileEncodingAndType", func(t *testing.T) {
		file1, err := os.ReadFile("./file_test_gb18030.md")
		require.NoError(t, err)

		file2, err := os.ReadFile("./file_test_utf8.md")
		require.NoError(t, err)

		buf, mt, err := NormalizeFileEncodingAndType(file1, "text/markdown")
		require.NoError(t, err)
		assert.Equal(t, "text/markdown", mt)
		assert.Equal(t, file2, buf)
	})
}
