package ipfix

import (
	"strings"
	"testing"
)

func TestCapitalization(t *testing.T) {
	t.Parallel()
	t.Run("lowerCase", func(t *testing.T) {
		name := "lowerCase"
		s := strings.ToUpper(string([]rune(name)[0:1])) // UTF-8
		t.Log("reversed" + s + name[1:])
	})
}
