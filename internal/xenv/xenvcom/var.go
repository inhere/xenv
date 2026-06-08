package xenvcom

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/x/ccolor"
)

var sessionID = os.Getenv(SessIdEnvName)

// SessionID 获取当前会话ID
func SessionID() string {
	if sessionID != "" {
		return sessionID
	}

	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	return SessionIDForDir(SessionRootDir(workDir))
}

// SessionRootDir returns the project root used to scope session state.
func SessionRootDir(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	dir = filepath.Clean(dir)

	markers := []string{
		LocalOverrideStateFile,
		LocalStateFile,
		"go.work",
		"go.mod",
		".tool-versions",
		"package.json",
		".nvmrc",
		".python-version",
		".git",
	}
	for {
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Clean(start)
		}
		dir = parent
	}
}

// SessionIDForDir returns a stable session ID for the given directory.
func SessionIDForDir(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	cleanDir := filepath.Clean(absDir)
	baseName := sanitizeSessionIDPart(filepath.Base(cleanDir))
	if baseName == "" {
		baseName = "root"
	}

	hashSource := filepath.ToSlash(cleanDir)
	hash := md5.Sum([]byte(strings.ToLower(hashSource)))
	return baseName + "_" + hex.EncodeToString(hash[:])[:10]
}

func sanitizeSessionIDPart(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}

// SessionFile 获取当前会话状态文件
func SessionFile() string {
	return SessionStateDir + "/" + SessionID() + ".json"
}

// SetSessionID 设置当前会话ID (用于测试)
func SetSessionID(id string) {
	sessionID = id
}

// DebugMode debug mode flag
var DebugMode = envutil.GetBool(XenvDebugEnvName, false)

// Debugf prints debug messages
func Debugf(format string, args ...any) {
	if DebugMode {
		ccolor.Printf("<cyan>DEBUG</>: "+format, args...)
	}
}

func Debugln(args ...any) {
	if DebugMode {
		ccolor.Println("<cyan>DEBUG</>: ", fmt.Sprint(args...))
	}
}
