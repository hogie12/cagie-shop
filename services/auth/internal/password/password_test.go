package password

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Fatalf("unexpected format: %s", h)
	}
	if !Verify("correct horse", h) {
		t.Fatal("right password rejected")
	}
	if Verify("wrong horse", h) {
		t.Fatal("wrong password accepted")
	}
	h2, _ := Hash("correct horse")
	if h == h2 {
		t.Fatal("same password must produce different hashes (random salt)")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "plain", "$argon2id$v=19$m=1,t=1,p=1$!!$!!", "$bcrypt$v=19$m=1,t=1,p=1$YQ$YQ", "$argon2id$v=19$m=0,t=1,p=1$YQ$YQ"} {
		if Verify("x", bad) {
			t.Errorf("accepted malformed hash %q", bad)
		}
	}
}
