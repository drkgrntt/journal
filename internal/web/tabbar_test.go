package web

import "testing"

func TestActiveTab(t *testing.T) {
	cases := map[string]string{
		"/dashboard":              "today",
		"/journal/new":            "today",
		"/journal":                "journal",
		"/journal/4b1e-uuid":      "journal",
		"/journal/4b1e-uuid/edit": "journal",
		"/action-items":           "actions",
		"/action-items/some-id":   "actions",
		"/profile":                "profile",
		"/reminders":              "",
		"/journalism":             "",
		"/action-itemsX":          "",
		"/":                       "",
		"/blog":                   "",
	}
	for path, want := range cases {
		if got := activeTab(path); got != want {
			t.Errorf("activeTab(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestEveryTabCanBeActive(t *testing.T) {
	for _, item := range tabItems {
		if got := activeTab(item.Href); got != item.Key {
			t.Errorf("tab %q links to %q, which activates %q", item.Key, item.Href, got)
		}
	}
}
