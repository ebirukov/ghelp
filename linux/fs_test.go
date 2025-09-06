package linux

import (
	"os"
	"strings"
	"testing"
)

func TestFS_Mount(t *testing.T) {
	tests := []struct {
		name        string
		fs          *FS
		setup       func() error
		cleanup     func() error
		wantErr     bool
		errContains string
	}{
		{
			name:        "mount point already exists",
			fs:          ProcFS,
			wantErr:     true,
			errContains: ErrMountExist.Error(),
		},
		{
			name: "mount target permission denied",
			fs: NewFS(
				"proc",
				"/root/protected_dir/proc",
				"proc",
				checkExist("self"),
			),
			wantErr:     true,
			errContains: "check mount point",
		},
		{
			name: "mount target permission denied without check",
			fs: NewFS(
				"proc",
				"/root/protected_dir/proc",
				"proc",
				nil,
			),
			wantErr:     true,
			errContains: "mount error",
		},
		{
			name:        "mount operation error",
			fs:          TempFS,
			wantErr:     true,
			errContains: "mount error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Cleanup после теста
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			// Выполнение
			err := tt.fs.Mount()

			// Проверка ошибок
			if (err != nil) != tt.wantErr {
				t.Errorf("Mount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Проверка содержимого ошибки
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
					return
				}
				if err.Error() != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error message '%s' should contain '%s'", err.Error(), tt.errContains)
				}
			}

			// Если mount успешен, проверяем что точка монтирования существует
			if !tt.wantErr && err == nil {
				if _, err := os.Stat(tt.fs.target); os.IsNotExist(err) {
					t.Errorf("Mount point %s should exist after successful mount", tt.fs.target)
				}
			}
		})
	}
}
