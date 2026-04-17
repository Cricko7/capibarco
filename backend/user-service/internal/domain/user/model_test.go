package user

import "testing"

func TestReviewValidate(t *testing.T) {
	tests := []struct {
		name    string
		r       Review
		wantErr bool
	}{
		{"ok", Review{TargetProfileID: "a", AuthorProfileID: "b", Text: "ok", Rating: 5}, false},
		{"bad rating", Review{TargetProfileID: "a", AuthorProfileID: "b", Text: "ok", Rating: 6}, true},
		{"missing target", Review{AuthorProfileID: "b", Text: "ok", Rating: 5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
