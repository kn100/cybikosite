package cydata

import (
	"fmt"
	"html"
	"io/ioutil"
	"net/url"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

const PageSize = 32

type CyFile struct {
	RefID            int
	Name             string
	FilePath         string
	ExtractionMethod string
	OriginalType     string
}

type Game struct {
	Name          string            `json:"name"`
	FolderName    string            `json:"folder_name"`
	CyFiles       map[string]CyFile `json:"files"`
	IntroImg      string            `json:"intro_img"`
	IconImg       string            `json:"icon_img"`
	Text          string            `json:"text"`
	TextShortened string            `json:"text_shortened"`
	PageNumber    int               `json:"page_number"`
	TweetString   string            `json:"tweet_string"`
}

type CyData struct {
	Games []Game `json:"games"`
}

func (cy *CyData) Length() int {
	return len(cy.Games)
}

func (cy *CyData) GetGames(gameNames []string) ([]Game, error) {
	games := []Game{}
	for _, gameName := range gameNames {
		for _, game := range cy.Games {
			if game.FolderName == gameName {
				games = append(games, game)
			}
		}
	}
	return games, nil
}

func (cy *CyData) Load() error {
	Games := []Game{}
	cy.Games = Games

	foldersWhichExist, err := foldersWhichExist()
	if err != nil {
		panic(err)
	}
	for i, folderName := range foldersWhichExist {

		// Create the game
		game := Game{
			Name:       makeTitle(folderName),
			FolderName: folderName,
			CyFiles:    map[string]CyFile{},
		}

		// Load in the files
		files, err := ioutil.ReadDir("/static/resources/" + folderName)
		if err != nil {
			return err
		}
		hasIntro := false
		hasIcon := false
		for i, file := range files {
			if file.IsDir() {
				continue
			}
			cyFile := CyFile{
				Name:     file.Name(),
				FilePath: "/public/resources/" + folderName + "/" + file.Name(),
				RefID:    1 + i,
			}
			if strings.HasSuffix(file.Name(), "-pic.png") {
				cyFile.ExtractionMethod = "Screenshot of PicView.exe from Cybiko SDK"
				cyFile.OriginalType = "PIC"
			} else if strings.HasSuffix(file.Name(), "-ico.png") {
				cyFile.ExtractionMethod = "Screenshot of PicView.exe from Cybiko SDK"
				cyFile.OriginalType = "ICO"
			} else if strings.HasSuffix(file.Name(), "-spl.txt") {
				cyFile.ExtractionMethod = "Text file converted from iso-8859-1 to utf-8 using iconv"
				cyFile.OriginalType = "SPL"
			} else if strings.HasSuffix(file.Name(), "-txt.txt") {
				cyFile.ExtractionMethod = "Text file converted from iso-8859-1 to utf-8 using iconv"
				cyFile.OriginalType = "TXT"
			} else {
				cyFile.ExtractionMethod = "Unknown"
				cyFile.OriginalType = "Unknown"
			}

			game.CyFiles[file.Name()] = cyFile
			if file.Name() == "root-spl.txt" {
				text, err := handleTextFile("/static/resources/" + folderName + "/" + file.Name())
				if err != nil {
					return err
				}
				game.Text = text
				game.TextShortened = shortenText(game.Text)

			}
			if file.Name() == "intro-pic.png" {
				hasIntro = true
			}
			if file.Name() == "root-ico.png" {
				hasIcon = true
			}
		}
		if hasIntro {
			game.IntroImg = "/public/resources/" + folderName + "/intro-pic.png"
		} else {
			game.IntroImg = "https://via.placeholder.com/160x100.png"
		}
		if hasIcon {
			game.IconImg = "/public/resources/" + folderName + "/root-ico.png"
		} else {
			game.IconImg = "https://via.placeholder.com/48x48.png"
		}
		game.PageNumber = i/PageSize + 1
		game.TweetString = url.QueryEscape(fmt.Sprintf("Remember %s? - It's a game released for the Cybiko Classic, a handheld from the early 2000! https://cybiko.kn100.me/game/%s #cybiko #retrogaming", game.Name, game.FolderName))
		// URL encode above string
		game.TweetString = strings.Replace(game.TweetString, " ", "%20", -1)
		log.Debug().
			Str("Game", game.Name).
			Int("Number of files", len(game.CyFiles)).
			Msg("Files scanned in")
		// Add the game to the list
		cy.Games = append(cy.Games, game)
	}
	return nil
}

func foldersWhichExist() ([]string, error) {
	files, err := ioutil.ReadDir("/static/resources/")
	if err != nil {
		return nil, err
	}

	foldersWhichExist := []string{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		foldersWhichExist = append(foldersWhichExist, file.Name())
	}

	sort.StringSlice(foldersWhichExist).Sort()
	log.Debug().
		Int("Number of games", len(foldersWhichExist)).
		Msg("Games scanned in")
	return foldersWhichExist, nil
}

func handleTextFile(path string) (string, error) {
	log.Debug().
		Str("Path", path).
		Msg("Handling Text file")
	text, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	fixedText := string(text)
	if fixedText == "" {
		return "No text found", nil
	}

	fixedText = html.EscapeString(fixedText)
	fixedText = strings.Replace(fixedText, "\n", "<br>", -1)
	fixedText = addFormatting(fixedText)
	return fixedText, nil
}

func addFormatting(text string) string {
	text = strings.Replace(text, "&lt;", "<kbd>&lt;", -1)
	text = strings.Replace(text, "&gt;", "&gt;</kbd>", -1)
	return text
}

func makeTitle(folderName string) string {
	return strings.Replace(strings.Title(folderName), "_", " ", -1)
}

func shortenText(text string) string {
	if len(text) > 280 {
		return text[:280] + "..."
	}
	return text
}

func (cy *CyData) GetGamesOnPage(pageNumber int) []Game {
	games := []Game{}
	// Check apge number in bounds
	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageNumber > cy.Length()/PageSize+1 {
		pageNumber = cy.Length()/PageSize + 1
	}
	for _, game := range cy.Games {
		if game.PageNumber == pageNumber {
			games = append(games, game)
		}
	}
	if len(games) == 0 {
		return nil
	}

	return games
}

func (cy *CyData) GetAllGames() []Game {
	return cy.Games
}
