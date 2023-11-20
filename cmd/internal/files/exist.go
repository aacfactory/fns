package files

import "os"

func ExistFile(path string) (ok bool) {
	_, err := os.Stat(path)
	if err == nil {
		ok = true
		return
	}
	if os.IsNotExist(err) {
		return
	}
	ok = true
	return
}
