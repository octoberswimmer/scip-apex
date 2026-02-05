package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForceCLI/force-md/metadata/objects"
	"github.com/ForceCLI/force-md/metadata/objects/field"
	"github.com/octoberswimmer/aer/ast"
)

func parseObjectFiles(dirs []string) *ast.CompilationUnit {
	type objectInfo struct {
		cls    *ast.ClassDeclaration
		fields []*ast.FieldDeclaration
	}

	objectMap := make(map[string]*objectInfo)

	getOrCreate := func(name, filePath string) *objectInfo {
		key := strings.ToLower(name)
		if info, ok := objectMap[key]; ok {
			if filePath != "" && info.cls.FilePath == "" {
				info.cls.FilePath = filePath
			}
			return info
		}
		info := &objectInfo{
			cls: &ast.ClassDeclaration{
				Name:     name,
				FilePath: filePath,
				Line:     1,
				Column:   0,
			},
		}
		objectMap[key] = info
		return info
	}

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			base := filepath.Base(path)

			switch {
			case strings.HasSuffix(base, ".object") && !strings.HasSuffix(base, "-meta.xml"):
				objectName := strings.TrimSuffix(base, ".object")
				obj, err := objects.Open(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: parse error in %s: %v\n", path, err)
					return nil
				}
				oi := getOrCreate(objectName, absPath)
				for _, f := range obj.Fields {
					if f.FullName == "" {
						continue
					}
					oi.fields = append(oi.fields, &ast.FieldDeclaration{
						Name:   f.FullName,
						Type:   "Object",
						Line:   1,
						Column: 0,
					})
				}

			case strings.HasSuffix(base, ".object-meta.xml"):
				objectName := strings.TrimSuffix(base, ".object-meta.xml")
				getOrCreate(objectName, absPath)

			case strings.HasSuffix(base, ".field-meta.xml"):
				fieldName := strings.TrimSuffix(base, ".field-meta.xml")
				parentDir := filepath.Dir(path)
				grandparentDir := filepath.Dir(parentDir)
				objectName := filepath.Base(grandparentDir)

				_, err := field.Open(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: parse error in %s: %v\n", path, err)
					return nil
				}

				oi := getOrCreate(objectName, "")
				oi.fields = append(oi.fields, &ast.FieldDeclaration{
					Name:     fieldName,
					Type:     "Object",
					FilePath: absPath,
					Line:     1,
					Column:   0,
				})
			}

			return nil
		})
	}

	if len(objectMap) == 0 {
		return nil
	}

	cu := &ast.CompilationUnit{}
	for _, info := range objectMap {
		info.cls.Fields = info.fields
		cu.Classes = append(cu.Classes, info.cls)
	}
	return cu
}
