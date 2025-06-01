package goripgrep

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// OptimizedFileWalker provides fast directory traversal with smart filtering
type OptimizedFileWalker struct {
	textExtensions   map[string]bool
	binaryExtensions map[string]bool
	skipDirs         map[string]bool
}

// NewOptimizedFileWalker creates a new optimized file walker
func NewOptimizedFileWalker() *OptimizedFileWalker {
	return &OptimizedFileWalker{
		textExtensions: map[string]bool{
			// Programming languages
			".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
			".c": true, ".cpp": true, ".cxx": true, ".cc": true, ".h": true, ".hpp": true,
			".rs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
			".scala": true, ".clj": true, ".hs": true, ".ml": true, ".fs": true,
			".vb": true, ".cs": true, ".pas": true, ".pl": true, ".pm": true,
			".lua": true, ".r": true, ".m": true, ".asm": true, ".s": true,

			// Web technologies
			".html": true, ".htm": true, ".xhtml": true, ".xml": true, ".xsl": true,
			".css": true, ".scss": true, ".sass": true, ".less": true,
			".json": true, ".yaml": true, ".yml": true, ".toml": true,
			".vue": true, ".jsx": true, ".tsx": true, ".svelte": true,

			// Documentation and text
			".txt": true, ".md": true, ".markdown": true, ".rst": true,
			".tex": true, ".ltx": true, ".org": true, ".adoc": true,
			".rtf": true, ".man": true, ".1": true, ".2": true, ".3": true,

			// Configuration and data
			".cfg": true, ".conf": true, ".config": true, ".ini": true,
			".env": true, ".properties": true, ".plist": true,
			".csv": true, ".tsv": true, ".log": true, ".sql": true,

			// Build and project files
			".mk": true, ".makefile": true, ".cmake": true, ".gradle": true,
			".sbt": true, ".cabal": true, ".gemspec": true, ".podspec": true,
			".dockerfile": true, ".dockerignore": true, ".gitignore": true,

			// Scripts and shells
			".sh": true, ".bash": true, ".zsh": true, ".fish": true,
			".ps1": true, ".psm1": true, ".bat": true, ".cmd": true,
		},

		binaryExtensions: map[string]bool{
			// Images
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
			".tiff": true, ".tif": true, ".webp": true, ".ico": true, ".svg": true,
			".psd": true, ".ai": true, ".eps": true, ".raw": true, ".cr2": true,

			// Videos
			".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
			".mkv": true, ".webm": true, ".m4v": true, ".3gp": true, ".mpg": true,
			".mpeg": true, ".ogv": true,

			// Audio
			".mp3": true, ".wav": true, ".flac": true, ".aac": true, ".ogg": true,
			".wma": true, ".m4a": true, ".opus": true, ".aiff": true,

			// Archives
			".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
			".rar": true, ".7z": true, ".dmg": true, ".iso": true, ".deb": true,
			".rpm": true, ".msi": true, ".pkg": true,

			// Executables and libraries
			".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true,
			".lib": true, ".o": true, ".obj": true, ".bin": true, ".class": true,
			".jar": true, ".war": true, ".ear": true, ".pyc": true, ".pyo": true,

			// Documents (binary formats)
			".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
			".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,

			// Fonts
			".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,

			// Database files
			".db": true, ".sqlite": true, ".sqlite3": true, ".mdb": true,
		},

		skipDirs: map[string]bool{
			// Version control
			".git": true, ".svn": true, ".hg": true, ".bzr": true,

			// Build outputs
			"node_modules": true, "target": true, "build": true, "dist": true,
			"out": true, "bin": true, "obj": true, ".gradle": true,

			// IDE and editor files
			".vscode": true, ".idea": true, ".vs": true, ".vscode-test": true,

			// OS specific
			".DS_Store": true, "Thumbs.db": true, "__pycache__": true,
			".pytest_cache": true, ".coverage": true,

			// Temporary
			"tmp": true, "temp": true, ".tmp": true, ".cache": true,
		},
	}
}

// WalkFiles walks the directory tree efficiently with early filtering
func (w *OptimizedFileWalker) WalkFiles(root string, fn func(path string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that we know contain no useful files
		if d.IsDir() {
			if w.shouldSkipDirectory(d.Name(), path, root) {
				return filepath.SkipDir
			}
			return nil
		}

		// Fast extension-based filtering
		if !w.isLikelyTextFile(path) {
			return nil
		}

		// Additional checks for files without clear extensions
		if filepath.Ext(path) == "" {
			if w.isBinaryFileByContent(path) {
				return nil
			}
		}

		return fn(path)
	})
}

// isLikelyTextFile performs fast extension-based filtering
func (w *OptimizedFileWalker) isLikelyTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// If we know it's binary, skip immediately
	if w.binaryExtensions[ext] {
		return false
	}

	// If we know it's text, include it
	if w.textExtensions[ext] {
		return true
	}

	// Files without extensions might be text (e.g., Makefile, README)
	if ext == "" {
		name := strings.ToLower(filepath.Base(filePath))
		// Common text files without extensions
		textFiles := map[string]bool{
			"makefile": true, "dockerfile": true, "readme": true,
			"changelog": true, "license": true, "authors": true,
			"contributors": true, "copying": true, "install": true,
			"news": true, "todo": true, "version": true,
		}
		return textFiles[name]
	}

	// Unknown extensions - let the binary detection decide
	return true
}

// isBinaryFileByContent checks file content for binary indicators
func (w *OptimizedFileWalker) isBinaryFileByContent(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return true // If we can't read it, treat as binary
	}
	defer file.Close()

	// Read first 512 bytes (same as Git's binary detection)
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return true
	}

	// Check for null bytes (strong binary indicator)
	nullCount := 0
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			nullCount++
		}
	}

	// If more than 0.1% are null bytes, consider it binary
	if n > 0 && float64(nullCount)/float64(n) > 0.001 {
		return true
	}

	// Check for high proportion of non-printable characters
	nonPrintable := 0
	for i := 0; i < n; i++ {
		b := buffer[i]
		// Count non-printable characters (excluding common whitespace)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
		if b > 126 {
			nonPrintable++
		}
	}

	// If more than 5% are non-printable, likely binary
	if n > 0 && float64(nonPrintable)/float64(n) > 0.05 {
		return true
	}

	return false
}

// shouldSkipDirectory determines if a directory should be skipped
func (w *OptimizedFileWalker) shouldSkipDirectory(dirName, fullPath, rootPath string) bool {
	// Always skip known uninteresting directories
	if w.skipDirs[dirName] {
		return true
	}

	// Skip hidden directories (except at root level)
	if strings.HasPrefix(dirName, ".") && fullPath != rootPath {
		// But allow some important hidden dirs
		allowedHidden := map[string]bool{
			".github": true,
			".gitlab": true,
		}
		return !allowedHidden[dirName]
	}

	return false
}

// GetFileCount estimates the number of files that would be processed
func (w *OptimizedFileWalker) GetFileCount(root string) (int, error) {
	count := 0
	err := w.WalkFiles(root, func(path string) error {
		count++
		return nil
	})
	return count, err
}
