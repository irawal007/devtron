/*
 * Copyright (c) 2024. Devtron Inc.
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

package bean

const (
	HostUrlKey                     string = "url"
	API_SECRET_KEY                 string = "apiTokenSecret"
	ENFORCE_DEPLOYMENT_TYPE_CONFIG string = "enforceDeploymentTypeConfig"
	PRIORITY_DEPLOYMENT_CONDITION  string = "priorityDeploymentCondition"
	UserPreferencesResourcesKey           = "resources"
)

type AttributesDto struct {
	Id     int    `json:"id"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
	Active bool   `json:"active"`
	UserId int32  `json:"-"`
}

type UserAttributesDto struct {
	EmailId string `json:"emailId"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	UserId  int32  `json:"-"`
}
