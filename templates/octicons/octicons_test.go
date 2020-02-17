package octicons

import "testing"

func TestOcticon(t *testing.T) {
	t.Run("fails when the octicon doesn't exis", func(t *testing.T) {
		_, err := Icon("octicon")
		if err == nil {
			t.Error("expected an error")
		}
	})
}
