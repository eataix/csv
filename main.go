package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

const (
	tokFile = "token.json"
	csvFile = "file.csv"
)

func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func prepare(filename string) ([][]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("os.Open(%q) failed: %v", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	record, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reader.RealdAll() failed: %v", err)
	}

	var all [][]interface{}
	for _, value := range record {
		var row []interface{}
		for _, item := range value {
			row = append(row, item)
		}
		all = append(all, row)
	}
	return all, nil
}

func main() {
	ctx := context.Background()

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	creationResp, err := srv.Spreadsheets.Create(&sheets.Spreadsheet{
		NamedRanges: nil,
		Properties: &sheets.SpreadsheetProperties{
			Title: "Summary",
		},
	}).Context(ctx).Do()

	if err != nil {
		log.Fatalf("Failed to create a spreadsheet: %v", err)
	}
	log.Printf("Created %s", creationResp.SpreadsheetUrl)

	data, err := prepare(csvFile)
	if err != nil {
		log.Fatalf("Failed to read the csv file: %v", err)
	}

	updateResp, err := srv.Spreadsheets.Values.BatchUpdate(creationResp.SpreadsheetId,
		&sheets.BatchUpdateValuesRequest{
			ValueInputOption: "RAW",
			Data: []*sheets.ValueRange{{
				Range:  "A1",
				Values: data,
			}},
		}).Context(ctx).Do()

	if err != nil {
		log.Fatalf("BatchUpdate() failed: %v", err)
	}

	fmt.Printf("%#v\n", updateResp)
}
