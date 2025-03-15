package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type Collection struct {
	ID    int
	Name  string
	Items []Item
}

type Item struct {
	ID    int
	Image string
	Title string
	Wins  int
}

var tpl *template.Template

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tLC := `
	CREATE TABLE IF NOT EXISTS collections (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS items (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		image TEXT,
		title TEXT NOT NULL,
		collection_id INTEGER,
		FOREIGN KEY (collection_id) REFERENCES collections(id)
	);`

	_, err = db.Exec(tLC)
	if err != nil {
		log.Fatalf("terror in creation of table: %q", err)
	}

	parseTemplates()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/calluponthecreator", creationHandler)
	http.HandleFunc("/calluponthecreator/create-item", createItemHandler)
	http.HandleFunc("/collections/", collectionsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func parseTemplates() {
	tmp := template.New("")

	funcs := template.FuncMap{
		"embed": func(name string, data any) template.HTML {
			var out strings.Builder
			if err := tmp.ExecuteTemplate(&out, name, data); err != nil {
				log.Println(err)
			}
			return template.HTML(out.String())
		}}

	tpl = template.Must(tmp.Funcs(funcs).ParseGlob("tmpl/*.html"))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	collectionsRows, err := db.Query("SELECT id, name FROM collections")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer collectionsRows.Close()

	var collections []Collection

	for collectionsRows.Next() {
		var col Collection
		if err := collectionsRows.Scan(&col.ID, &col.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		itemsRows, err := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", col.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer itemsRows.Close()

		for itemsRows.Next() {
			var item Item
			if err := itemsRows.Scan(&item.ID, &item.Image, &item.Title); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			col.Items = append(col.Items, item)
		}

		collections = append(collections, col)
	}

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "home.html",
		"Data": collections,
	})
	if err != nil {
		log.Println("Template error:", err)
		return
	}
}

func creationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Har har~ We differ in methodology but could we still be friends?", http.StatusMethodNotAllowed)
	}
	name := r.FormValue("name") // skipped validation
	_, err := db.Exec("INSERT INTO collections(name) VALUES(?)", name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func createItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Error due to provision of fictional methodology", http.StatusMethodNotAllowed)
	}
	collectionID, err := strconv.Atoi(r.FormValue("collection_id"))
	if err != nil {
		log.Print(collectionID)
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	} // missing db call

	image := r.FormValue("image")
	title := r.FormValue("title")
	_, err = db.Exec("INSERT INTO items (image, title, collection_id) VALUES (?, ?, ?)", image, title, collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func collectionsHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/result") {
		collectionResultHandler(w, r)
		return
	}

	collectionID, err := strconv.Atoi(r.URL.Path[len("/collections/"):])
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	} // missing db call

	var col Collection
	_ = db.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionID).Scan(&col.ID, &col.Name) // skipped validation

	itemRows, _ := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", collectionID) // skipped validation

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.ID, &item.Image, &item.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		col.Items = append(col.Items, item)
	}

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "rank.html",
		"Data": col,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func collectionResultHandler(w http.ResponseWriter, r *http.Request) {
	collectionID := strings.Split(r.URL.Path, "/")[2]

	var scores map[string]int

	_ = json.Unmarshal([]byte(r.FormValue("scores")), &scores)

	var res Collection
	_ = db.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionID).Scan(&res.ID, &res.Name)

	itemRows, err := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer itemRows.Close()

	var items []Item

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.ID, &item.Image, &item.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		item.Wins = scores[strconv.Itoa(item.ID)]

		items = append(items, item)

	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Wins > items[j].Wins
	})

	res.Items = items

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "result.html",
		"Data": res,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
