package main

import (
	"database/sql"
	"html/template"
	"log"
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
	DROP TABLE rankings;
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
	mux.HandleFunc("GET /rank/{id}/{$}", serveRankPage)
	mux.HandleFunc("GET /results/{id}/{$}", serveResultsPage)

	// mux.HandleFunc("GET /home/{$}", serveHomePage) ?

	mux.HandleFunc("POST /rank/update", handleRankUpdate)
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

func handleRankUpdate(w http.ResponseWriter, r *http.Request) {
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
	mu.Lock()
	defer mu.Unlock()

	// Check if user has actually ranked, if not redirect to "/rank/{id}"

	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "results.html",
		"Data": userId,
	})
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

// func getCollectionWithItems(id int) Collection {
// 	var col Collection

// 	err := db.QueryRow("SELECT name FROM collections WHERE id = ?", id).Scan(col.Name)
// 	if err != nil {
// 		log.Print(err)
// 		return col
// 	}
// 	col.Id, col.Items = id, getCollectionItems(id)
// 	return col
// }

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
