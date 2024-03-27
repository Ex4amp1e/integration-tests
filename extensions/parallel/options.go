// Copyright (c) 2024 Pragmagic Inc. and/or its affiliates.
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

package parallel

type parallelOptions struct {
	excludedTests []string
}

// Option is an option pattern for parallel package
type Option func(o *parallelOptions)

// WithExcludedTests - set a list of tests to exclude from parallel execution
func WithExcludedTests(tests []string) Option {
	return func(o *parallelOptions) {
		o.excludedTests = tests
	}
}