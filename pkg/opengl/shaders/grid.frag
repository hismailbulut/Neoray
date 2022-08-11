#version 330 core
layout(location = 0) out vec4 outFragColor;

in GS_OUT {
	vec2 tex1pos;
	vec2 tex2pos;
	vec2 ucPos;
	vec4 fgColor;
	vec4 bgColor;
	vec4 spColor;
} fs_in;

uniform sampler2D atlas;

void main() {
	vec4 tex1    = texture(atlas, fs_in.tex1pos);
	vec4 fg      = mix(tex1, fs_in.fgColor, fs_in.fgColor.a);           // Use texture color if fg.A < 1
	float texA   = max(tex1.a, texture(atlas, fs_in.tex2pos).a);        // Use both of textures
	float ucA    = min(texture(atlas, fs_in.ucPos).a, fs_in.spColor.a); // If sp.A == 0 we don't draw undercurl
	vec4 result  = mix(fs_in.bgColor, fg, texA);                        // Draw foreground over background
	outFragColor = mix(result, fs_in.spColor, ucA);                     // Draw special over result color
}
