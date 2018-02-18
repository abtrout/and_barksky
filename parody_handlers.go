package main

import (
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Parody struct {
	template *template.Template
	ids      []string
}

// Returns a random Imgur id from our collection.
func (p *Parody) RandomGif() string {
	index := rand.Intn(len(p.ids))
	return p.ids[index]
}

func (p *Parody) Index(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Check for most recently used coordinates in last_loc cookie. If
	// valid coordinates can be found, we'll redirect the user there now.
	lastLoc, _ := r.Cookie("last_loc")
	if lastLoc != nil {
		lat, lng, err := parseCoords(lastLoc.Value)
		if err == nil {
			forecastURL := fmt.Sprintf("/forecast/%f,%f", lat, lng)
			http.Redirect(w, r, forecastURL, 302)
		}
	}
	// Otherwise, the basic Index is rendered.
	p.template.Execute(w, map[string]interface{}{
		"ImgurID": p.RandomGif(),
	})
}

func (p *Parody) Search(client *maps.Client) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Parse the query.
		r.ParseForm()
		query := r.Form["query"][0]
		// Geocode from the query to get latitude and longitude.
		req := &maps.GeocodingRequest{Address: query}
		res, err := client.Geocode(context.Background(), req)
		if err != nil || len(res) == 0 {
			log.Println("Geocoding failed with error:", err)
			http.Redirect(w, r, "/search/failed", 303)
			return
		}
		latLng := res[0].Geometry.Location
		// Redirect to get the Forecast.
		redirectURL := fmt.Sprintf("/forecast/%.4f,%.4f", latLng.Lat, latLng.Lng)
		http.Redirect(w, r, redirectURL, 302)
	}
}

func (p *Parody) SearchFailed(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	p.template.Execute(w, map[string]interface{}{
		"ImgurID":        p.RandomGif(),
		"IsSearchFailed": true,
	})
}

func (p *Parody) Forecast(client *maps.Client) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Validate coordinates.
		lat, lng, err := parseCoords(ps.ByName("coords"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Invalid coordinates")
			return
		}
		// Turn coordinates into an address/name.
		req := &maps.GeocodingRequest{LatLng: &maps.LatLng{Lat: lat, Lng: lng}}
		res, err := client.ReverseGeocode(context.Background(), req)
		if err != nil || len(res) == 0 {
			log.Println("GeocodingRequest failed with error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Send a cookie so we can return to this location next time.
		http.SetCookie(w, &http.Cookie{
			Name:    "last_loc",
			Value:   fmt.Sprintf("%f,%f", lat, lng),
			Path:    "/",
			Expires: time.Now().Add(7 * 24 * time.Hour)})
		// Determine if the user has specified a preference for units. Currently
		// the possibilities are US and UK, though Forecast's embed supports more.
		var units string
		switch ps.ByName("units") {
		case "uk", "us":
			units = ps.ByName("units")
			http.SetCookie(w, &http.Cookie{
				Name:    "units_pref",
				Value:   units,
				Path:    "/",
				Expires: time.Now().Add(30 * 24 * time.Hour)})
		default:
			unitsPref, err := r.Cookie("units_pref")
			if err == nil && unitsPref.Value == "uk" {
				units = "uk"
			} else {
				units = "us"
			}
		}
		// Render the weather template with these values.
		p.template.Execute(w, map[string]interface{}{
			"ImgurID":      p.RandomGif(),
			"IsFrameReady": true,
			"Lat":          lat,
			"Lng":          lng,
			"Name":         res[0].FormattedAddress,
			"Units":        units,
		})
	}
}

func parseCoords(coords string) (float64, float64, error) {
	// Split provided coordinates into latitude and longitude.
	xs := strings.SplitN(coords, ",", 2)
	if len(xs) < 2 {
		return 0, 0, errors.New("Invalid coordinate format")
	}
	// Make sure they are valid float numbers.
	lat, latErr := strconv.ParseFloat(xs[0], 64)
	lng, lngErr := strconv.ParseFloat(xs[1], 64)
	// And that they map to actual points on Earth.
	ok := (-90 <= lat && lat <= 90) && (-180 <= lng && lng <= 180)
	if latErr != nil || lngErr != nil || !ok {
		return lat, lng, errors.New("Invalid coordinates")
	}
	return lat, lng, nil
}

// TODO: don't really need a separate method for this ...
func parseLastLoc(r *http.Request) (float64, float64, error) {
	lastLoc, err := r.Cookie("last_loc")
	if err != nil {
		return 0, 0, err
	}
	return parseCoords(lastLoc.Value)
}
