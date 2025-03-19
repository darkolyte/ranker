package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	Title string
	Wins  int
}

var tpl *template.Template
var db *sql.DB

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
		name TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER NOT NULL PRIMARY KEY,
		image TEXT,
		title TEXT NOT NULL,
		wins INTEGER DEFAULT 0,
		collection_id INTEGER,
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
	mux.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Path[1:]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			serveErrorPage(w, r)
			return
		} // q
		http.ServeFile(w, r, filePath)
	}))

	mux.HandleFunc("/", serveErrorPage)
	mux.HandleFunc("GET /{$}", handleRoot)
	mux.HandleFunc("GET /home/{$}", serveHomePage)
	mux.HandleFunc("GET /rank/{id}/{$}", serveRankPage)
}

func serveErrorPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "error.html",
		"Data": []string{"Error on the wall", "Here we are again"},
	})
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/home", http.StatusFound)
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "home.html",
		"Data": getCollectionsWithItems(),
	})
}

func serveRankPage(w http.ResponseWriter, r *http.Request) {
	collectionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	parseTemplates() // for testing
	tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "rank.html",
		"Data": getCollectionItems(collectionId),
	})
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

	rows, err := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", id)
	if err != nil {
		log.Print(err)
		return items
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Id, &item.Image, &item.Title); err != nil {
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
