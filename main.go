package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const DOCKER_IMAGE = "mcr.microsoft.com/playwright:v1.52.0-noble"

func main() {
	mcpServer := server.NewMCPServer(
		"python-executor",
		"1.0.0",
	)

	// write python execute by run in docker container for web scrapting using playwright
	pythonTool := mcp.NewTool(
		"python-scraper-executor",
		mcp.WithDescription(
			"Python executor tool in separate docker container, using playWright and headless browser for web scraping, and return output in print statements!",
		),
		mcp.WithString(
			"code",
			mcp.Description("Python code to be execute"),
			mcp.Required(),
		),
		mcp.WithArray(
			"libraries",
			mcp.Description("Python libraries to be installed in the container"),
		),
	)

	mcpServer.AddTool(pythonTool, handlePythonExecution)

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handlePythonExecution(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	code, ok := request.Params.Arguments["code"].(string)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'code' argument"), nil
	}

	var libraries []string
	if libs, ok := request.Params.Arguments["libraries"].([]interface{}); ok {
		libraries = make([]string, 0, len(libs))
		for _, lib := range libs {
			if libStr, ok := lib.(string); ok {
				libraries = append(libraries, libStr)
			}
		}
	}

	directory, err := os.MkdirTemp("", "python-executor")
	if err != nil {
		return mcp.NewToolResultError("Failed to create temporary directory"), nil
	}
	defer os.RemoveAll(directory)

	scriptPath := path.Join(directory, "script.py")
	err = os.WriteFile(scriptPath, []byte(code), 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write script to file: %v", err)), nil
	}

	cmdArgs := []string{
		"run",
		"--rm",
		"-v",
		fmt.Sprintf("%s:/app", directory),
		"--net=host",
		DOCKER_IMAGE,
	}

	scriptContent := "#!/bin/bash\nset -e\n"

	// Install virtual environment support and create a venv
	scriptContent += "apt-get update -qq > /dev/null 2>&1\n"
	scriptContent += "apt-get install -y python3-venv -qq > /dev/null 2>&1\n"
	scriptContent += "python3 -m venv /tmp/pyenv\n"

	if len(libraries) > 0 {
		scriptContent += "/tmp/pyenv/bin/pip install --no-cache-dir --quiet " + strings.Join(libraries, " ") + "\n"
	}

	scriptContent += "cd /app && /tmp/pyenv/bin/python /app/script.py\n"

	shellScriptPath := path.Join(directory, "run.sh")
	err = os.WriteFile(shellScriptPath, []byte(scriptContent), 0755)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write shell script: %v", err)), nil
	}

	cmdArgs = append(cmdArgs, "bash", "/app/run.sh")

	cmd := exec.Command("docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Try to extract the error from Python's traceback
			errStr := string(out)
			return mcp.NewToolResultError(fmt.Sprintf("Python executed failed with %d error: %s", exitError.ExitCode(), errStr)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute Python script: %v\nOutput: %s", err, string(out))), nil
	}
	result := string(out)

	return mcp.NewToolResultText(result), nil
}
