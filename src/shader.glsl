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
layout (triangle_strip, max_vertices = 6) out;

in VS_OUT {
	vec4 tex1pos;
	vec4 tex2pos;
    vec4 ucPos;
    mat4 projection;
    vec4 fgColor;
    vec4 bgColor;
    vec4 spColor;
} gs_in[];

out vec2 fs_tex1pos;
out vec2 fs_tex2pos;
out vec2 fs_ucPos;
out vec4 fs_fgColor;
out vec4 fs_bgColor;
out vec4 fs_spColor;

vec2 pluspos[] = vec2[6](
	vec2(0, 0),
	vec2(0, 1),
	vec2(1, 1),
	vec2(1, 1),
	vec2(1, 0),
	vec2(0, 0)
);

void main() {
	for(int i = 0; i < 6; i++) {
		vec4 pos    = gl_in[0].gl_Position;
		gl_Position = vec4(pos.xy + (pluspos[i] * pos.zw), 0, 1) * gs_in[0].projection;
		fs_tex1pos  = gs_in[0].tex1pos.xy + (pluspos[i] * gs_in[0].tex1pos.zw);
		fs_tex2pos  = gs_in[0].tex2pos.xy + (pluspos[i] * gs_in[0].tex2pos.zw);
		fs_ucPos    = gs_in[0].ucPos.xy   + (pluspos[i] * gs_in[0].ucPos.zw);
		fs_fgColor  = gs_in[0].fgColor;
		fs_bgColor  = gs_in[0].bgColor;
		fs_spColor  = gs_in[0].spColor;
		EmitVertex();
	}
	EndPrimitive();
}

// Fragment Shader
#version 330 core
in vec2 fs_tex1pos;
in vec2 fs_tex2pos;
in vec2 fs_ucPos;
in vec4 fs_fgColor;
in vec4 fs_bgColor;
in vec4 fs_spColor;

uniform sampler2D atlas;

void main() {
	// Mix colors with special. (For undercurl)
	vec4 ucColor    = texture(atlas, fs_ucPos);
	vec4 foreground = mix(fs_fgColor, fs_spColor, min(ucColor.a, fs_spColor.a));
	vec4 background = mix(fs_bgColor, fs_spColor, min(ucColor.a, fs_spColor.a));
	// Mix background and foreground color with textures.
	float maxAlpha = max(texture(atlas, fs_tex1pos).a, texture(atlas, fs_tex2pos).a);
	float texAlpha = maxAlpha; //smoothstep(0.01, 0.99, maxAlpha); // Changing values changes text sharpness.
	// if(texAlpha < 0.2) {
	//     foreground.r = 0;
	//     foreground.g = 0;
	// } else if(texAlpha < 0.4) {
	//     foreground.r = 0;
	// }
	vec4 result  = mix(background, foreground, texAlpha);
	gl_FragColor = result;
}
