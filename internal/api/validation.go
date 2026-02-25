package api

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MaxAttachmentSizeBytes is the maximum allowed attachment size (10 MB).
const MaxAttachmentSizeBytes = 10 * 1024 * 1024

// blockedExtensions mirrors the backend BLOCKED_EXTENSIONS list from constants.py.
var blockedExtensions = map[string]bool{
	// Windows executables and libraries
	".exe": true, ".dll": true, ".scr": true, ".com": true, ".pif": true, ".cpl": true, ".sys": true,
	// Windows scripts
	".bat": true, ".cmd": true, ".ps1": true, ".vbs": true, ".vbe": true,
	".wsf": true, ".wsh": true, ".wsc": true, ".sct": true, ".shb": true,
	// Windows installers and packages
	".msi": true, ".msp": true, ".mst": true, ".cab": true,
	// Windows management and help
	".msc": true, ".inf": true, ".reg": true, ".rgs": true, ".ins": true, ".isp": true, ".chm": true, ".hta": true,
	// Shortcuts and links
	".lnk": true, ".url": true, ".scf": true,
	// Disk images and containers
	".iso": true, ".img": true, ".vhd": true, ".vhdx": true, ".dmg": true,
	// Mobile and cross-platform executables
	".apk": true, ".app": true, ".action": true, ".jar": true,
	// Access database extensions
	".ade": true, ".adp": true, ".mde": true,
}

// ValidateAttachment checks if a file is safe to upload as an attachment.
// Returns an error if the extension is blocked or the file is too large.
func ValidateAttachment(filename string, size int64) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" && blockedExtensions[ext] {
		return fmt.Errorf("file extension %q is not allowed", ext)
	}
	if size > MaxAttachmentSizeBytes {
		return fmt.Errorf("file size %d bytes exceeds maximum of %d bytes (10 MB)", size, MaxAttachmentSizeBytes)
	}
	return nil
}
