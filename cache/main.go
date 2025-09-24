package cache

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mxlint/mxlint-cli/shared"
	"gopkg.in/yaml.v3"
)

// move this to shared
func GetMprPath(inputDirectory string) (string, error) {
	var mprPath string
	found := false

	err := filepath.Walk(inputDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, ".mendix-cache") {
			// log.Debugf("Skipping system managed file %s", path)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mpr") {
			mprPath = path
			found = true
			return filepath.SkipDir // Stop walking once we find an MPR file
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if !found {
		return "", fmt.Errorf("no .mpr file found in directory: %s", inputDirectory)
	}

	return mprPath, nil
}

func ExporApptMeta(inputDirectory string, outputDirectory string, documents []shared.MxDocument) (func(data MxFileListMeta) error, MxFileListMeta, MxFileListMeta, error) {
	var fullOutputPath = filepath.Join(outputDirectory, "MetaFileList.yaml")
	diffed := make(MxFileListMeta)
	diffedPathBased := make(MxCacheDiffMap)

	fileList, err := createCahceMap(documents)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating cache: %v", err)
	}

	if _, err := os.Stat(fullOutputPath); os.IsNotExist(err) {
		fmt.Println("File is not found we will have to create it now")

		return func(data MxFileListMeta) error {
			err = commitCache(data, outputDirectory, "MetaFileList.yaml")
			err = commitCacheDiff(diffedPathBased, outputDirectory)
			if err != nil {
				return fmt.Errorf("error comitting cache: %v", err)
			}
			return err
		}, nil, fileList, nil
	}

	mprPath, err := GetMprPath(inputDirectory)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting mpr path: %v", err)
	}

	fmt.Println("File is found we will diff it now")

	diffed, err = cacheDiff(mprPath, fullOutputPath)

	return func(data MxFileListMeta) error {
		err = commitCache(data, outputDirectory, "MetaFileList.yaml")
		for key, item := range diffed {
			if item.Path != "" {
				diffedPathBased[item.Path] = MxCacheDiff{ID: key, diffType: item.diffType}
			}
		}
		err = commitCacheDiff(diffedPathBased, outputDirectory)

		if err != nil {
			return fmt.Errorf("error comitting cache: %v", err)
		}
		return err
	}, diffed, fileList, nil

}

func createCahceMap(units []shared.MxDocument) (MxFileListMeta, error) {
	metaList := MxFileListMeta{}

	for _, unit := range units {

		meta := MxFileMeta{
			Hash: unit.Hash,
		}
		metaList[unit.Id] = meta
	}

	return metaList, nil
}

func cacheDiff(MPRFilePath string, outputPath string) (MxFileListMeta, error) {
	db, err := sql.Open("sqlite", MPRFilePath)

	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	rows, err := db.Query("SELECT UnitID, ContentsHash FROM Unit")

	if err != nil {
		return nil, fmt.Errorf("error querying units: %v", err)
	}

	defer rows.Close()

	cachedApp, err := GetCahce(outputPath)
	visited, err := GetCahce(outputPath)

	if err != nil {
		return nil, fmt.Errorf("error fetching cache: %v", err)
	}

	for rows.Next() {
		var UnitId []byte
		var Hash string

		if err := rows.Scan(&UnitId, &Hash); err != nil {
			return nil, fmt.Errorf("error scanning unit: %v", err)
		}

		var strId = base64.StdEncoding.EncodeToString(UnitId)

		if meta, ok := cachedApp[strId]; !ok {
			cachedApp[strId] = MxFileMeta{Hash: Hash, diffType: "add"}
		} else if ok && meta.Hash != Hash {
			cachedApp[strId] = MxFileMeta{Hash: Hash, diffType: "update"}
		} else {
			delete(cachedApp, strId)
		}
		delete(visited, strId)

		// itemes can be removed, added or just diffed
		// added and diffed should be relinted
		// removed should be removed from json and not relinted
	}
	if len(visited) > 0 {
		// todo handle this case
		for key, item := range visited {
			cachedApp[key] = MxFileMeta{
				Hash:     item.Hash,
				diffType: "delete",
			}
		}
	}

	return cachedApp, nil
}

func commitCache(metaList any, outputDirectory string, fileName string) error {
	// write metadata to file
	metadataYAML, err := yaml.Marshal(metaList)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %v", err)
	}

	if _, err := os.Stat(outputDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDirectory, 0755); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	metadataFileName := filepath.Join(outputDirectory, fileName)

	if err := os.WriteFile(metadataFileName, metadataYAML, 0644); err != nil {
		return fmt.Errorf("error writing metadata file: %v", err)
	}

	return nil
}

func commitCacheDiff(diffMap MxCacheDiffMap, outputDirectory string) error {
	fullPath := filepath.Join(outputDirectory, "cacheDiff.yaml")
	fmt.Println("output directory", outputDirectory)
	var metadataYAML []byte
	fmt.Println("this is the diff map", diffMap)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		metadataYAML, err = yaml.Marshal(diffMap)
		if err != nil {
			return fmt.Errorf("error marshaling metadata: %v", err)
		}
	} else {
		prevCahceFile, err := os.ReadFile(fullPath)

		if err != nil {
			return fmt.Errorf("error reading old cache: %v", err)
		}

		var prevCache MxCacheDiffMap
		yaml.Unmarshal(prevCahceFile, &prevCache)

		if len(diffMap) > 0 {
			for key := range prevCache {
				prevCache[key] = diffMap[key]
			}

			for key := range diffMap {
				prevCache[key] = diffMap[key]
			}

			metadataYAML, err = yaml.Marshal(prevCache)
			if err != nil {
				return fmt.Errorf("error marshaling metadata: %v", err)
			}
		}

	}

	writeCacheDiff(outputDirectory, metadataYAML)

	return nil
}

func GetDiffCache(outputDirectory string) (MxCacheDiffMap, error) {
	data, err := os.ReadFile(path.Join(outputDirectory, "cacheDiff.yaml"))
	if err != nil {
		return nil, fmt.Errorf("can't read file")
	}
	var cacheMap = make(MxCacheDiffMap)

	err = yaml.Unmarshal(data, &cacheMap)
	if err != nil {
		return nil, fmt.Errorf("error parsing the file")
	}
	return cacheMap, nil
}

// Just write an empty object as i dont want to handle file removeal and recreations
func InvalidateCahce(outputDirectory string) error {
	empty := make(MxCacheDiffMap)
	yaml, err := yaml.Marshal(empty)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}
	writeCacheDiff(outputDirectory, yaml)
	return nil
}

func GetCahce(path string) (MxFileListMeta, error) {
	cacheFile, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("error reading cache file : %v", err)
	}
	cachedApp := make(MxFileListMeta)

	err = yaml.Unmarshal(cacheFile, &cachedApp)

	if err != nil {
		return nil, fmt.Errorf("error Parsing cache file : %v", err)
	}
	return cachedApp, nil
}
func writeCacheDiff(outputDirectory string, YAML []byte) error {
	fullPath := filepath.Join(outputDirectory, "cacheDiff.yaml")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDirectory, 0755); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	if err := os.WriteFile(fullPath, YAML, 0644); err != nil {
		return fmt.Errorf("error writing metadata file: %v", err)
	}

	return nil
}
