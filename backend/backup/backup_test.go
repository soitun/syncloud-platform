package backup

import (
	"github.com/syncloud/platform/cli"
	"github.com/syncloud/platform/log"
	"github.com/syncloud/platform/snap/model"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/assert"
	"testing"
)

type DiskUsageStub struct {
	used uint64
}

func (e *DiskUsageStub) Used(_ string) (uint64, error) {
	return e.used, nil
}

type SnapServiceStub struct {
}

func (s *SnapServiceStub) Stop(_ string) error {
	return nil
}

func (s *SnapServiceStub) Start(_ string) error {
	return nil
}

func (s *SnapServiceStub) RunCmdIfExists(_ model.Snap, _ string) error {
	return nil
}

type SnapInfoStub struct {
}

func (s *SnapInfoStub) Snap(_ string) (model.Snap, error) {
	return model.Snap{}, nil
}

func TestRemove(t *testing.T) {
	logger := log.Default()

	backupDir := createTempDir("backup")
	varDir := createTempDir("var")
	defer os.Remove(backupDir)
	defer os.Remove(varDir)
	tmpfn := filepath.Join(backupDir, "tmpfile")
	if err := ioutil.WriteFile(tmpfn, []byte(""), 0666); err != nil {
		panic(err)
	}
	backup := New(backupDir, varDir, cli.New(log.Default()), &DiskUsageStub{100}, &SnapServiceStub{}, &SnapInfoStub{}, logger)
	err := backup.Remove("tmpfile")
	assert.Nil(t, err)
	list, err := backup.List()
	assert.Nil(t, err)
	assert.Equal(t, len(list), 0)
}

func TestBackup(t *testing.T) {
	backupDir := createTempDir("backup")
	varDir := createTempDir("var")
	defer os.Remove(backupDir)
	defer os.Remove(varDir)
	appDir := filepath.Join(varDir, "test-app")
	_ = os.Mkdir(appDir, 0750)
	versionDir := filepath.Join(appDir, "x1")
	_ = os.Mkdir(versionDir, 0750)
	currentDir := filepath.Join(appDir, "current")
	_ = os.Symlink(versionDir, currentDir)
	commonDir := filepath.Join(appDir, "common")
	_ = os.Mkdir(commonDir, 0750)

	currentFile := filepath.Join(versionDir, "current.file")
	if err := ioutil.WriteFile(currentFile, []byte("current"), 0666); err != nil {
		panic(err)
	}

	commonFile := filepath.Join(commonDir, "common.file")
	if err := ioutil.WriteFile(commonFile, []byte("common"), 0666); err != nil {
		panic(err)
	}

	diskusage := &DiskUsageStub{100}
	logger := log.Default()
	shellExecutor := cli.New(logger)
	backup := New(backupDir+"/non-existent", varDir, shellExecutor, diskusage, &SnapServiceStub{}, &SnapInfoStub{}, logger)
	backup.Init()
	err := backup.Create("test-app")
	assert.Nil(t, err)
	backups, err := backup.List()
	assert.Nil(t, err)
	assert.Equal(t, len(backups), 1)

	err = os.Remove(currentFile)
	assert.Nil(t, err)

	err = os.Remove(commonFile)
	assert.Nil(t, err)

	err = backup.Restore(backups[0].File)
	assert.Nil(t, err)
	currentFileContent, err := ioutil.ReadFile(currentFile)
	assert.Nil(t, err)
	assert.Equal(t, "current", string(currentFileContent))
	commonFileContent, err := ioutil.ReadFile(commonFile)
	assert.Nil(t, err)
	assert.Equal(t, "common", string(commonFileContent))

}

func createTempDir(pattern string) string {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		panic(err)
	}
	return dir
}
