package services

import (
	"strings"
	"testing"

	"ltseries/backend/internal/models"
)

func TestAnalysisSystemPromptRequiresChineseOutput(t *testing.T) {
	prompt := analysisSystemPrompt()
	for _, required := range []string{
		"所有面向用户显示的字段必须使用中文",
		"不要把角色描述、镜头动作、镜头语言写成英文",
		"分镜要覆盖完整剧情，而不是只生成一个镜头",
		"短剧至少拆成 4 个镜头",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("analysis prompt missing %q: %s", required, prompt)
		}
	}
}

func TestNormalizeAnalysisResolvesShotEntityNamesToIDs(t *testing.T) {
	analysis := &models.ScriptAnalysis{
		Characters: []models.Entity{{ID: "char_1", Name: "林遥"}},
		Scenes:     []models.Entity{{ID: "scene_1", Name: "城市天台"}},
		Shots: []models.StoryboardShot{
			{SceneID: "城市天台", CharacterIDs: []string{"林遥"}},
		},
	}

	normalizeAnalysis(analysis)

	if analysis.Shots[0].SceneID != "scene_1" {
		t.Fatalf("scene_id = %q, want scene_1", analysis.Shots[0].SceneID)
	}
	if len(analysis.Shots[0].CharacterIDs) != 1 || analysis.Shots[0].CharacterIDs[0] != "char_1" {
		t.Fatalf("character_ids = %#v, want [char_1]", analysis.Shots[0].CharacterIDs)
	}
}

func TestFrameReferenceURLsUseAssetsAndNameReferences(t *testing.T) {
	project := &models.Project{
		Characters: []models.Entity{{
			ID:   "char_1",
			Name: "林遥",
			FullBody: models.AssetUnit{
				SelectedImageID: "full",
				ImageVariants:   []models.ImageVariant{{ID: "full", URL: "https://cdn.example.com/full.png"}},
			},
			ThreeViews: models.AssetUnit{
				SelectedImageID: "three",
				ImageVariants:   []models.ImageVariant{{ID: "three", URL: "https://cdn.example.com/three.png"}},
			},
			HeadShot: models.AssetUnit{
				SelectedImageID: "head",
				ImageVariants:   []models.ImageVariant{{ID: "head", URL: "https://cdn.example.com/head.png"}},
			},
		}},
		Scenes: []models.Entity{{
			ID:    "scene_1",
			Name:  "城市天台",
			Image: models.ImageAsset{SelectedID: "scene", Variants: []models.ImageVariant{{ID: "scene", URL: "https://cdn.example.com/scene.png"}}},
		}},
		Props: []models.Entity{{
			ID:    "prop_1",
			Name:  "发光信件",
			Image: models.ImageAsset{SelectedID: "prop", Variants: []models.ImageVariant{{ID: "prop", URL: "https://cdn.example.com/prop.png"}}},
		}},
	}
	frame := models.StoryboardShot{
		SceneID:      "城市天台",
		CharacterIDs: []string{"林遥"},
		Action:       "林遥伸手触碰发光信件",
	}

	urls := frameReferenceURLs(project, frame, "https://cdn.example.com/previous.png")
	joined := strings.Join(urls, "\n")
	for _, want := range []string{"previous.png", "full.png", "three.png", "head.png", "scene.png", "prop.png"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("references = %#v, missing %s", urls, want)
		}
	}
}
