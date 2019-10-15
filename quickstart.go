package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
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

func createFile(name string, data []byte) {
	f, err := os.Create(name)
	if err != nil {
		log.Fatalf("Unable to create file %s: %v", name, err)
	}
	defer f.Close()
	n3, err := f.Write(data)
	fmt.Printf("wrote %d bytes\n", n3)
	f.Sync()
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	const mimeTypePdf = "application/pdf"
	const parentDirectory = "0B-VJpOQeezDjZktuTnlEMEpGMUU"
	const targetDir = "/home/fjammes/src/k8s-school-www/content/pdf"

	r, err := srv.Files.List().
		Q("\"" + parentDirectory + "\" in parents and trashed=false and mimeType != 'application/vnd.google-apps.folder'").Fields("files(id,name,parents,mimeType)").Do()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	for _, i := range r.Files {
		r, err := srv.Files.Get(i.Parents[0]).Fields("name").Do()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		log.Printf("FileID=%s, Filename=%s, FolderName=%s MimeType=%s\n", i.Id, i.Name, r.Name, i.MimeType)

		var data []byte
		var res *http.Response
		var outFileName string

		if i.MimeType == "application/vnd.google-apps.form" {
			log.Printf("Excluding filename=%s, MimeType=%s\n", i.Name, i.MimeType)
		} else if i.MimeType != mimeTypePdf {
			res, err = srv.Files.Export(i.Id, mimeTypePdf).Download()
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
			outFileName = fmt.Sprintf("%s/%s.pdf", targetDir, i.Name)
		} else {
			res, err = srv.Files.Get(i.Id).Download()
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
			outFileName = fmt.Sprintf("%s/%s", targetDir, i.Name)
		}

		data, err = ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		createFile(outFileName, data)
	}
}
