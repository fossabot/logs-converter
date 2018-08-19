package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func testConfigCreate(configPath string, logsFilesListJSON string, logLevel string, mongoURL string, mongoDB string, mongoCollection string, dropDB bool) error {
	fmt.Println("Helper func in action")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		log.Fatalf("failed creating all dirs for config file %v", filepath.Dir(configPath), err)

	}
	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalf("error opening config file: %v", err)
	}

	if err != nil {
		return fmt.Errorf("Failed to create test config: %v", err)
	}
	_, err = configFile.WriteString(fmt.Sprintf(`LogLevel="%s"
	LogsFilesListJSON='%s'
	MongoURL="%s"
	MongoDB="%s"
	MongoCollection="%s"
	DropDB=%t`, logLevel, logsFilesListJSON, mongoURL, mongoDB, mongoCollection, dropDB))
	if err != nil {
		return fmt.Errorf("Failed to write test config: %v", err)
	}
	if err := configFile.Close(); err != nil {
		return fmt.Errorf("failed to close test config file descriptor: %v", err)
	}

	return nil
}

func TestLoadConfig(t *testing.T) {
	type input struct {
		logsFilesListJSON string
		logLevel          string
		mongoURL          string
		mongoDB           string
		mongoCollection   string
		dropDB            bool
	}
	type expectedResult struct {
		wantConfig *Config
	}
	type testCase struct {
		id             int
		description    string
		input          input
		expectedResult expectedResult
	}
	type testSuite []testCase
	var ts = testSuite{
		testCase{
			id:          1,
			description: `Check configuration loading from cofig file`,
			input: input{
				logsFilesListJSON: `{"testdata/testfile1.log":"second_format","testdata/dir1/testfile2.log":"first_format"}`,
				logLevel:          "Info",
				mongoURL:          "localhost:27017",
				mongoDB:           "myDB",
				mongoCollection:   "logs",
				dropDB:            true,
			},
			expectedResult: expectedResult{
				wantConfig: &Config{
					LogsFilesListJSON: `{"testdata/testfile1.log":"second_format","testdata/dir1/testfile2.log":"first_format"}`,
					LogLevel:          "Info",
					MongoURL:          "localhost:27017",
					MongoDB:           "myDB",
					MongoCollection:   "logs",
					DropDB:            true,
					LogsFilesList:     map[string]string{"testdata/testfile1.log": "second_format", "testdata/dir1/testfile2.log": "first_format"},
				},
			},
		},
	}
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(currentDir, "config.toml")

	for _, tc := range ts {

		t.Run(fmt.Sprintf("Test%d:%s", tc.id, tc.description), func(t *testing.T) {

			err = testConfigCreate(configPath, tc.input.logsFilesListJSON, tc.input.logLevel, tc.input.mongoURL, tc.input.mongoDB, tc.input.mongoCollection, tc.input.dropDB)
			if err != nil {
				t.Fatalf("Error while creating test config: %v", err)
			}

			gotModel := LoadConfig(configPath)

			if !reflect.DeepEqual(gotModel, tc.expectedResult.wantConfig) {
				t.Errorf("LoadConfig() = %+v, \nwant %+v", gotModel, tc.expectedResult.wantConfig)
			}
		})
	}
}
