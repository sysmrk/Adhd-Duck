package imageref

import (
	"strings"
	"testing"
)

func TestImageValidationAndPinnedBinding(t *testing.T) {
	root := "sha256:" + strings.Repeat("a", 64)
	for _, image := range []string{"busybox", "busybox:1.37.0", "localhost:5000/team/app:v1", "team/app@" + root} {
		if err := Validate(image); err != nil {
			t.Fatal(image, err)
		}
	}
	for _, image := range []string{"", "https://registry.test/a:v1", "app;touch /tmp/no", "UPPER/Invalid:tag", "app@sha256:bad", strings.Repeat("x", 513)} {
		if Validate(image) == nil {
			t.Fatal("accepted", image)
		}
	}
	if err := MatchDigest("team/app@"+root, root); err != nil {
		t.Fatal(err)
	}
	if MatchDigest("team/app@"+root, "sha256:"+strings.Repeat("b", 64)) == nil {
		t.Fatal("accepted pinned mismatch")
	}
	if err := MatchDigest("team/app:tag", root); err != nil {
		t.Fatal(err)
	}
}
