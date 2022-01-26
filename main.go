package main

import (
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/kn100/cyarchive/cydata"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type overviewResponse struct {
	Games []cydata.Game `json:"games"`
	Pages []page        `json:"total_pages"`
}

type detailResponse struct {
	Game cydata.Game `json:"game"`
}

type page struct {
	Page     int
	IsActive bool
	IsFirst  bool
	IsLast   bool
}

var cyData cydata.CyData

const PageLimit = 32

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	// Load in game list
	err := cyData.Load()
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}

	r := mux.NewRouter()
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)
	fs := http.FileServer(http.Dir("/static/"))
	r.PathPrefix("/public").Handler(http.StripPrefix("/public/", fs))
	r.HandleFunc("/", RedirToArchiveSite)
	r.HandleFunc("/page/{page}", GetTemplatedArchiveSiteHandler)
	r.Path("/game/{game}").HandlerFunc(GetTemplatedGameViewMorePageHandler)
	http.Handle("/", r)
	log.Fatal().Err(http.ListenAndServe("0.0.0.0:32768", handler)).Msg("Server stopped!!!")
}

func RedirToArchiveSite(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/page/1", http.StatusFound)
}

func GetTemplatedGameViewMorePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameName := vars["game"]
	log.Debug().
		Str("game", gameName).
		Msg("Getting game")
	games, err := cyData.GetGames([]string{gameName})
	if err != nil {
		// return 404
		http.Error(w, "Page not found", http.StatusNotFound)
		log.Error().Err(err).Msg(err.Error())
		return
	}
	if len(games) == 0 {
		// return 404
		http.Error(w, "Page not found", http.StatusNotFound)
		log.Error().Str("Requested", gameName).Msg("Requested game that doesn't exist")
		return
	}
	gr := detailResponse{Game: games[0]}

	t, err := template.ParseFiles("/templates/game.html")
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}
	t.Execute(w, gr)
}

func GetTemplatedArchiveSiteHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Getting archive site")
	limit := 0
	vars := mux.Vars(r)
	pageStr := vars["page"]
	pageInt := 1
	gwh := overviewResponse{}

	if pageStr == "all" {
		limit = len(cyData.Games) + 1
		gwh.Games = cyData.GetAllGames()
	} else {
		limit = PageLimit
		var err error
		pageInt, err = strconv.Atoi(pageStr)
		if err != nil {
			// return error 404
			log.Error().Err(err).Msg(err.Error())
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}
		gwh.Games = cyData.GetGamesOnPage(pageInt)
	}

	if gwh.Games == nil {
		log.Error().Msg("Games list empty for some reason")
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	gwh.Pages = make([]page, 0)
	for i := 1; i <= cyData.Length()/limit+1; i++ {
		p := page{Page: i, IsActive: false, IsFirst: false, IsLast: false}
		if i == 1 {
			p.IsFirst = true
		}
		if i == cyData.Length()/limit {
			p.IsLast = true
		}
		if i == pageInt {
			p.IsActive = true
		}
		gwh.Pages = append(gwh.Pages, p)
	}
	log.Debug().Msg("Getting templated archive site")
	w.Header().Set("Content-Type", "text/html")
	// Load in the template
	tmpl := template.Must(template.ParseFiles("/templates/index.html"))
	tmpl.Execute(w, gwh)
}

func GetArchiveSiteHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/static/index.html")
}
