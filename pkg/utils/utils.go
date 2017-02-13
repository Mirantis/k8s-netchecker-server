package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang/glog"
)

type errProcessor struct {
	err error
}

func (ep *errProcessor) ReadBody(req *http.Request) []byte {
	if ep.err != nil {
		return nil
	}
	body := make([]byte, req.ContentLength)
	n, err := req.Body.Read(body)
	if n <= 0 && err != nil {
		ep.err = errors.New(
			fmt.Sprintf(
				"Error while reading bytes from the request's body. Details: %v", err))
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
