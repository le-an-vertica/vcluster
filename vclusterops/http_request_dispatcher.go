/*
 (c) Copyright [2023] Open Text.
 Licensed under the Apache License, Version 2.0 (the "License");
 You may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package vclusterops

import (
	"github.com/vertica/vcluster/vclusterops/vlog"
)

type HTTPRequestDispatcher struct {
	pool AdapterPool
}

func MakeHTTPRequestDispatcher() HTTPRequestDispatcher {
	newHTTPRequestDispatcher := HTTPRequestDispatcher{}

	return newHTTPRequestDispatcher
}

// set up the pool connection for each host
func (dispatcher *HTTPRequestDispatcher) Setup(hosts []string) {
	dispatcher.pool = getPoolInstance()

	for _, host := range hosts {
		adapter := MakeHTTPAdapter()
		adapter.host = host
		dispatcher.pool.connections[host] = &adapter
	}
}

func (dispatcher *HTTPRequestDispatcher) sendRequest(clusterHTTPRequest *ClusterHTTPRequest) error {
	vlog.LogInfoln("HTTP request dispatcher's sendRequest is called")
	return dispatcher.pool.sendRequest(clusterHTTPRequest)
}
