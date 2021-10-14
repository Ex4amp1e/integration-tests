// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package prefetch exports suite that can do prefetch of required images once per suite.
package prefetch

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/gotestmd/pkg/suites/shell"
)

const (
	defaultDomain = "docker.io"
	officialLib   = "library"
	defaultTag    = ":latest"
)

// Suite creates `prefetch` daemonset which pulls all test images for all cluster nodes.
type Suite struct {
	shell.Suite
	Dir string
}

var once sync.Once

// SetupSuite prefetches docker images for each k8s node.
func (s *Suite) SetupSuite() {
	once.Do(func() {
		testImages, err := s.findTestImages()
		require.NoError(s.T(), err)

		tmpDir := uuid.NewString()
		require.NoError(s.T(), os.MkdirAll(tmpDir, 0750))

		r := s.Runner(tmpDir)

		r.Run(createNamespace)
		r.Run(strings.ReplaceAll(createConfigMap, "{{.TestImages}}", strings.Join(testImages, " ")))
		r.Run(createDaemonSet)
		r.Run(createKustomization)

		r.Run("kubectl apply -k .")
		r.Run("kubectl -n prefetch wait --timeout=10m --for=condition=ready pod -l app=prefetch")

		r.Run("kubectl delete ns prefetch")
		_ = os.RemoveAll(tmpDir)
	})
}

func (s *Suite) findTestImages() ([]string, error) {
	imagePattern := regexp.MustCompile(".*image: (?P<image>.*)")
	imageSubexpIndex := imagePattern.SubexpIndex("image")

	var testImages []string
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if ok, skipErr := s.shouldSkipWithError(info, err); ok {
			return skipErr
		}
		// #nosec
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if imagePattern.MatchString(scanner.Text()) {
				image := imagePattern.FindAllStringSubmatch(scanner.Text(), -1)[0][imageSubexpIndex]
				testImages = append(testImages, s.fullImageName(image))
			}
		}

		return nil
	}

	if err := filepath.Walk(filepath.Join(s.Dir, "apps"), walkFunc); err != nil {
		return nil, err
	}
	if err := filepath.Walk(filepath.Join(s.Dir, "examples", "spire"), walkFunc); err != nil {
		return nil, err
	}

	return testImages, nil
}

func (s *Suite) shouldSkipWithError(info os.FileInfo, err error) (bool, error) {
	if err != nil {
		return true, err
	}

	if info.IsDir() {
		if IsExcluded(info.Name()) {
			return true, filepath.SkipDir
		}
		return true, nil
	}

	if !strings.HasSuffix(info.Name(), ".yaml") {
		return true, nil
	}

	return false, nil
}

func (s *Suite) fullImageName(image string) string {
	var domain, remainder string
	i := strings.IndexRune(image, '/')
	if i == -1 || (!strings.ContainsAny(image[:i], ".:")) {
		domain, remainder = defaultDomain, image
	} else {
		domain, remainder = image[:i], image[i+1:]
	}
	if domain == defaultDomain && !strings.ContainsRune(remainder, '/') {
		remainder = officialLib + "/" + remainder
	}

	switch len(strings.Split(remainder, ":")) {
	case 2:
		// nothing to do
	case 1:
		remainder += defaultTag
	default:
		return ""
	}

	return domain + "/" + remainder
}
