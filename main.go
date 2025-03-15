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

var DB *sql.DB

func main() {
	var err error
	DB, err = sql.Open("sqlite", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer DB.Close()

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

	_, err = DB.Exec(tLC)
	if err != nil {
		log.Fatalf("terror in creation of table: %q", err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/calluponthecreator", creationHandler)
	http.HandleFunc("/calluponthecreator/create-item", createItemHandler)
	http.HandleFunc("/collections/", collectionsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	collectionsRows, err := DB.Query("SELECT id, name FROM collections")
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

		itemsRows, err := DB.Query("SELECT id, image, title FROM items WHERE collection_id = ?", col.ID)
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

	t, _ := template.ParseFiles("index.html") // skipped validation
	t.Execute(w, collections)
}

func creationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Har har~ We differ in methodology but could we still be friends?", http.StatusMethodNotAllowed)
	}
	name := r.FormValue("name") // skipped validation
	_, err := DB.Exec("INSERT INTO collections(name) VALUES(?)", name)
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
	collectionID, _ := strconv.Atoi(r.FormValue("collection_id")) // skipped validation
	image := r.FormValue("image")                                 // skipped validation
	title := r.FormValue("title")                                 // skipped validation
	_, err := DB.Exec("INSERT INTO items (image, title, collection_id) VALUES (?, ?, ?)", image, title, collectionID)
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

	collectionID, _ := strconv.Atoi(r.URL.Path[len("/collections/"):]) // skipped validation

	var col Collection
	_ = DB.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionID).Scan(&col.ID, &col.Name) // skipped validation

	itemRows, _ := DB.Query("SELECT id, image, title FROM items WHERE collection_id = ?", collectionID) // skipped validation

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.ID, &item.Image, &item.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		col.Items = append(col.Items, item)
	}

	t, _ := template.ParseFiles("rank.html") // skippped validation
	t.Execute(w, col)
}

func collectionResultHandler(w http.ResponseWriter, r *http.Request) {
	collectionID := strings.Split(r.URL.Path, "/")[2]

	var scores map[string]int

	_ = json.Unmarshal([]byte(r.FormValue("scores")), &scores)

	var res Collection
	_ = DB.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionID).Scan(&res.ID, &res.Name)

	itemRows, _ := DB.Query("SELECT id, image, title FROM items WHERE collection_id = ?", collectionID)

	var items []Item

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.ID, &item.Image, &item.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		item.Wins = scores[strconv.Itoa(item.ID)]

		items = append(items, item)

		sort.Slice(items, func(i, j int) bool {
			return items[i].Wins > items[j].Wins
		})

	}

	res.Items = items

	t, _ := template.ParseFiles("result.html")
	t.Execute(w, res)
}
