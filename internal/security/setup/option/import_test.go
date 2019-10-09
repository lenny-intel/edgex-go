//
// Copyright (c) 2019 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under
// the License.
//
// SPDX-License-Identifier: Apache-2.0'
//

package option

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgexfoundry/edgex-go/internal/security/setup"
	"github.com/stretchr/testify/assert"
)

func TestImportCacheDirEmpty(t *testing.T) {
	// this test case is for import running first
	// and PKI_CACHE_DIR is empty- should expect an error by design

	tearDown := setupImportTest(t)
	defer tearDown()

	options := PkiInitOption{
		ImportOpt: true,
	}
	importOn, _, _ := NewPkiInitOption(options)
	importOn.setExecutor(testExecutor)

	var exitStatus exitCode
	var err error
	f := Import()
	exitStatus, err = f(importOn.(*PkiInitOption))

	deployEmpty, emptyErr := isDirEmpty(pkiInitDeployDir)

	assert := assert.New(t)
	assert.Equal(exitWithError, exitStatus)
	assert.NotNil(err)
	assert.Nil(emptyErr)
	assert.True(deployEmpty)
}

func TestImportFromPKICache(t *testing.T) {
	// this test case is for import pre-populated cached PKI
	// files from PKI_CACHE dir

	tearDown := setupImportTest(t)
	defer tearDown()

	// put some test file into the cache dir first
	writeTestFileToCacheDir(t)

	options := PkiInitOption{
		ImportOpt: true,
	}
	importOn, _, _ := NewPkiInitOption(options)
	importOn.setExecutor(testExecutor)

	f := Import()

	exitStatus, err := f(importOn.(*PkiInitOption))

	deployEmpty, emptyErr := isDirEmpty(pkiInitDeployDir)

	assert := assert.New(t)
	assert.Equal(normal, exitStatus)
	assert.Nil(err)
	assert.Nil(emptyErr)
	assert.False(deployEmpty)
}

func TestEmptyPkiCacheEnvironment(t *testing.T) {
	options := PkiInitOption{
		ImportOpt: true,
	}
	importOn, _, _ := NewPkiInitOption(options)
	importOn.setExecutor(testExecutor)
	exitCode, err := importOn.executeOptions(Import())

	// when PKI_CACHE env is empty, it leads to non-existing dir
	// and should be an error case
	assert := assert.New(t)
	assert.NotNil(err)
	assert.Equal(exitWithError, exitCode)
}

func TestImportOff(t *testing.T) {
	tearDown := setupImportTest(t)
	defer tearDown()

	options := PkiInitOption{
		ImportOpt: false,
	}
	importOff, _, _ := NewPkiInitOption(options)
	importOff.setExecutor(testExecutor)
	exitCode, err := importOff.executeOptions(Import())

	assert := assert.New(t)
	assert.Equal(normal, exitCode)
	assert.Nil(err)
}

func TestIsDirEmpty(t *testing.T) {
	assert := assert.New(t)
	_, err := isDirEmpty("/non/existing/dir/")

	assert.NotNil(err)

	// put some test file into the current dir to trigger event
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get the working dir %s: %v", curDir, err)
	}
	empty, err := isDirEmpty(curDir)
	assert.Nil(err)
	assert.False(empty)

	// create an empty temp dir
	tempDir, err := ioutil.TempDir(curDir, "test")
	if err != nil {
		t.Fatalf("cannot create the temporary dir %s: %v", tempDir, err)
	}
	empty, err = isDirEmpty(tempDir)
	defer func() {
		// remove tempDir:
		os.RemoveAll(tempDir)
	}()

	assert.Nil(err)
	assert.True(empty)
}

func setupImportTest(t *testing.T) func() {
	testExecutor = &mockOptionsExecutor{}
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get the working dir %s: %v", curDir, err)
	}

	testResourceDir := filepath.Join(curDir, resourceDirName)
	if err := createDirectoryIfNotExists(testResourceDir); err != nil {
		t.Fatalf("cannot create resource dir %s for the test: %v", testResourceDir, err)
	}

	resDir := filepath.Join(curDir, "testdata", resourceDirName)

	testResTomlFile := filepath.Join(testResourceDir, configTomlFile)
	if _, err := copyFile(filepath.Join(resDir, configTomlFile), testResTomlFile); err != nil {
		t.Fatalf("cannot copy %s for the test: %v", configTomlFile, err)
	}

	_ = setup.Init()

	origEnvXdgRuntimeDir := os.Getenv(envXdgRuntimeDir)
	// change it to the current working directory
	os.Setenv(envXdgRuntimeDir, curDir)

	origEnvPkiCache := os.Getenv(envPkiCache)
	// use curDir/cache as the working directory for test
	pkiCacheDir := filepath.Join(curDir, "cache")
	os.Setenv(envPkiCache, pkiCacheDir)
	if err := createDirectoryIfNotExists(pkiCacheDir); err != nil {
		t.Fatalf("cannot create the PKI_CACHE dir %s: %v", pkiCacheDir, err)
	}

	origDeployDir := pkiInitDeployDir
	tempDir, tempDirErr := ioutil.TempDir(curDir, "deploytest")
	if tempDirErr != nil {
		t.Fatalf("cannot create temporary scratch directory for the test: %v", tempDirErr)
	}
	pkiInitDeployDir = tempDir

	return func() {
		// cleanup
		os.Setenv(envXdgRuntimeDir, origEnvXdgRuntimeDir)
		os.Setenv(envPkiCache, origEnvPkiCache)
		os.RemoveAll(pkiInitDeployDir)
		os.RemoveAll(pkiCacheDir)
		os.RemoveAll(testResourceDir)
		pkiInitDeployDir = origDeployDir
	}
}

func writeTestFileToCacheDir(t *testing.T) {
	pkiCacheDir := os.Getenv(envPkiCache)
	// make a test dir
	testFileDir := filepath.Join(pkiCacheDir, "test", caServiceName)
	_ = createDirectoryIfNotExists(testFileDir)
	testFile := filepath.Join(testFileDir, "testFile")
	testData := []byte("test data\n")
	if err := ioutil.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("cannot write testData to direcotry %s: %v", pkiCacheDir, err)
	}
}