package api

import "testing"

func TestValidateAttachment_BlockedExtensions(t *testing.T) {
	blocked := []string{
		"malware.exe", "script.bat", "installer.msi", "disk.iso",
		"app.apk", "shortcut.lnk", "code.dll", "image.dmg",
	}
	for _, name := range blocked {
		if err := ValidateAttachment(name, 1024); err == nil {
			t.Errorf("ValidateAttachment(%q, 1024) = nil, want error for blocked extension", name)
		}
	}
}

func TestValidateAttachment_AllowedExtensions(t *testing.T) {
	allowed := []string{
		"document.pdf", "photo.png", "code.py", "script.sh",
		"script.js", "readme.md", "data.csv", "archive.zip",
	}
	for _, name := range allowed {
		if err := ValidateAttachment(name, 1024); err != nil {
			t.Errorf("ValidateAttachment(%q, 1024) = %v, want nil for allowed extension", name, err)
		}
	}
}

func TestValidateAttachment_CaseInsensitive(t *testing.T) {
	if err := ValidateAttachment("MALWARE.EXE", 1024); err == nil {
		t.Error("ValidateAttachment(\"MALWARE.EXE\", 1024) = nil, want error (case-insensitive)")
	}
	if err := ValidateAttachment("script.Bat", 1024); err == nil {
		t.Error("ValidateAttachment(\"script.Bat\", 1024) = nil, want error (case-insensitive)")
	}
}

func TestValidateAttachment_SizeLimit(t *testing.T) {
	// Exactly at limit — should pass
	if err := ValidateAttachment("doc.pdf", MaxAttachmentSizeBytes); err != nil {
		t.Errorf("ValidateAttachment at exact limit: %v", err)
	}

	// Over limit — should fail
	if err := ValidateAttachment("doc.pdf", MaxAttachmentSizeBytes+1); err == nil {
		t.Error("ValidateAttachment over limit = nil, want error")
	}
}

func TestValidateAttachment_NoExtension(t *testing.T) {
	if err := ValidateAttachment("Makefile", 1024); err != nil {
		t.Errorf("ValidateAttachment(\"Makefile\", 1024) = %v, want nil", err)
	}
}
