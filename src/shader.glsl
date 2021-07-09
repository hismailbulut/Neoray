// Vertex Shader
#version 330 core
layout(location = 0) in vec2 pos;
layout(location = 1) in vec2 tex;
layout(location = 2) in vec4 fg;
layout(location = 3) in vec4 bg;
layout(location = 4) in vec4 sp;
layout(location = 5) in vec2 tex2;

out vec2 texPos;
out vec2 tex2Pos;
out vec4 fgColor;
out vec4 bgColor;
out vec4 spColor;
out vec2 ucPos;

uniform mat4 projection;
uniform vec4 undercurlRect;

vec2 getVertexTexturePos(vec4 rect) {
	switch(gl_VertexID % 4) {
	case 0:
		return vec2(rect.x, rect.y);
	case 1:
		return vec2(rect.x, rect.y + rect.w);
	case 2:
		return vec2(rect.x + rect.z, rect.y + rect.w);
	case 3:
		return vec2(rect.x + rect.z, rect.y);
	}
	return vec2(0);
}

void main() {
	gl_Position =  vec4(pos, 0, 1) * projection;
	texPos = tex;
	tex2Pos = tex2;
	fgColor = fg;
	bgColor = bg;
	spColor = sp;
	ucPos = getVertexTexturePos(undercurlRect);
}

// Fragment Shader
#version 330 core
in vec2 texPos;
in vec2 tex2Pos;
in vec4 fgColor;
in vec4 bgColor;
in vec4 spColor;
in vec2 ucPos;

uniform sampler2D atlas;

// Taked from github.com/couleurs/glsl-sharpen/blob/master/index.glsl
/*
vec4 sharpen(in sampler2D tex, in vec2 coords, in vec2 renderSize) {
	float dx = 1.0 / renderSize.x;
	float dy = 1.0 / renderSize.y;
	vec4 sum = vec4(0.0);
	sum += -0.1 * texture2D(tex, coords + vec2( -1.0 * dx , 0.0 * dy));
	sum += -0.1 * texture2D(tex, coords + vec2( 0.0 * dx , -1.0 * dy));
	sum +=  1.5 * texture2D(tex, coords + vec2( 0.0 * dx , 0.0 * dy));
	sum += -0.1 * texture2D(tex, coords + vec2( 0.0 * dx , 1.0 * dy));
	sum += -0.1 * texture2D(tex, coords + vec2( 1.0 * dx , 0.0 * dy));
	return sum;
}
*/

void main() {
	// Mix colors with special. (For undercurl)
	vec4 ucColor = texture(atlas, ucPos);
	vec4 foreground = mix(fgColor, spColor, min(ucColor.a, spColor.a));
	vec4 background = mix(bgColor, spColor, min(ucColor.a, spColor.a));
	// Mix background and foreground color with textures.
	vec4 tex1Color = /*sharpen(atlas, texPos, vec2(256, 256));*/  texture(atlas, texPos);
	vec4 tex2Color = /*sharpen(atlas, tex2Pos, vec2(256, 256));*/ texture(atlas, tex2Pos);
	vec4 result = mix(background, foreground, max(tex1Color.a, tex2Color.a));
	gl_FragColor = result;
}
