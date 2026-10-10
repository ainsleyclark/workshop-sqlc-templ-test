package main

import (
	"encoding/json"
	"testing"
)

func TestClean(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"plain":         {in: "sk-ant-1\n", want: "sk-ant-1"},
		"double quoted": {in: `"sk-ant-1"` + "\r\n", want: "sk-ant-1"},
		"single quoted": {in: "  'sk-ant-1' ", want: "sk-ant-1"},
		"empty":         {in: "\n", want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := clean(tc.in); got != tc.want {
				t.Errorf("clean(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing string
		wantEnv  map[string]any
		wantKeep string
		wantErr  bool
	}{
		"no file": {
			existing: "",
			wantEnv:  map[string]any{"ANTHROPIC_API_KEY": "k"},
		},
		"other settings kept": {
			existing: `{"model": "opus", "env": {"FOO": "bar"}}`,
			wantEnv:  map[string]any{"ANTHROPIC_API_KEY": "k", "FOO": "bar"},
			wantKeep: "model",
		},
		"key replaced": {
			existing: `{"env": {"ANTHROPIC_API_KEY": "old"}}`,
			wantEnv:  map[string]any{"ANTHROPIC_API_KEY": "k"},
		},
		"bad json": {
			existing: `{not json`,
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, err := merge([]byte(tc.existing), "k")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("output is not JSON: %v\n%s", err, out)
			}
			env, _ := got["env"].(map[string]any)
			for k, v := range tc.wantEnv {
				if env[k] != v {
					t.Errorf("env[%q] = %v, want %v", k, env[k], v)
				}
			}
			if tc.wantKeep != "" {
				if _, ok := got[tc.wantKeep]; !ok {
					t.Errorf("setting %q was dropped", tc.wantKeep)
				}
			}
			if out[len(out)-1] != '\n' {
				t.Error("output should end with a newline")
			}
		})
	}
}
