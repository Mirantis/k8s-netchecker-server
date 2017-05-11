// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"io/ioutil"
)

type errProcessor struct {
	err error
}

func (ep *errProcessor) ReadBody(req *http.Request) []byte {
	if ep.err != nil {
		return nil
	}
	body := make([]byte, req.ContentLength)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ep.err = errors.New(
			fmt.Sprintf(
				"Error while reading bytes from the request's body. Details: %v", err))
	} else {
		req.Body.Close()
	}
	if len(body) < int(req.ContentLength) {
		ep.err = errors.New(
			fmt.Sprintf("%v out of %v bytes were read from the request's body.",
				len(body), req.ContentLength))
	}
	return body
}

func (ep *errProcessor) UnmarshalBytes(data []byte, dst interface{}) {
	if ep.err != nil {
		return
	}

	err := json.Unmarshal(data, dst)
	if err != nil {
		ep.err = errors.New(
			fmt.Sprintf("Error while unmarshaling data. Details: %v", err))
	}
}

func (ep *errProcessor) MarshalBytes(src interface{}) []byte {
	if ep.err != nil {
		return nil
	}

	marshaled, err := json.Marshal(src)
	if err != nil {
		ep.err = errors.New(
			fmt.Sprintf("Error while marshaling the agents' cache data. Details: %v", err))
	}
	return marshaled
}

func (ep *errProcessor) WriteBody(rw http.ResponseWriter, data []byte) {
	if ep.err != nil {
		return
	}

	_, err := rw.Write(data)
	if err != nil {
		ep.err = errors.New(
			fmt.Sprintf(
				"Error while writing the response's body. Details: %v", err))
	}
}

func ProcessRequest(r *http.Request, dst interface{}, rw http.ResponseWriter) error {
	ep := &errProcessor{}
	body := ep.ReadBody(r)
	ep.UnmarshalBytes(body, dst)
	if ep.err != nil {
		glog.Errorf("Failed to process the request's data. %v", ep.err)
		http.Error(rw, ep.err.Error(), http.StatusInternalServerError)
	}
	return ep.err
}

func ProcessResponse(rw http.ResponseWriter, data interface{}) error {
	ep := &errProcessor{}
	marshaled := ep.MarshalBytes(data)
	ep.WriteBody(rw, marshaled)

	if ep.err != nil {
		glog.Errorf("Failed to prepare the response. %v", ep.err)
		http.Error(rw, ep.err.Error(), http.StatusInternalServerError)
	}
	return ep.err
}
