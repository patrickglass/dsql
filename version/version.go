// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package version

import (
	pv "github.com/prometheus/common/version"
)

// Version specifier for the build which will be overridden via ldflags
var (
	Version   = "develop"
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
)

// init will copy the build version information into prometheus
func init() {
	pv.Version = Version
	pv.Revision = Revision
	pv.Branch = Branch
	pv.BuildUser = BuildUser
	pv.BuildDate = BuildDate
}
