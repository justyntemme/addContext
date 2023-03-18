package main

import (
	"fmt"
	"io/ioutil"
	"log"
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

	defer os.RemoveAll(tempDir)
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
	c.Flags().StringP("question", "q", "", "question to ask")

	c.Flags().StringP("file", "f", "", "File to add in chat")
	c.Flags().StringP("key", "k", "", "OpenAI API key")
	err := c.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
func getApiKey(c *cobra.Command) string {

	apiKey, _ := c.Flags().GetString("key")
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
		if apiKey == "" {
			fmt.Println("API key not set")
			os.Exit(1)
		}

	}

	return apiKey
}

func addContext(c *cobra.Command, args []string) {
	// get the API key from the command parameter or the environment variable
	apiKey := getApiKey(c)
	text := []byte("")
	// get the text argument from the command
	url, err := c.Flags().GetString("url")
	if url != "" {
		tempDir, err := cloneRepo(url)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		text, err = readGoFiles(tempDir)
	} else {
		file, err := c.Flags().GetString("file")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		textbytes, err := readfile(file)
		text = []byte(textbytes)

	}
	question, err := c.Flags().GetString("question")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ctx := context.Background()

	client := gpt3.NewClient(apiKey)
	// Split text into multiple prompts
	req := gpt3.EmbeddingsRequest{
		Model: gpt3.TextEmbeddingAda002,
		Input: []string{string(text)}}
	resp, err := client.Embeddings(ctx, req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(resp.Object)
	fmt.Println(resp.Data[0].Embedding)
	var responses []gpt3.ChatCompletionResponse
	text = []byte("This is a csv of sports data, read it and use the data to answer the following question from the user, if you can't figure it out from the data, simply say i don't know")
	chatResp, err := client.ChatCompletion(ctx, gpt3.ChatCompletionRequest{
		Model: gpt3.GPT3Dot5Turbo0301,
		Messages: []gpt3.ChatCompletionRequestMessage{
			{
				Role:    "system",
				Content: string(text),
			},
			{
				Role:    "user",
				Content: question,
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Printf("%+v\n", chatResp)

	// Combine the responses into a single string
	var output strings.Builder
	for _, resp := range responses {
		output.WriteString(resp.Choices[0].Message.Content)
	}

	fmt.Print(output.String())
}

func readfile(filepath string) (string, error) {
	// Open the file at the given file path
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the contents of the file into a byte slice
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Convert the byte slice to a string and return it
	return string(bytes), nil
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
