package notification

import "testing"

func TestDefaultPreference(t *testing.T) {
	t.Parallel()

	preference := DefaultPreference("profile-1")

	if preference.RecipientProfileID != "profile-1" {
		t.Fatalf("unexpected recipient profile id: %q", preference.RecipientProfileID)
	}
	if !preference.PushEnabled || !preference.InAppEnabled || !preference.EmailEnabled {
		t.Fatal("default preference should enable all delivery channels")
	}
	if preference.QuietHoursEnabled || preference.Muted {
		t.Fatal("default preference should not enable quiet hours or mute recipient")
	}
}
