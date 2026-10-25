package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{512, "512 B"},
		{4300, "4.2 KB"},
		{1363149, "1.3 MB"},
		{2254857830, "2.1 GB"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, FormatSize(c.n))
	}
}

func TestFormatMode(t *testing.T) {
	assert.Equal(t, "drwxr-xr-x", FormatMode(os.ModeDir|0755))
	assert.Equal(t, "-rw-r--r--", FormatMode(0644))
}
