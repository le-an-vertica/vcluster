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
	"fmt"

	"github.com/vertica/vcluster/vclusterops/util"
	"github.com/vertica/vcluster/vclusterops/vlog"
)

type HTTPSCheckSubclusterOp struct {
	OpBase
	OpHTTPSBase
	scName      string
	isSecondary bool
	ctlSetSize  int
}

func makeHTTPSCheckSubclusterOp(log vlog.Printer, useHTTPPassword bool, userName string, httpsPassword *string,
	scName string, isPrimary bool, ctlSetSize int) (HTTPSCheckSubclusterOp, error) {
	httpsCheckSubclusterOp := HTTPSCheckSubclusterOp{}
	httpsCheckSubclusterOp.name = "HTTPSCheckSubclusterOp"
	httpsCheckSubclusterOp.log = log.WithName(httpsCheckSubclusterOp.name)
	httpsCheckSubclusterOp.scName = scName
	httpsCheckSubclusterOp.isSecondary = !isPrimary
	httpsCheckSubclusterOp.ctlSetSize = ctlSetSize

	httpsCheckSubclusterOp.useHTTPPassword = useHTTPPassword
	if useHTTPPassword {
		err := util.ValidateUsernameAndPassword(httpsCheckSubclusterOp.name, useHTTPPassword, userName)
		if err != nil {
			return httpsCheckSubclusterOp, err
		}
		httpsCheckSubclusterOp.userName = userName
		httpsCheckSubclusterOp.httpsPassword = httpsPassword
	}
	return httpsCheckSubclusterOp, nil
}

func (op *HTTPSCheckSubclusterOp) setupClusterHTTPRequest(hosts []string) error {
	for _, host := range hosts {
		httpRequest := HostHTTPRequest{}
		httpRequest.Method = GetMethod
		httpRequest.buildHTTPSEndpoint("subclusters/" + op.scName)
		if op.useHTTPPassword {
			httpRequest.Password = op.httpsPassword
			httpRequest.Username = op.userName
		}
		op.clusterHTTPRequest.RequestCollection[host] = httpRequest
	}

	return nil
}

func (op *HTTPSCheckSubclusterOp) prepare(execContext *OpEngineExecContext) error {
	if len(execContext.upHosts) == 0 {
		return fmt.Errorf(`[%s] Cannot find any up hosts in OpEngineExecContext`, op.name)
	}
	execContext.dispatcher.setup(execContext.upHosts)

	return op.setupClusterHTTPRequest(execContext.upHosts)
}

func (op *HTTPSCheckSubclusterOp) execute(execContext *OpEngineExecContext) error {
	if err := op.runExecute(execContext); err != nil {
		return err
	}

	return op.processResult(execContext)
}

// the following struct will store a subcluster's information for this op
type SCInfo struct {
	SCName      string `json:"subcluster_name"`
	IsSecondary bool   `json:"is_secondary"`
	CtlSetSize  int    `json:"control_set_size"`
}

func (op *HTTPSCheckSubclusterOp) processResult(_ *OpEngineExecContext) error {
	var err error

	for host, result := range op.clusterHTTPRequest.ResultCollection {
		op.logResponse(host, result)

		if result.isUnauthorizedRequest() {
			// skip checking response from other nodes because we will get the same error there
			return result.err
		}
		if !result.isPassing() {
			err = result.err
			// try processing other hosts' responses when the current host has some server errors
			continue
		}

		// decode the json-format response
		// A successful response object will be like below:
		/*
			{
			    "subcluster_name": "sc1",
			    "control_set_size": 2,
			    "is_secondary": true,
			    "is_default": false,
			    "sandbox": ""
			}
		*/
		scInfo := SCInfo{}
		err = op.parseAndCheckResponse(host, result.content, &scInfo)
		if err != nil {
			return fmt.Errorf(`[%s] fail to parse result on host %s, details: %w`, op.name, host, err)
		}

		if scInfo.SCName != op.scName {
			return fmt.Errorf(`[%s] new subcluster name should be '%s' but got '%s'`, op.name, op.scName, scInfo.SCName)
		}
		if scInfo.IsSecondary != op.isSecondary {
			if op.isSecondary {
				return fmt.Errorf(`[%s] new subcluster should be a secondary subcluster but got a primary subcluster`, op.name)
			}
			return fmt.Errorf(`[%s] new subcluster should be a primary subcluster but got a secondary subcluster`, op.name)
		}
		if scInfo.CtlSetSize != op.ctlSetSize {
			return fmt.Errorf(`[%s] new subcluster should have control set size as %d but got %d`, op.name, op.ctlSetSize, scInfo.CtlSetSize)
		}

		return nil
	}

	return err
}

func (op *HTTPSCheckSubclusterOp) finalize(_ *OpEngineExecContext) error {
	return nil
}
