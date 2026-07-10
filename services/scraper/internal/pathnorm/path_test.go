package pathnorm

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific path shapes")
	}
	got, err := Abs("/C:/Users/shche/Documents/monorepo/services/scraper/cmd/scraper/session.json")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
	assert.Equal(t, `C:\Users\shche\Documents\monorepo\services\scraper\cmd\scraper\session.json`, got)

	got, err = Abs(`C:\Users\shche\foo\session.json`)
	require.NoError(t, err)
	assert.Equal(t, `C:\Users\shche\foo\session.json`, got)
}
