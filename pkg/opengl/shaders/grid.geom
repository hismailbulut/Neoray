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
