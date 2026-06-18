package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	type args struct {
		databaseURL string
	}
	type want struct {
		wantErr bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "異常系 DATABASE_URL 未指定で di.InitHandler が失敗し起動前に error",
			args: args{databaseURL: ""},
			want: want{wantErr: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.args.databaseURL)
			err := run()
			if tt.want.wantErr && err == nil {
				t.Fatal("run(): want error")
			}
			if !tt.want.wantErr && err != nil {
				t.Fatalf("run() = %v, want nil", err)
			}
		})
	}
}
