#version 330 core
layout(location = 0) in vec4 pos;
layout(location = 1) in vec4 tex1;
layout(location = 2) in vec4 tex2;
layout(location = 3) in vec4 fg;
layout(location = 4) in vec4 bg;
layout(location = 5) in vec4 sp;

uniform mat4 projection;
uniform vec4 undercurlRect;

out VS_OUT {
	vec4 tex1pos;
	vec4 tex2pos;
	vec4 ucPos;
	mat4 projection;
	vec4 fgColor;
	vec4 bgColor;
	vec4 spColor;
} vs_out;

void main() {
	gl_Position       = pos;
	vs_out.tex1pos    = tex1;
	vs_out.tex2pos    = tex2;
	vs_out.ucPos      = undercurlRect;
	vs_out.projection = projection;
	vs_out.fgColor    = fg;
	vs_out.bgColor    = bg;
	vs_out.spColor    = sp;
}
