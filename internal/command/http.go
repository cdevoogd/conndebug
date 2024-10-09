package command

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type HTTP struct {
	URL string `arg:"" name:"url" help:"the URL to send a request to"`
}

func (cmd *HTTP) Run() error {
	req, err := http.NewRequest(http.MethodGet, cmd.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}
