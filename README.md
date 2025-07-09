# BforeAI CLI
    ▒▒▒▒▒▒▒▒▒▒                                              
    ▒    ▒    ▒    ████    ██                     ██    █   
    ▒   ▒ ▒   ▒    █   █ ████  ███   █ ██ ████    ███   █   
    ░ ▒▒    ▒▒     █████  ██  █    █ ██  █    █  █  █   █   
    ░░  ▒░▒▒  ▒    █   ██ ██  █    █ ██  █      ██████  █   
    ░    ░    ▒    █████  ██   ████  ██   ████  █    ██ █   
    ░░░░░░░▒▒▒
A powerful, interactive command-line interface (CLI) built with Go for interacting with the Bfore.ai API. This tool provides a terminal-like experience for authentication, data querying, and file management, and serves as a comprehensive example of building a real-world Go application.

## Features

* **Interactive Shell**: A user-friendly shell environment powered by `go-prompt`, complete with command history and a dynamic prompt.

![interactive-shell](/img/bforecli1.png)

* **Secure Authentication**: `login` and `logout` commands to manage session tokens for accessing protected API endpoints. __Password Masking Coming Soon__

![login](/img/interactive-prompt.png)

* **Dynamic Prompt**: The command prompt changes to show the currently logged-in user (e.g., `dave@go-cli $`).
* **Concurrent Data Fetching**: The `generate-sample` command uses goroutines and channels to efficiently fetch large amounts of data from the API in concurrent, time-based chunks, preventing timeouts.

* **Real-time Progress Bar**: Visual feedback for long-running operations using a progress bar.

![progress](/img/progress-bar.png)

* **Context-Aware Autocompletion**: Smart command suggestions that offer filenames when using the `read` command.
* **Flexible Output**: Save API results as pretty-printed JSON, CSV, or both.

![save](/img/save-output.png)

* **Built-in File Viewer**: Use the `read` command to view the contents of saved `.json` and `.csv` files without leaving the application.

![readfiles](/img/read-files.png)

## Getting Started

### Prerequisites

* Go version 1.24 or later.
* A C compiler (like GCC or Clang) for some dependencies.

### Installation & Running

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/mikey-p-codes/bforeCLI.git
    cd bforeCLI
    ```

2.  **Install dependencies:**
    This command will download the necessary libraries (`go-prompt`, `progressbar`).
    ```bash
    go mod tidy
    ```

3.  **Build the application:**
    For the best experience (especially for the `clear` command and progress bar to work correctly), build the executable first.
    ```bash
    go build -o bfore-ai-cli .
    ```

4.  **Run the application:**
    Execute the compiled binary from your terminal.
    * On macOS/Linux: `./bfore-ai-cli`
    * On Windows: `.\bfore-ai-cli.exe`

## Usage

Once the application is running, you will be greeted by the ASCII banner and the `$` prompt. Type `help` to see a list of available commands.

### Commands

* `login`
  Prompts for a username and password to authenticate with the API and store a session token. The command prompt will update to show your username.

* `logout`
  Clears the current session token and username, logging you out. The command prompt will revert to the default.

* `generate-sample`
  Initiates a long-running process to fetch domain data from the API between a specified start and end time. It makes concurrent requests in 30 second intervals to avoid timeouts and shows a progress bar. Once complete, it sorts the data and prompts you to save it as a JSON or CSV file.

* `read <filename>`
  Displays the contents of a local file. This command requires you to be logged in. If the file is a `.json` file, it will be pretty-printed.
    * *Usage:* `read my_data.json`

* `clear`
  Clears the terminal screen. __[Currently not working]__

* `help`
  Displays the list of available commands and their descriptions.

* `exit`
  Exits the CLI application.
   