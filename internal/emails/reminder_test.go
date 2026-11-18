package emails

import (
	"strings"
	"testing"
)

func TestReminderEmailSubjectAndLine(t *testing.T) {
	def := &ReminderEmailVariables{}
	if def.Subject() != DefaultReminderEmailSubject || def.Line() != DefaultReminderEmailLine {
		t.Errorf("default: subject %q line %q", def.Subject(), def.Line())
	}

	custom := &ReminderEmailVariables{Message: "Breathe.\nThen write."}
	if custom.Subject() != "Breathe." {
		t.Errorf("custom subject should be the first line, got %q", custom.Subject())
	}
	if custom.Line() != "Breathe.\nThen write." {
		t.Errorf("custom body should replace the default, got %q", custom.Line())
	}

	long := &ReminderEmailVariables{Message: strings.Repeat("a", 200)}
	if got := []rune(long.Subject()); len(got) != reminderSubjectMaxRunes || got[len(got)-1] != '…' {
		t.Errorf("long subject should be shortened to %d runes with an ellipsis, got %d", reminderSubjectMaxRunes, len(got))
	}

	blankLine := &ReminderEmailVariables{Message: "\n  \n"}
	if blankLine.Subject() != DefaultReminderEmailSubject {
		t.Errorf("blank message should use the default subject, got %q", blankLine.Subject())
	}
}
