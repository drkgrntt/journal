package web

import (
	"strings"
	"testing"
)

func TestServiceWorkerSourceStampsAssetVersion(t *testing.T) {
	src, err := ServiceWorkerSource()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "__ASSET_VERSION__") {
		t.Error("placeholder was not replaced")
	}
	if !strings.Contains(string(src), `const VERSION = "`+AssetVersion+`"`) {
		t.Error("asset version missing from the worker source")
	}
}
