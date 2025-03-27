package main

import (
	"database/sql"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type Collection struct {
	Id    int
	Name  string
	Items []Item
}

type Item struct {
	Id    int
	Image string
	Name  string
}

type Rank struct {
	Position int
	Name     string
	Wins     int
	Losses   int
}

type Matchup struct {
	FirstItem  string
	SecondItem string
	Winner     string
}

var (
	db  *sql.DB
	mu  sync.Mutex
	tpl *template.Template
)

func main() {
	initDB()
	defer db.Close()

	parseTemplates()

	mux := http.NewServeMux()
	registerRoutes(mux)

	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Starting server on http://localhost:8080")
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("main: %v", err)
	}
}

func initDB() {
	const setupQuery = `
	CREATE TABLE IF NOT EXISTS collections (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER NOT NULL PRIMARY KEY,
		image TEXT,
		name TEXT NOT NULL,
		collection_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS rankings (
		id INTEGER NOT NULL PRIMARY KEY,
		user_id TEXT,
		first_item_id INTEGER NOT NULL,
		second_item_id INTEGER NOT NULL,
		winner_id INTEGER DEFAULT NULL,
		collection_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (first_item_id) REFERENCES items(id) ON DELETE CASCADE,
		FOREIGN KEY (second_item_id) REFERENCES items(id) ON DELETE CASCADE,
		FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
	);`

	var err error
	db, err = sql.Open("sqlite", "test.db")
	if err != nil {
		log.Fatalf("initDB: %v", err)
	}

	_, err = db.Exec(setupQuery)
	if err != nil {
		log.Fatalf("initDB: %v", err)
	}
}

func parseTemplates() {
	tmp := template.New("")

	funcs := template.FuncMap{
		"embed": func(name string, data any) template.HTML {
			var out strings.Builder
			if err := tmp.ExecuteTemplate(&out, name, data); err != nil {
				log.Fatalf("parseTemplates: %v", err)
			}
			return template.HTML(out.String())
		}}

	tpl = template.Must(tmp.Funcs(funcs).ParseGlob("template/*.html"))
}

// func renderTemplate(w http.ResponseWriter, pageName string, data any) {
// 	tpl.ExecuteTemplate(w, "base.html", map[string]any{
// 		"Page":   pageName + ".html",
// 		"Script": "static/js/" + pageName + ".js",
// 		"Data":   data,
// 	})
// }

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/static/", serveStaticFiles)

	mux.HandleFunc("/", serveErrorPage)
	mux.HandleFunc("GET /{$}", serveHomePage)
	mux.HandleFunc("GET /collection/{id}/{$}", serveCollectionPage)
	mux.HandleFunc("GET /rank/{id}/{$}", serveRankPage)
	mux.HandleFunc("GET /results/{id}/{$}", serveResultsPage)

	mux.HandleFunc("POST /rank/update", updateRank)
	mux.HandleFunc("POST /collection/create", createCollection)
	mux.HandleFunc("POST /collection/delete", deleteCollection)
	mux.HandleFunc("POST /item/create", createItem)
	mux.HandleFunc("POST /item/delete", deleteItem)
}

func serveStaticFiles(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[1:]
	if in, err := os.Stat(filePath); err != nil || in.IsDir() {
		serveErrorPage(w, r)
		return
	}
	http.ServeFile(w, r, filePath)
}

func serveErrorPage(w http.ResponseWriter, r *http.Request) {
	parseTemplates()
	w.WriteHeader(http.StatusNotFound)
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":   "error.html",
		"Errors": []string{"Error on the wall", "Here we are again"},
	})
}

// func handleRoot(w http.ResponseWriter, r *http.Request) {
// 	http.Redirect(w, r, "/home", http.StatusFound)
// }

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":        "home.html",
		"Collections": getCollectionsWithItems(),
	})
}

func serveCollectionPage(w http.ResponseWriter, r *http.Request) {
	collectionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Print("invalid collection id")
		return
	}
	parseTemplates()
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":       "collections.html",
		"Collection": getCollectionWithItems(collectionId),
	})
}

func serveRankPage(w http.ResponseWriter, r *http.Request) {
	userId := r.RemoteAddr
	mu.Lock()
	defer mu.Unlock()

	collectionIdStr := r.PathValue("id")
	collectionId, err := strconv.Atoi(collectionIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // don't like it
		return
	}

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM rankings WHERE user_id = ? AND collection_id = ?)", userId, collectionId).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !exists {
		pairs := generatePairs(getCollectionItems(collectionId))
		shufflePairs(pairs)
		for _, pair := range pairs {
			_, err := db.Exec("INSERT INTO rankings (user_id, first_item_id, second_item_id, collection_id) VALUES (?, ?, ?, ?)", userId, pair[0], pair[1], collectionId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	pairIds := []int{0, 0}
	err = db.QueryRow("SELECT first_item_id, second_item_id FROM rankings WHERE user_id = ? AND collection_id = ? AND winner_id IS NULL", userId, collectionId).Scan(&pairIds[0], &pairIds[1])
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/results/"+collectionIdStr, http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":         "rank.html",
		"CollectionId": collectionId,
		"Pair":         getItemsById(pairIds),
	})
}

func updateRank(w http.ResponseWriter, r *http.Request) {
	userId := r.RemoteAddr
	mu.Lock()
	defer mu.Unlock()

	firstId, err := strconv.Atoi(r.PostFormValue("first_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	secondId, err := strconv.Atoi(r.PostFormValue("second_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	winnerId, err := strconv.Atoi(r.PostFormValue("winner_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	collectionIdStr := r.PostFormValue("collection_id")
	collectionId, err := strconv.Atoi(collectionIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// injection
	// take form value
	_, err = db.Exec("UPDATE rankings SET winner_id = ? WHERE user_id = ? AND first_item_id = ? AND second_item_id = ? AND collection_id = ? AND winner_id IS NULL", winnerId, userId, firstId, secondId, collectionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/rank/"+collectionIdStr, http.StatusSeeOther)
}

func serveResultsPage(w http.ResponseWriter, r *http.Request) {
	userId := r.RemoteAddr

	collectionIdStr := r.PathValue("id")

	collectionId, err := strconv.Atoi(collectionIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var hasUnresolvedMatchup bool
	err = db.QueryRow("SELECT(COUNT(*) = 0 OR COUNT(*) > COUNT(winner_id)) FROM rankings WHERE user_id = ? AND collection_id = ?;", userId, collectionId).Scan(&hasUnresolvedMatchup)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// stop attempting to rank empty collections

	if hasUnresolvedMatchup {
		http.Redirect(w, r, "/rank/"+collectionIdStr, http.StatusSeeOther)
		return
	}

	rankRows, err := db.Query("SELECT i.name, SUM(CASE WHEN winner_id = item_id THEN 1 ELSE 0 END) AS wins, SUM(CASE WHEN winner_id <> item_id THEN 1 ELSE 0 END) AS losses FROM( SELECT first_item_id AS item_id, winner_id FROM rankings WHERE collection_id = ? AND user_id = ? UNION ALL SELECT second_item_id AS item_id, winner_id FROM rankings WHERE collection_id = ? AND user_id = ?) AS r LEFT JOIN items i ON i.id = r.item_id GROUP BY i.id, i.name ORDER BY wins DESC;", collectionId, userId, collectionId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rankRows.Close()

	var ranks []Rank

	for rankRows.Next() {
		var rank Rank
		if err := rankRows.Scan(&rank.Name, &rank.Wins, &rank.Losses); err != nil {
			log.Print(err)
			continue
		}
		rank.Position = len(ranks) + 1
		ranks = append(ranks, rank)
	}

	matchupRows, err := db.Query("SELECT i1.name, i2.name, i_winner.name FROM rankings r JOIN items i1 ON r.first_item_id = i1.id JOIN items i2 ON r.second_item_id = i2.id JOIN items i_winner ON r.winner_id = i_winner.id WHERE r.collection_id = ? and r.user_id = ?;", collectionId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer matchupRows.Close()

	var matchups []Matchup

	for matchupRows.Next() {
		var matchup Matchup
		if err := matchupRows.Scan(&matchup.FirstItem, &matchup.SecondItem, &matchup.Winner); err != nil {
			log.Print(err)
			continue
		}
		matchups = append(matchups, matchup)
	}

	for i := len(ranks) - 1; i > 1; i-- {
		if ranks[i].Wins != ranks[i-1].Wins {
			continue
		}
		for _, matchup := range matchups {
			if (matchup.Winner == ranks[i].Name) && (matchup.FirstItem == ranks[i-1].Name || matchup.SecondItem == ranks[i-1].Name) {
				ranks[i], ranks[i-1] = ranks[i-1], ranks[i]
			}
		}
	}

	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":     "results.html",
		"Ranks":    ranks,
		"Matchups": matchups,
	})
}

func createCollection(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	if name == "" {
		log.Print("empty collection name provided")
	} else {

		_, err := db.Exec("INSERT INTO collections (name) VALUES (?)", name)
		if err != nil {
			log.Print("failed to create collection")
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteCollection(w http.ResponseWriter, r *http.Request) {
	_, err := db.Exec("DELETE FROM collections WHERE id >= ?", 7)
	if err != nil {
		log.Print("couldn't do it,", err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	collectionIdStr := r.PostFormValue("collection_id")
	collectionId, err := strconv.Atoi(collectionIdStr)
	if err != nil {
		log.Print("failed to create collection item")
		return
	}

	_, err = db.Exec("INSERT INTO items (name, image, collection_id) VALUES (?,?,?)", name, "", collectionId)
	if err != nil {
		log.Print("failed to create collection item")
		return
	}

	http.Redirect(w, r, "/collection/"+collectionIdStr, http.StatusSeeOther)
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	collectionIdStr := r.PostFormValue("collection_id")

	_, err := db.Exec("DELETE FROM items WHERE id >= ?", 86)
	if err != nil {
		log.Print("couldn't do it,", err)
		return
	}
	http.Redirect(w, r, "/collection/"+collectionIdStr, http.StatusSeeOther)
}

func getItemsById(ids []int) []Item {
	var items []Item
	for _, id := range ids {
		var item Item
		err := db.QueryRow("SELECT id, image, name FROM items WHERE id = ?", id).Scan(&item.Id, &item.Image, &item.Name)
		if err != nil {
			return items
		}
		items = append(items, item)
	}
	return items
}

func generatePairs(items []Item) [][]int {
	var pairs [][]int
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			pairs = append(pairs, []int{items[i].Id, items[j].Id})
		}
	}
	return pairs
}

func shufflePairs(pairs [][]int) {

	rand.Shuffle(len(pairs), func(i, j int) {
		pairs[i], pairs[j] = pairs[j], pairs[i]

		if rand.Intn(2) == 0 {
			pairs[i][0], pairs[i][1] = pairs[i][1], pairs[i][0]
		}

		if rand.Intn(2) == 0 {
			pairs[j][0], pairs[j][1] = pairs[j][1], pairs[j][0]
		}
	})
}

func getCollectionWithItems(id int) Collection {
	var col Collection

	err := db.QueryRow("SELECT name FROM collections WHERE id = ?", id).Scan(&col.Name)
	if err != nil {
		log.Print(err)
		return col
	}
	col.Id, col.Items = id, getCollectionItems(id)
	return col
}

func getCollectionsWithItems() []Collection {
	var cols []Collection

	rows, err := db.Query("SELECT id, name FROM collections")
	if err != nil {
		log.Print(err)
		return cols
	}
	defer rows.Close()

	for rows.Next() {
		var col Collection
		if err := rows.Scan(&col.Id, &col.Name); err != nil {
			log.Print(err)
			continue
		}
		col.Items = getCollectionItems(col.Id)
		cols = append(cols, col)
	}

	if err := rows.Err(); err != nil {
		log.Print(err)
	}

	return cols
}

func getCollectionItems(id int) []Item {
	var items []Item

	rows, err := db.Query("SELECT id, image, name FROM items WHERE collection_id = ?", id)
	if err != nil {
		log.Print(err)
		return items
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Id, &item.Image, &item.Name); err != nil {
			log.Print(err)
			continue
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Print(err)
	}

	return items
}
