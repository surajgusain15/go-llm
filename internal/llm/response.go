package llm

import (
	"fmt"
	"io"
	"net/http"
)

func (c *OllamaClient) do(
	req *http.Request,
) (*http.Response, error) {

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"execute request: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf(
			"ollama returned status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return resp, nil
}
