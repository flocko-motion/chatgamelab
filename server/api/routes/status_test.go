package routes

import (
	"testing"
	"time"

	"cgl/obj"
)

func TestGradeBackup(t *testing.T) {
	maxAge := 48 * time.Hour
	entry := func(success bool, age time.Duration) *obj.BackupLog {
		return &obj.BackupLog{Success: success, FinishedAt: time.Now().Add(-age)}
	}

	tests := []struct {
		name  string
		entry *obj.BackupLog
		want  string
	}{
		{"no run ever recorded", nil, backupUnknown},
		{"recent success", entry(true, time.Hour), backupOk},
		{"success just inside the limit", entry(true, 47*time.Hour), backupOk},
		{"success past the limit", entry(true, 49*time.Hour), backupStale},
		{"failure", entry(false, time.Hour), backupFailed},
		{"old failure stays failed", entry(false, 200*time.Hour), backupFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, age := gradeBackup(tt.entry, maxAge)
			if got != tt.want {
				t.Errorf("gradeBackup() = %q, want %q", got, tt.want)
			}
			if (age == nil) != (tt.entry == nil) {
				t.Errorf("gradeBackup() age = %v, want nil only when there is no entry", age)
			}
		})
	}
}
