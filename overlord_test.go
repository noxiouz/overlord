package overlord

import (
	"testing"
)

func TestMain(t *testing.T) {
	cfg := OverlordConfig{
		Path:         "/Users/noxiouz/suicide_echo/m.py",
		Locator:      "127.0.0.1:10053",
		HTTPEndpoint: ":8080",
	}
	over, _ := NewOverlord(&cfg)

	err := over.Start()
	t.Log(err)
}
