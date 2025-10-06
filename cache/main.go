package cache

import (
	"database/sql"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mxlint/mxlint-cli/shared"
	"sigs.k8s.io/yaml"
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

func ExporApptMeta(inputDirectory string, outputDirectory string, documents []shared.MxDocument, debug bool) (func(data MxFileListMeta) error, MxFileListMeta, MxFileListMeta, error) {
	var fullOutputPath = filepath.Join(outputDirectory, "cache", "MetaFileList.gob")

	if _, err := os.Stat(path.Join(outputDirectory, "cache")); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDirectory, 0755); err != nil {
			return nil, nil, nil, fmt.Errorf("error creating directory: %v", err)
		}
	}

	diffed := make(MxFileListMeta)
	diffedPathBased := MxCacheDiffWrapper{
		Status: "vaild",
	}

	fileList, err := createCahceMap(documents)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating cache: %v", err)
	}

	if _, err := os.Stat(fullOutputPath); os.IsNotExist(err) {
		return func(data MxFileListMeta) error {
			err = saveCache(outputDirectory, "MetaFileList", data, debug)
			err = saveCacheDiff(outputDirectory, diffedPathBased, debug)

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
		diffedPathBased := make(MxCacheDiffMap)

		err = saveCache(outputDirectory, "MetaFileList", data, debug)

		for key, item := range diffed {
			if item.Path != "" {
				diffedPathBased[item.Path] = MxCacheDiff{ID: key, DiffType: item.DiffType, Hash: item.Hash}
			}
		}

		err = saveCacheDiff(outputDirectory, MxCacheDiffWrapper{
			Status: "valid",
			Data:   diffedPathBased,
		}, debug)

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

	cachedApp, err := GetCache(outputPath)
	visited, err := GetCache(outputPath)

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
			cachedApp[strId] = MxFileMeta{Hash: Hash, DiffType: "add"}
		} else if ok && meta.Hash != Hash {

			cachedApp[strId] = MxFileMeta{Hash: Hash, DiffType: "update"}
		} else {
			delete(cachedApp, strId)
		}
		delete(visited, strId)
	}

	if len(visited) > 0 {
		for key, item := range visited {
			cachedApp[key] = MxFileMeta{
				Hash:     item.Hash,
				DiffType: "delete",
				Path:     cachedApp[key].Path,
			}
		}
	}

	return cachedApp, nil
}

func commitCacheYAML(metaList any, outputDirectory string) error {
	// write metadata to file
	metadataYAML, err := yaml.Marshal(metaList)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %v", err)
	}

	if err := os.WriteFile(outputDirectory, metadataYAML, 0644); err != nil {
		return fmt.Errorf("error writing metadata file: %v", err)
	}

	return nil
}

func saveCache(outputDirectory string, filename string, cache MxFileListMeta, debug bool) error {
	fullPath := path.Join(outputDirectory, "cache", filename+".gob")

	file, err := os.Create(fullPath)

	if err != nil {
		return err
	}

	if debug {
		err = commitCacheYAML(cache, path.Join(outputDirectory, "cache", filename+".yaml"))
		if err != nil {
			return err
		}
	}

	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(cache)

	if err != nil {
		return err
	}

	return nil
}

func loadCache[T any](filePath string, makeFunc func() T) (T, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return makeFunc(), nil
		}
		return makeFunc(), err
	}

	defer file.Close()

	cache := makeFunc()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&cache)

	if err != nil {

		return makeFunc(), err
	}

	return cache, nil
}

func makeDiff() MxCacheDiffWrapper {
	return MxCacheDiffWrapper{Status: "vaild", Data: nil}
}

func saveCacheDiff(outputDirectory string, cache MxCacheDiffWrapper, debug bool) error {
	gobPath := path.Join(outputDirectory, "cache", "diff.gob")
	yamlPath := path.Join(outputDirectory, "cache", "diff.yaml")
	prevCache, err := loadCache(gobPath, makeDiff)
	fmt.Println("here", prevCache, cache, err)

	if err != nil {
		return err
	}

	file, err := os.Create(gobPath)
	if err != nil {
		return err
	}

	// merge old and new caches
	if len(prevCache.Data) == 0 && len(cache.Data) != 0 {
		prevCache.Data = cache.Data
	} else if len(cache.Data) > 0 {
		for key := range cache.Data {
			prevCache.Data[key] = cache.Data[key]
		}
	}

	// write gob file
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(prevCache)
	defer file.Close()

	if err != nil {
		return err
	}

	if debug {
		yaml, err := yaml.Marshal(prevCache)

		if err != nil {
			return err
		}

		if err := os.WriteFile(yamlPath, yaml, 0644); err != nil {
			return fmt.Errorf("error writing metadata file: %v", err)
		}

	}

	return nil
}

func GetDiffCache(outputDirectory string) (MxCacheDiffWrapper, error) {
	data, err := os.ReadFile(path.Join(outputDirectory, "cache", "diff.yaml"))
	var cacheMap = MxCacheDiffWrapper{}

	if err != nil {
		return cacheMap, fmt.Errorf("can't read file")
	}

	err = yaml.Unmarshal(data, &cacheMap)
	if err != nil {
		return cacheMap, fmt.Errorf("error parsing the file")
	}

	return cacheMap, nil
}

// Just write an empty object as i dont want to handle file removeal and recreations
func InvalidateCahce(outputDirectory string, full bool, debug bool) error {
	var status string
	if full {
		status = "invalid"
	} else {
		status = "valid"
	}

	empty := MxCacheDiffWrapper{
		Status: status,
	}

	err := saveCacheDiff(outputDirectory, empty, debug)

	if err != nil {
		return fmt.Errorf("error invalidating cache: %v", err)
	}

	return nil
}

func GetCache(path string) (MxFileListMeta, error) {
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
