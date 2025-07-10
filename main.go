// main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/schollz/progressbar/v3"
)

// apiToken still holds our authentication state for the session
var apiToken string
var apiUser string

// --- Struct Definitions from v1 of program

type APIResponse struct {
	Reports []Report `json:"Reports"`
}
type Report struct {
	Certificate Certificate `json:"Certificate"`
	Created     string      `json:"Created"`
	ID          string      `json:"Id"`
	Records     []Record    `json:"Records"`
}
type Certificate struct {
	Issuer     string `json:"Issuer"`
	NotAfter   string `json:"NotAfter"`
	NotBefore  string `json:"NotBefore"`
	Subject    string `json:"Subject"`
	Thumbprint string `json:"Thumbprint"`
}
type Record struct {
	Address    string   `json:"Address,omitempty"`
	DomainName string   `json:"DomainName"`
	RecordType string   `json:"RecordType"`
	Server     string   `json:"Server,omitempty"`
	Texts      []string `json:"Texts,omitempty"`
}

type PreCrime struct {
	Domains []Domains `json:"Domains"`
}

type Domains struct {
	Id            int     `json:"Id"`
	Name          string  `json:"Name"`
	DomainCreated string  `json:"DomainCreated"`
	ScoreCreated  string  `json:"ScoreCreated"`
	Score         float64 `json:"Score"`
}

func displayBanner() {
	banner := `
    ▒▒▒▒▒▒▒▒▒▒                                              
    ▒    ▒    ▒    ████    ██                     ██    █   
    ▒   ▒ ▒   ▒    █   █ ████  ███   █ ██ ████    ███   █   
    ░ ▒▒    ▒▒     █████  ██  █    █ ██  █    █  █  █   █   
    ░░  ▒░▒▒  ▒    █   ██ ██  █    █ ██  █      ██████  █   
    ░    ░    ▒    █████  ██   ████  ██   ████  █    ██ █   
    ░░░░░░░▒▒▒
`
	fmt.Println(banner)
}

// executor to make the CLI a CLI
func executor(in string) {
	in = strings.TrimSpace(in)
	if in == "" {
		return
	}
	parts := strings.Split(in, " ")
	command := parts[0]
	args := parts[1:]

	// A map to hold our command handlers, makes code clean and scalable
	commandHandlers := map[string]func([]string){
		"login":       handleLoginCmd,
		"logout":      handleLogoutCmd,
		"clear":       handleClearCmd,
		"read":        handleReadCmd,
		"show":        handleshowcmd,
		"domain-info": handleDomainInfoCmd,
		//	"domain-scores":   handleDomainScoresCmd,
		"generate-sample": handleGenerateSampleCmd,
		"help":            handleHelpCmd,
	}
	// Execute the command if it exists.
	if handler, ok := commandHandlers[command]; ok {
		handler(args)
	} else if command == "exit" {
		// Special case for exiting the application
		fmt.Println("Bye!")
		os.Exit(0)
	} else if command != "" {
		fmt.Printf("Unknown command: '%s'. Type 'help' for a list of supported commands.\n", command)
	}
}
func fileCompleter(d prompt.Document) []prompt.Suggest {
	files, err := os.ReadDir(".")
	if err != nil {
		return nil
	}

	suggestions := make([]prompt.Suggest, 0)
	for _, f := range files {
		if !f.IsDir() {
			ext := filepath.Ext(f.Name())
			if ext == ".json" || ext == ".csv" {
				suggestions = append(suggestions, prompt.Suggest{Text: f.Name()})
			}
		}
	}
	return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
}

// completer provides suggestions as the user types
func completer(d prompt.Document) []prompt.Suggest {
	if strings.HasPrefix(d.Text, "read ") {
		if apiUser != "" {
			return fileCompleter(d)
		}
		return nil
	}
	s := []prompt.Suggest{
		{Text: "login", Description: "Authenticate with the BforeAI API with  your username and password."},
		{Text: "logout", Description: "Logout from BforeAI API."},
		{Text: "show", Description: "Display Username and Token"},
		{Text: "read", Description: "Read a local .json or .csv file"},
		{Text: "clear", Description: "Clear the terminal screen"},
		{Text: "domain-info", Description: "Get detailed information about the domain."},
		{Text: "domain-scores", Description: "Get domain PreCrime score information."},
		{Text: "generate-sample", Description: "Generate a sample JSON or CSV file."},
		{Text: "help", Description: "Show this help message."},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

// create live prefix
func getLivePrefix() (string, bool) {
	if apiUser != "" {
		return fmt.Sprintf("%s-cli $ ", apiUser), true
	}
	return "$ ", false
}

// handleHelpCmd prints a list of available commands
func handleHelpCmd(args []string) {
	fmt.Println("Available commands:")
	fmt.Println("	login			- Authenticate with the BforeAI API with  your username and password.")
	fmt.Println("	show			- Show authenticated user information.")
	fmt.Println("  read <filename>      - Read a local .json or .csv file")
	fmt.Println("  clear                - Clear the terminal screen")
	fmt.Println("	domain-info		- Get detailed information about the domain.")
	fmt.Println("	domain-scores	- Get domain PreCrime score information.")
	fmt.Println("generate-sample		- Generate a sample JSON or CSV file.")
	fmt.Println("	help			- Show this help message.")
	fmt.Println("	exit			- Exit the application.")
}

// handleClearCmd clears the console
func handleClearCmd(args []string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd = exec.Command("clear", "cls")
	case "windows":
		cmd = exec.Command("clear", "cls")
	default:
		return
	}
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return
	}
}

// handleReadCmd read a file and print to console
func handleReadCmd(args []string) {
	if apiUser == "" {
		fmt.Println("\nWarning: you must be authenticated to read the file system.")
		return
	}

	if len(args) == 0 {
		fmt.Println("Usage: read <filename>")
		return
	}
	filename := args[0]

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file '%s': %v\n", filename, err)
		return
	}

	fmt.Printf("\n--- Content of %s ---\n", filename)
	ext := filepath.Ext(filename)
	if ext == ".json" {
		var prettyJSON bytes.Buffer
		if json.Indent(&prettyJSON, content, "", "  ") == nil {
			fmt.Println(prettyJSON.String())
		} else {
			// If it's not valid JSON, just print as plain text.
			fmt.Println(string(content))
		}
	} else {
		// For .csv and any other file type, print as plain text.
		fmt.Println(string(content))
	}
	fmt.Printf("--- End of %s ---\n", filename)
}

// handleLoginCmd is our new login logic, now as a command
func handleLoginCmd(args []string) {
	username := prompt.Input("Enter username: ", completer)
	password := prompt.Input("Enter password: ", completer, prompt.OptionPrefixTextColor(prompt.Red))

	// same login logic previously
	loginData := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
	}
	payload, err := json.Marshal(loginData)
	if err != nil {
		fmt.Println("Error creating JSON payload:", err)
		return
	}

	loginURL := "https://api.bfore.ai/user/login"
	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request to API:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing body:", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Printf("Login failed. Status: %s. Response: %s\n", resp.Status, string(body))
		return
	}
	var loginResponse struct {
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &loginResponse); err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return
	}

	if loginResponse.Token == "" {
		fmt.Println("Login successful, but no token provided.")
		return
	}

	apiToken = loginResponse.Token
	apiUser = loginResponse.Username
	fmt.Println("\nSuccessfully authenticated! Token has been stored for this session.")
}

func handleshowcmd(args []string) {
	fmt.Printf("Current User Information: %s\n", apiUser)
	fmt.Printf("Current Token: 			 %s\n", apiToken)
}

func printRecordsToScreen(records []Record) {
	fmt.Println("\n--- API Results ---")
	for i, r := range records {
		fmt.Printf("Record %d:\n", i+1)
		fmt.Printf("  Domain Name: %s\n", r.DomainName)
		fmt.Printf("  Record Type: %s\n", r.RecordType)
		if r.Address != "" {
			fmt.Printf("  Address:     %s\n", r.Address)
		}
		if r.Server != "" {
			fmt.Printf("  Server:      %s\n", r.Server)
		}
		if len(r.Texts) > 0 {
			fmt.Printf("  Texts:       %s\n", strings.Join(r.Texts, "; "))
		}
		fmt.Println("-------------------")
	}
}

// handleDomainInfocmd
func handleDomainInfoCmd(args []string) {
	if apiToken == "" {
		fmt.Println("No API token stored, please login again to store your token.")
		return
	}
	// interactive prompting is the same
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter domain: ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)

	fmt.Print("Include Screenshot?: ")
	screenshot, _ := reader.ReadString('\n')
	screenshot = strings.TrimSpace(screenshot)

	fmt.Print("Include whois?: ")
	whois, _ := reader.ReadString('\n')
	whois = strings.TrimSpace(whois)

	fmt.Print("Include DNS?: ")
	dns, _ := reader.ReadString('\n')
	dns = strings.TrimSpace(dns)

	fmt.Print("Include Certificate Details?: ")
	certificate, _ := reader.ReadString('\n')
	certificate = strings.TrimSpace(certificate)

	// Help for time format
	iso8601Format := "2006-01-02T15:04:05-0700"
	fmt.Printf("Enter start time (ISO8601 format, e.g., %s): ", time.Now().UTC().Format(iso8601Format))

	startTime, _ := reader.ReadString('\n')
	startTime = strings.TrimSpace(startTime)

	fmt.Printf("Enter end time (ISO8601 format, e.g., %s): ", time.Now().UTC().Add(24*time.Hour).Format(iso8601Format))
	endTime, _ := reader.ReadString('\n')
	endTime = strings.TrimSpace(endTime)

	baseURL := "https://api.bfore.ai/report/list"
	params := url.Values{}
	params.Add("d", domain)
	params.Add("c", certificate)
	params.Add("w", whois)
	params.Add("n", dns)
	params.Add("s", screenshot)
	params.Add("st", startTime)
	params.Add("en", endTime)

	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	// retrieve data
	fmt.Println("\nRequesting data from:", fullURL)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request to API:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("API request failed.  Status: %s. Response: %s\n", resp.Status, string(body))
	}
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		fmt.Println("\n--- Error failed to parse the JSON. ---")
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "  ")
		if err != nil {
			return
		}
		fmt.Println(prettyJSON.String())
		return
	}

	var allRecords []Record
	for _, report := range apiResponse.Reports {
		allRecords = append(allRecords, report.Records...)
	}

	fmt.Printf("Successfully retrieved %d total records from %d report.\n", len(allRecords), len(apiResponse.Reports))
	if len(allRecords) == 0 {
		return
	}
	fmt.Print("Would you like to save the results? (Y/n): ")
	saveResult, _ := reader.ReadString('\n')
	saveResult = strings.ToLower(strings.TrimSpace(saveResult))

	// If user enters 'n', print to screen and stop. Otherwise, proceed to save.
	if saveResult == "n" || saveResult == "no" {
		printRecordsToScreen(allRecords)
		return
	}

	// file saving logic
	fmt.Print("Enter base filename for output (e.g., domain_data): ")
	filenameBase, _ := reader.ReadString('\n')
	filenameBase = strings.TrimSpace(filenameBase)

	fmt.Print("Choose output format (json, csv, both): ")
	format, _ := reader.ReadString('\n')
	format = strings.ToLower(strings.TrimSpace(format))

	if format == "json" || format == "both" {
		prettyJSON, err := json.MarshalIndent(allRecords, "", "  ")
		if err != nil {
			fmt.Println("Error formatting JSON:", err)
		} else {
			filename := filenameBase + ".json"
			err = os.WriteFile(filename, prettyJSON, 0644)
			if err != nil {
				fmt.Println("Error writing JSON file:", err)
			} else {
				fmt.Println("Successfully saved data to", filename)
			}
		}
	}

	if format == "csv" || format == "both" {
		filename := filenameBase + ".csv"
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error creating CSV file:", err)
		} else {
			defer func(file *os.File) {
				err := file.Close()
				if err != nil {

				}
			}(file)
			writer := csv.NewWriter(file)
			defer writer.Flush()

			// Write CSV header
			header := []string{"DomainName", "RecordType", "Address", "Server", "Texts"}
			err := writer.Write(header)
			if err != nil {
				return
			}

			// Loop through our collected records and write to the file
			for _, r := range allRecords {
				row := []string{r.DomainName, r.RecordType, r.Address, r.Server, strings.Join(r.Texts, ", ")}
				err := writer.Write(row)
				if err != nil {
					return
				}
			}
			fmt.Println("Successfully saved data to", filename)
		}
	}
}

// handleGenerateSampleCmd fetches data in chunks using goroutines.
func handleGenerateSampleCmd(args []string) {
	if apiToken == "" {
		fmt.Println("\nError: You must be logged in. Please run the 'login' command first.")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	const iso8601Format = "2006-01-02T15:04:00" // Use Z for UTC as is standard

	fmt.Print("How many records per request would you like to see?: ")
	records, _ := reader.ReadString('\n')
	records = strings.TrimSpace(records)

	fmt.Print("What is the minimum score you'd like to see?: ")
	minScore, _ := reader.ReadString('\n')
	minScore = strings.TrimSpace(minScore)

	fmt.Printf("Enter start time (YYYY-MM-DDTHH:MM:SS): ")
	startStr, _ := reader.ReadString('\n')
	startStr = strings.TrimSpace(startStr)
	startTime, err := time.Parse(iso8601Format, startStr)
	if err != nil {
		fmt.Println("Invalid start time format. Please use YYYY-MM-DDTHH:MM:SS")
		return
	}

	fmt.Printf("Enter end time (YYYY-MM-DDTHH:MM:SS): ")
	endStr, _ := reader.ReadString('\n')
	endStr = strings.TrimSpace(endStr)
	endTime, err := time.Parse(iso8601Format, endStr)
	if err != nil {
		fmt.Println("Invalid end time format. Please use YYYY-MM-DDTHH:MM:SS")
		return
	}

	var allDomains []Domains
	var wg sync.WaitGroup
	domainChan := make(chan []Domains)
	// Progress Bar
	interval := 30 * time.Minute
	totalDuration := endTime.Sub(startTime)
	totalSteps := int(math.Ceil(totalDuration.Minutes() / interval.Minutes()))
	bar := progressbar.Default(int64(totalSteps), "Fetching data")

	fmt.Println("\nStarting sample generation...")
	// Loop from startTime to endTime in 30-minute intervals.
	for t := startTime; t.Before(endTime); t = t.Add(30 * time.Minute) {
		wg.Add(1)
		go func(start, end time.Time) {
			defer wg.Done()
			defer bar.Add(1)

			baseURL := "https://api.bfore.ai/domain/list"
			// Manually build the URL to enforce parameter order.
			fullURL := fmt.Sprintf("%s?c=%s&d=%s&s=%s&e=%s",
				baseURL,
				url.QueryEscape(records),
				url.QueryEscape(minScore),
				url.QueryEscape(start.Format(iso8601Format)),
				url.QueryEscape(end.Format(iso8601Format)),
			)

			req, _ := http.NewRequest("GET", fullURL, nil)
			req.Header.Set("Authorization", "Bearer "+apiToken)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error fetching data for %s: %v\n", start.Format(iso8601Format), err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				// The API returns a flat array, so we unmarshal into a slice of Domains.
				var domainsResponse []Domains
				if json.Unmarshal(body, &domainsResponse) == nil {
					if len(domainsResponse) > 0 {
						domainChan <- domainsResponse
					}
				}
			}
		}(t, t.Add(30*time.Minute))
	}

	go func() {
		wg.Wait()
		close(domainChan)
	}()

	for domains := range domainChan {
		allDomains = append(allDomains, domains...)
	}

	fmt.Printf("\nFinished sample generation. Found %d total domains.\n", len(allDomains))
	if len(allDomains) == 0 {
		return
	}

	// Sort the collected domains by the ScoreCreated timestamp.
	sort.Slice(allDomains, func(i, j int) bool {
		return allDomains[i].ScoreCreated < allDomains[j].ScoreCreated
	})
	fmt.Println("All domains Sorted by Score Creation Date.")

	// Ask to save or print the final results.
	fmt.Print("Would you like to save the results? (Y/n): ")
	saveResult, _ := reader.ReadString('\n')
	saveResult = strings.ToLower(strings.TrimSpace(saveResult))

	if saveResult == "n" {
		printDomainsToScreen(allDomains)
	} else {
		saveDomains(allDomains, reader)
	}
}

// printDomainsToScreen formats and prints the domain results to the console.
func printDomainsToScreen(domains []Domains) {
	fmt.Println("\n======= API Results ===============================")
	for i, d := range domains {
		fmt.Printf("	 Result 			%d:\n", i+1)
		fmt.Printf("  ID:     			%d\n", d.Id)
		fmt.Printf("  Name:    			%s\n", d.Name)
		fmt.Printf("  Score Created:   	%s\n", d.ScoreCreated)
		fmt.Printf("  Domain Created: 	%s\n", d.DomainCreated)
		fmt.Printf("Score: 			%f\n", d.Score)
		fmt.Println("================================================")
	}
}

// saveDomains handles the logic for saving domain results to a file.
func saveDomains(domains []Domains, reader *bufio.Reader) {
	fmt.Print("Enter base filename for output (e.g., domain_data): ")
	filenameBase, _ := reader.ReadString('\n')
	filenameBase = strings.TrimSpace(filenameBase)

	fmt.Print("Choose output format (json, csv, both): ")
	format, _ := reader.ReadString('\n')
	format = strings.ToLower(strings.TrimSpace(format))

	if format == "json" || format == "both" {
		// Use the PreCrime struct to match the expected JSON output structure
		outputData := PreCrime{Domains: domains}
		prettyJSON, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			fmt.Println("Error formatting JSON:", err)
		} else {
			filename := filenameBase + ".json"
			if err := os.WriteFile(filename, prettyJSON, 0644); err != nil {
				fmt.Println("Error writing JSON file:", err)
			} else {
				fmt.Println("Successfully saved data to", filename)
			}
		}
	}

	if format == "csv" || format == "both" {
		filename := filenameBase + ".csv"
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error creating CSV file:", err)
			return
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write header row
		header := []string{"ID", "Name", "Score", "Created"}
		writer.Write(header)

		// Write data rows
		for _, d := range domains {
			row := []string{
				fmt.Sprintf("%d", d.Id),
				d.Name,
				fmt.Sprintf("%f", d.Score),
				d.DomainCreated,
			}
			writer.Write(row)
		}
		fmt.Println("Successfully saved data to", filename)
	}
}
func handleLogoutCmd(args []string) {
	if apiToken == "" {
		fmt.Println("You are not logged in.")
		return
	}
	apiToken = ""
	apiUser = ""
	fmt.Println("You have been logged out.")
}
func main() {
	handleClearCmd(nil)
	displayBanner()
	fmt.Println(" BforeAI CLI -- Type 'help' for commands, 'exit' to quit.")
	p := prompt.New(
		executor,
		completer,
		prompt.OptionTitle("bforeai-cli >"),
		prompt.OptionLivePrefix(getLivePrefix),
		prompt.OptionInputTextColor(prompt.Green),
		prompt.OptionSetExitCheckerOnInput(func(in string, breakline bool) bool {
			return in == "exit"
		}),
	)
	p.Run()
}
