package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {

	// TODO: map from group/kind to namespaced (ignore version)
	// schema.GroupKind
	gvkNamespaced, err := parseGVKNamespacedMapping()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	file, err := os.OpenFile("discovery/local_discovery.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		os.Exit(1)
	}

	file.WriteString(fmt.Sprintf("package discovery\n\n"))
	file.WriteString(fmt.Sprintf("import \"k8s.io/apimachinery/pkg/runtime/schema\"\n\n"))
	file.WriteString(fmt.Sprintf("var gvkNamespaced = map[schema.GroupVersionKind]bool{\n"))
	for gvk, namespaced := range gvkNamespaced {
		file.WriteString(fmt.Sprintf("  schema.GroupVersionKind{Group: \"%s\", Version: \"%s\", Kind: \"%s\"}: %s,\n", gvk.Group, gvk.Version, gvk.Kind, strconv.FormatBool(namespaced)))
	}
	file.WriteString(fmt.Sprintf("}"))

	if err := file.Close(); err != nil {
		os.Exit(1)
	}
}

func extractGVKNamespacedMapping(typesFileName, group, version string) (map[schema.GroupVersionKind]bool, error) {
	gvkNamespaced := map[schema.GroupVersionKind]bool{}

	file, err := os.Open(typesFileName)
	if err != nil {
		return gvkNamespaced, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Find kind marker
		if line == "// +genclient" {
			gvk := schema.GroupVersionKind{
				Group:   group,
				Version: version,
			}
			// Determine whether type is Namespaced
			namespaced := false
			for scanner.Scan() {
				line := scanner.Text()
				// Break if line is not a comment
				if !strings.HasPrefix(line, "//") {
					break
				}
				if line == "// +genclient:nonNamespaced" {
					namespaced = true
					break
				}
			}
			// Extract kind
			for scanner.Scan() {
				line := scanner.Text()
				// Break if line is not a comment, whitespace or a type definition
				if !strings.HasPrefix(line, "//") && line != "" && !strings.HasPrefix(line, "type ") {
					break
				}
				if strings.HasPrefix(line, "type ") {
					gvk.Kind = strings.Split(line, " ")[1]
				}
			}
			if gvk.Kind == "" {
				return gvkNamespaced, fmt.Errorf("Unable to find kind: %s", typesFileName)
			}
			gvkNamespaced[gvk] = namespaced
		}
	}

	return gvkNamespaced, nil
}

func extractSubstring(registerFileName, prefix, suffix string) (string, error) {
	file, err := os.Open(registerFileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, suffix) {
			return strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix), nil
		}
	}

	return "", fmt.Errorf("failed to find substring: %s", registerFileName)
}

func parseGVKNamespacedMapping() (map[schema.GroupVersionKind]bool, error) {

	gvkNamespaced := map[schema.GroupVersionKind]bool{}

	err := filepath.Walk("/Users/luke/go/src/k8s.io/api",
		func(fileName string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(fileName, "/types.go") {
				group, err := extractSubstring(strings.TrimSuffix(fileName, "/types.go")+"/register.go", "const GroupName = \"", "\"")
				if err != nil {
					return err
				}
				version, err := extractSubstring(strings.TrimSuffix(fileName, "/types.go")+"/register.go", "var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: \"", "\"}")
				if err != nil {
					return err
				}
				extractedGVKNamespaced, err := extractGVKNamespacedMapping(fileName, group, version)
				if err != nil {
					return err
				}
				for k, v := range extractedGVKNamespaced {
					gvkNamespaced[k] = v
				}
			}
			return nil
		})
	if err != nil {
		return gvkNamespaced, nil
	}

	// input := "api-resources.txt"

	// 	gv, err := schema.ParseGroupVersion(words[len(words)-3])
	// 	if err != nil {
	// 		return gvkNamespaced, nil
	// 	}
	// 	namespaced, err := strconv.ParseBool(words[len(words)-2])
	// 	if err != nil {
	// 		return gvkNamespaced, err
	// 	}
	// 	kind := words[len(words)-1]

	// 	gvk := schema.GroupVersionKind{
	// 		gv.Group,
	// 		gv.Version,
	// 		kind,
	// 	}
	// 	gvkNamespaced[gvk] = namespaced
	// }

	return gvkNamespaced, nil
}
