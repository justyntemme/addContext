package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	gpt3 "github.com/PullRequestInc/go-gpt3"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"golang.org/x/net/context"
)

func cloneRepo(url string) (string, error) {
	// Create a temporary directory
	tempDir, err := ioutil.TempDir("", "repoToAdd")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return "", err
	}

	// Clone the repository to the temporary directory
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return "", err
	}

	fmt.Println("Repository cloned to", tempDir)
	return tempDir, err
}

func main() {
	var c = &cobra.Command{
		Use:   "addContext",
		Short: "add context to questions for gptApi via github repositories",
		Run:   addContext,
	}

	// add the text and key flags to the command
	c.Flags().StringP("url", "u", "", "the repositoy to evaluate")
	c.Flags().StringP("question", "q", "", "question to ask about code")
	c.Flags().StringP("key", "k", "", "OpenAI API key")
	err := c.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func addContext(c *cobra.Command, args []string) {
	// get the API key from the command parameter or the environment variable
	apiKey, _ := c.Flags().GetString("key")
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
		if apiKey == "" {
			fmt.Println("API key not set")
			os.Exit(1)
		}
	}

	// get the text argument from the command
	url, err := c.Flags().GetString("url")
	tempDir, err := cloneRepo(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	question, err := c.Flags().GetString("question")

	text, err := readGoFiles(tempDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ctx := context.Background()

	client := gpt3.NewClient(apiKey)
	// Split text into multiple prompts
	var prompts []string
	code := string(" `") + string(text) + string("` \n")
	prompts = append(prompts, code+question)

	// Call the OpenAI API to generate a completion for each prompt
	var responses []gpt3.CompletionResponse
	for _, prompt := range prompts {
		resp, err := client.Completion(ctx, gpt3.CompletionRequest{
			Prompt: []string{prompt},
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		responses = append(responses, *resp)
	}

	// Combine the responses into a single string
	var output strings.Builder
	for _, resp := range responses {
		output.WriteString(resp.Choices[0].Text)
	}

	fmt.Print(output.String())
	defer os.RemoveAll(tempDir)
}

func readGoFiles(dirPath string) ([]byte, error) {
	var buffer []byte

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process Go files
		if strings.HasSuffix(path, ".go") {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			buffer = append(buffer, data...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return buffer, nil
}
