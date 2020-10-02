package bitbucket

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/reviewdog/reviewdog/service/bitbucket/openapi"
)

func checkAPIError(err error, resp *http.Response, expectedCode int) error {
	if err != nil {
		e, ok := err.(openapi.GenericOpenAPIError)
		if ok {
			return fmt.Errorf(`Bitbucket API error:
	Response error: %s
	Response body: %s`,
				e.Error(), string(e.Body()))
		}
	}

	if resp != nil && resp.StatusCode != expectedCode {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Received unexpected %d code from Bitbucket API", resp.StatusCode)
		if len(body) > 0 {
			msg += " with message:\n" + string(body)
		}
		return errors.New(msg)
	}

	return err
}
