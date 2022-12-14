/*
 * Copyright 2020 Intel Corporation, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package module

// App contains metadata for Apps
type SiteObject struct {
	Metadata      ObjectMetaData `json:"metadata"`
	Specification SiteObjectSpec `json:"spec"`
}

//SiteObjectSpec contains the parameters
type SiteObjectSpec struct {
	Hubs   []string `json:"hubs"`
	Url    string   `json:"url"`
	Subnet string   `json:"subnet"`
}

func (c *SiteObject) GetMetadata() ObjectMetaData {
	return c.Metadata
}

func (c *SiteObject) GetType() string {
	return "site"
}
