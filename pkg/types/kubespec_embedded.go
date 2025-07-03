package types

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"sort"
)

//go:embed kubespec_data/*.json.gz
var kubespecDataFS embed.FS

var kubernetesVersions = []string{
	"v1.18", "v1.19", "v1.20", "v1.21", "v1.22", "v1.23",
	"v1.24", "v1.25", "v1.26", "v1.27", "v1.28", "v1.29",
	"v1.30", "v1.31", "v1.32", "v1.33",
}

func KubespecRegistry() (*Registry, error) {
	r := NewRegistry()
	since := map[string]map[string]map[string]string{}
	versions := append([]string(nil), kubernetesVersions...)
	sort.Strings(versions)
	for _, v := range versions {
		b, err := kubespecDataFS.ReadFile("kubespec_data/" + v + ".json.gz")
		if err != nil {
			return nil, err
		}
		zr, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		var spec map[string]interface{}
		if err := json.NewDecoder(zr).Decode(&spec); err != nil {
			_ = zr.Close()
			return nil, err
		}
		if err := zr.Close(); err != nil {
			return nil, err
		}
		defs, _ := extractDefinitionsLocal(spec)
		for _, schemaData := range defs {
			schemaObj, ok := schemaData.(map[string]interface{})
			if !ok {
				continue
			}
			gvkList, ok := schemaObj["x-kubernetes-group-version-kind"].([]interface{})
			if !ok {
				continue
			}
			for _, gvk := range gvkList {
				gvkMap, ok := gvk.(map[string]interface{})
				if !ok {
					continue
				}
				grp := getStringFieldLocal(gvkMap, "group")
				ver := getStringFieldLocal(gvkMap, "version")
				kind := getStringFieldLocal(gvkMap, "kind")
				td := convertSchemaToTypeDefLocal(grp, ver, kind, "", schemaObj)
				setFieldSince(&td, v, since)
				r.AddType(&td)
			}
		}
	}
	return r, nil
}
