package main

type ModeInfo struct {
	cursor_shape    string
	cell_percentage int
	blinkwait       int
	blinkon         int
	blinkoff        int
	attr_id         int
	attr_id_lm      int
	short_name      string
	name            string
}

type Mode struct {
	cursor_style_enabled bool
	mode_infos           map[string]ModeInfo
	current_mode_name    string
	current_mode         int
}

func CreateMode() Mode {
	return Mode{
		mode_infos: make(map[string]ModeInfo),
	}
}
