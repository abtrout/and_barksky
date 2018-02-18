package main

import (
	"bufio"
	"flag"
	"github.com/julienschmidt/httprouter"
	"googlemaps.github.io/maps"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	// Parse configuration from flags.
	listenAddr := flag.String("listen", "127.0.0.1:8080", "Address to listen on")
	mapsKey := flag.String("mapsKey", "CHANGE_ME", "Google Maps API Key")
	templateFile := flag.String("template", "resources/forecats.html", "Path to parody template file")
	idsFile := flag.String("ids", "resources/cats.txt", "Path to file with Imgur ids for gif collection")
	flag.Parse()

	// Initialize Google Maps client.
	client, err := maps.NewClient(maps.WithAPIKey(*mapsKey))
	if err != nil {
		log.Fatal("Failed to initialize Google Maps client:", err)
	}

	// Parse template file and load Imgur gif ids into memory.
	template, err := template.ParseFiles(*templateFile)
	if err != nil {
		log.Fatal("Failed to parse template file:", err)
	}
	ids := parseIDs(*idsFile)

	// Initialize our Parody with the specified template and ids collection.
	p := &Parody{template, ids}
	// Bind our endpoints using this Parody. The /search and /forecast
	// endpoints require Google Maps client to resolve lat/lng coordinates.
	router := httprouter.New()
	router.GET("/", p.Index)
	router.POST("/search", p.Search(client))
	router.GET("/search/failed", p.SearchFailed)
	router.GET("/forecast/:coords", p.Forecast(client))
	router.GET("/forecast/:coords/:units", p.Forecast(client))

	// Listen and serve!
	log.Fatal(http.ListenAndServe(*listenAddr, router))
	log.Printf("Listening on %s", *listenAddr)
}

// Reads Imgur ids from the specified file. Each line should contain
// a single string id corresponding to a hopefully-still-valid Imgur URL.
func parseIDs(file string) []string {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("Failed to parseIDs from %s\n", file)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var ids []string
	for scanner.Scan() {
		ids = append(ids, scanner.Text())
	}
	log.Printf("Parsed %d ids from file %s\n", len(ids), file)
	return ids
}
