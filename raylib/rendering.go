package raylib

type RenderingFeature string

const (
	BeginDrawing     RenderingFeature = "begin-drawing"
	EndDrawing       RenderingFeature = "end-drawing"
	ClearBackground  RenderingFeature = "clear-background"
	DrawText         RenderingFeature = "draw-text"
	DrawShapes       RenderingFeature = "draw-shapes"
	DrawTextures     RenderingFeature = "draw-textures"
	PassColorValues  RenderingFeature = "pass-color-values"
	RenderLoopSafety RenderingFeature = "render-loop-safety"
)

func RenderingFeatures() []RenderingFeature {
	return []RenderingFeature{
		BeginDrawing,
		EndDrawing,
		ClearBackground,
		DrawText,
		DrawShapes,
		DrawTextures,
		PassColorValues,
		RenderLoopSafety,
	}
}
