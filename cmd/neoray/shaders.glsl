// NOTE: These tags are used for specifying shader type, do not delete or change them!
// If this file name has changed, update filename in the renderer_gl (above the shader_sources)
// Vertex Shader
#version 330 core
layout(location = 0) in vec2 pos;
layout(location = 1) in vec2 tex;
layout(location = 2) in vec4 fg;
layout(location = 3) in vec4 bg;

out vec2 texCoord;
out vec4 fgColor;
out vec4 bgColor;

uniform mat4 projection;

void main() {
	gl_Position =  vec4(pos, 0, 1) * projection;
	texCoord = tex;
	fgColor = fg;
	bgColor = bg;
}

// Fragment Shader
#version 330 core

in vec2 texCoord;
in vec4 fgColor;
in vec4 bgColor;

uniform sampler2D atlas;

void main() {
	vec4 texColor = texture2D(atlas, texCoord);
	gl_FragColor = mix(bgColor, fgColor, texColor.a);
}
