package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

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
}

var DB *sql.DB

func main() {
	var err error
	DB, err = sql.Open("sqlite", "file:test.db?mode=memory&cache=shared")
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

	t, _ := template.ParseFiles("index.html")
	t.Execute(w, collections)
}

func creationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Har har~ We differ in methodology but could we still be friends?", http.StatusMethodNotAllowed)
	}
	name := r.FormValue("name")
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
	collectionID, _ := strconv.Atoi(r.FormValue("collection_id"))
	image := r.FormValue("image")
	title := r.FormValue("title")
	_, err := DB.Exec("INSERT INTO items (image, title, collection_id) VALUES (?, ?, ?)", image, title, collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
