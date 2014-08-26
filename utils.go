package flotilla

import "path/filepath"

func dirAdd(dir string, envDirs []string) []string {
	adddir := dirAbs(dir)
	if dirAppendable(adddir, envDirs) {
		envDirs = append(envDirs, adddir)
	}
	return envDirs
}

func dirAbs(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	} else {
		return filepath.Join(workingPath, dir)
	}
}

func dirAppendable(dir string, envDirs []string) bool {
	for _, d := range envDirs {
		if d == dir {
			return false
		}
	}
	return true
}
