package sandbox

import "testing"

func TestIsStandardTemplateRecognizesProviderScopedName(t *testing.T) {
	for _, name := range []string{"enterpriserag", "team/enterpriserag", "project-b89e/EnterpriseRag"} {
		if !isStandardTemplate(name) {
			t.Fatalf("expected %q to identify the EnterpriseRag standard template", name)
		}
	}
	if isStandardTemplate("enterpriserag-custom") {
		t.Fatal("custom template must not be treated as the standard template")
	}
	if isStandardTemplate(DesktopTemplateName) {
		t.Fatal("the desktop sibling must not be classified as the CLI standard template")
	}
}

func TestIsDesktopTemplateRecognizesProviderScopedName(t *testing.T) {
	for _, name := range []string{"enterpriserag-desktop", "team/enterpriserag-desktop", "project-b89e/EnterpriseRag-Desktop"} {
		if !isDesktopTemplate(name) {
			t.Fatalf("expected %q to identify the EnterpriseRag desktop template", name)
		}
	}
	if isDesktopTemplate("enterpriserag") {
		t.Fatal("the CLI template must not be classified as desktop")
	}
	if isDesktopTemplate("enterpriserag-desktop-custom") {
		t.Fatal("a similarly prefixed custom name must not be the desktop template")
	}
}

func TestClassifyEnterpriseRagTemplatePrefersNameOverImage(t *testing.T) {
	standard, desktop := classifyEnterpriseRagTemplate(DesktopTemplateName, DefaultDockerImage)
	if standard || !desktop {
		t.Fatalf(
			"named desktop template must be desktop even if the image repo matches CLI, got standard=%v desktop=%v",
			standard, desktop,
		)
	}
	standard, desktop = classifyEnterpriseRagTemplate(StandardTemplateName, DefaultDesktopDockerImage)
	if !standard || desktop {
		t.Fatalf(
			"named CLI template must stay CLI even if the image tag is desktop, got standard=%v desktop=%v",
			standard, desktop,
		)
	}
}

func TestClassifyEnterpriseRagTemplateNamelessImageUsesTag(t *testing.T) {
	standard, desktop := classifyEnterpriseRagTemplate("", DefaultDockerImage)
	if !standard || desktop {
		t.Fatalf("CLI image with no name must be standard, got standard=%v desktop=%v", standard, desktop)
	}
	standard, desktop = classifyEnterpriseRagTemplate("", DefaultDesktopDockerImage)
	if standard || !desktop {
		t.Fatalf("desktop image with no name must be desktop, got standard=%v desktop=%v", standard, desktop)
	}
	standard, desktop = classifyEnterpriseRagTemplate("", DefaultCubeDesktopTemplateImage)
	if standard || !desktop {
		t.Fatalf("Cube desktop image with no name must be desktop, got standard=%v desktop=%v", standard, desktop)
	}
}
