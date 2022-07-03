package main

import (
	"os"
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hismailbulut/neoray/src/common"
)

func TestMain(m *testing.M) {
	// We need to initialize glfw for GetKeyName function
	if glfw.Init() != nil {
		return
	}
	c := m.Run()
	glfw.Terminate()
	os.Exit(c)
}

func Test_parseCharInput(t *testing.T) {
	type args struct {
		char rune
		mods common.BitMask
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Ctrl + AltGr + Backslash",
			args: args{
				char: '\\',
				mods: ModControl | ModAltGr,
			},
			want: "<C-Bslash>",
		},
		{
			name: "Ctrl + Alt + AltGr + Bracket",
			args: args{
				char: '}',
				mods: ModControl | ModAlt | ModAltGr,
			},
			want: "<M-C-}>",
		},
		{
			name: "Ctrl + Alt + Sharp",
			args: args{
				char: '#',
				mods: ModControl | ModAlt,
			},
			want: "", // handled in key callback
		},
		{
			name: "Shift + S",
			args: args{
				char: 'S',
				mods: ModShift,
			},
			want: "S",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCharInput(tt.args.char, tt.args.mods); got != tt.want {
				t.Errorf("parseCharInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseKeyInput(t *testing.T) {
	type args struct {
		key      glfw.Key
		scancode int
		mods     common.BitMask
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "AltGr + Enter",
			args: args{
				key:  glfw.KeyEnter,
				mods: ModAltGr,
			},
			want: "<M-C-CR>",
		},
		{
			name: "Ctrl + Space",
			args: args{
				key:  glfw.KeySpace,
				mods: ModControl,
			},
			want: "<C-Space>",
		},
		{
			name: "Shift + AltGr + kp9",
			args: args{
				key:  glfw.KeyKP9,
				mods: ModAltGr | ModShift,
			},
			want: "<M-C-S-k9>",
		},
		{
			name: "Ctrl + Alt + h",
			args: args{
				key:  glfw.KeyH,
				mods: ModControl | ModAlt,
			},
			want: "<M-C-h>",
		},
		{
			name: "Ctrl + Shift + Alt + Super + Home",
			args: args{
				key:  glfw.KeyHome,
				mods: ModControl | ModShift | ModAlt | ModSuper,
			},
			want: "<M-C-S-D-Home>", // Yes neoray must send this
		},
		{
			name: "g", // must handled in char callback
			args: args{
				key:  glfw.KeyG,
				mods: 0,
			},
			want: "",
		},
		{
			name: "AltGr + 1", // must handled in char callback
			args: args{
				key:  glfw.Key1,
				mods: ModAltGr,
			},
			want: "",
		},
		{
			name: "Shift + 1", // must handled in char callback
			args: args{
				key:  glfw.Key1,
				mods: ModShift,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseKeyInput(tt.args.key, tt.args.scancode, tt.args.mods); got != tt.want {
				t.Errorf("parseKeyInput() = %v, want %v", got, tt.want)
			}
		})
	}
}
