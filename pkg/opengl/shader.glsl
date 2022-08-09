// Tags and order is important.

// Vertex Shader
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

// Geometry Shader
#version 330 core
layout (points) in;
layout (triangle_strip, max_vertices = 4) out;

in VS_OUT {
	vec4 tex1pos;
	vec4 tex2pos;
	vec4 ucPos;
	mat4 projection;
	vec4 fgColor;
	vec4 bgColor;
	vec4 spColor;
} gs_in[];

out GS_OUT {
	vec2 tex1pos;
	vec2 tex2pos;
	vec2 ucPos;
	vec4 fgColor;
	vec4 bgColor;
	vec4 spColor;
} gs_out;

vec2 pluspos[] = vec2[4](
	vec2(0, 1),
	vec2(0, 0),
	vec2(1, 1),
	vec2(1, 0)
);

void main() {
	for(int i = 0; i < 4; i++) {
		vec4 pos       = gl_in[0].gl_Position;
		gl_Position    = vec4(pos.xy + (pluspos[i] * pos.zw), 0, 1) * gs_in[0].projection;
		gs_out.tex1pos = gs_in[0].tex1pos.xy + (pluspos[i] * gs_in[0].tex1pos.zw);
		gs_out.tex2pos = gs_in[0].tex2pos.xy + (pluspos[i] * gs_in[0].tex2pos.zw);
		gs_out.ucPos   = gs_in[0].ucPos.xy + (pluspos[i] * gs_in[0].ucPos.zw);
		gs_out.fgColor = gs_in[0].fgColor;
		gs_out.bgColor = gs_in[0].bgColor;
		gs_out.spColor = gs_in[0].spColor;
		EmitVertex();
	}
	EndPrimitive();
}

// Fragment Shader
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
