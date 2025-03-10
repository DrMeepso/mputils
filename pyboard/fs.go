package pyboard

import (
	"encoding/hex"
	"strconv"
	"strings"
)

type PyFileSystem struct {
	pyboard    *Pyboard
	CurrentDir string
}

func NewPyFileSystem() *PyFileSystem {
	return &PyFileSystem{
		CurrentDir: "/",
	}
}

func (fs *PyFileSystem) ListDir() []string {
	files, err := fs.pyboard.Exec("import os; print(','.join(os.listdir()))")
	if err {
		return []string{}
	}
	return strings.Split(files, ",")
}

func (fs *PyFileSystem) ChangeDir(dir string) {
	fs.pyboard.Exec("import os; os.chdir('" + dir + "')")
	fs.CurrentDir = dir
}

// read the file in mutiple chunks
func (fs *PyFileSystem) readFileChunked(filename string, chunkSize int) string {
	python := `
import os
import binascii

fileHex = ""
with open("` + filename + `", 'rb') as f:
	while True:
		data = f.read( ` + strconv.Itoa(chunkSize) + ` )
		if not data:
			break
		fileHex += binascii.hexlify(data).decode('utf-8')
print(fileHex + "#")
`

	fileContent, Rerr := fs.pyboard.Exec(python)
	if Rerr {
		println("Error reading file")
		println(fileContent)
		return ""
	}

	// decode the hex string
	proper, err := hex.DecodeString(fileContent[:len(fileContent)-1]) // remove the last #
	if err != nil {
		println(err.Error())
		return ""
	}

	return string(proper)
}

func (fs *PyFileSystem) ReadFile(filename string) string {
	return fs.readFileChunked(filename, 1024)
}

func writeChunk(filename string, chunk string, fileSystem PyFileSystem) bool {

	hexChunk := hex.EncodeToString([]byte(chunk))

	python := "import os\n\r"
	python += "import binascii\n\r"
	python += "with open('" + filename + "', 'ab') as f:\n\r"
	python += "    f.write(binascii.unhexlify('" + hexChunk + "'))\n\r"
	ret, err := fileSystem.pyboard.Exec(python)
	println(ret)
	if err {
		return false
	}
	return true
}

// write the file in mutiple chunks
func (fs *PyFileSystem) writeFileChunked(filename string, data string, chunkSize int) bool {
	fileData := []byte(data)
	chunkCount := len(fileData) / chunkSize

	// clear the file
	fs.pyboard.Exec("open('" + filename + "', 'w').close()")

	for i := 0; i < chunkCount; i++ {
		chunk := fileData[i*chunkSize : (i+1)*chunkSize]
		writeChunk(filename, string(chunk), *fs)
	}

	// write the last chunk
	if len(fileData)%chunkSize != 0 {
		chunk := fileData[chunkCount*chunkSize:]
		writeChunk(filename, string(chunk), *fs)
	}

	return true
}

func (fs *PyFileSystem) WriteFile(filename string, data string) bool {
	return fs.writeFileChunked(filename, data, 1024)
}

func (fs *PyFileSystem) RemoveFile(filename string) {
	fs.pyboard.Exec("import os; os.remove('" + filename + "')")
}

func (fs *PyFileSystem) MakeDir(dir string) {
	fs.pyboard.Exec("import os; os.mkdir('" + dir + "')")
}

func (fs *PyFileSystem) RemoveDir(dir string) {
	fs.pyboard.Exec("import os; os.rmdir('" + dir + "')")
}

func (fs *PyFileSystem) GetSize(filename string) int {
	size, _ := fs.pyboard.Exec("import os; print(os.stat('" + filename + "').st_size)")
	fileSize, err := strconv.Atoi(size)
	if err != nil {
		return -1
	}
	return fileSize
}

func (fs *PyFileSystem) Exists(filename string) bool {
	ret, _ := fs.pyboard.Exec("import os; print(os.path.exists('" + filename + "'))")
	return ret == "True"
}

func (fs *PyFileSystem) IsDir(filename string) bool {
	ret, _ := fs.pyboard.Exec("import os; print(os.path.isdir('" + filename + "'))")
	return ret == "True"
}

func (fs *PyFileSystem) GetSHA256(filename string) string {

	python := "import hashlib\n\r"
	python += "import binascii\n\r"
	python += "hash = hashlib.sha256()\n\r"
	python += "with open('" + filename + "', 'rb') as f:\n\r"
	python += "    while True:\n\r"
	python += "        data = f.read(256)\n\r"
	python += "        if not data:\n\r"
	python += "            break\n\r"
	python += "        hash.update(data)\n\r"
	python += "print(binascii.hexlify(hash.digest()).decode('utf-8'))\n\r"

	hash, err := fs.pyboard.Exec(python)
	if err {
		println(hash)
		return ""
	}
	return hash
}
