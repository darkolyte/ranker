package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

//

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
	);`

	_, err = DB.Exec(tLC)
	if err != nil {
		log.Fatalf("terror in creation of table: %q", err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/calluponthecreator", creationHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

type Collection struct {
	ID   int
	Name string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := DB.Query("SELECT id, name FROM collections")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var collections []Collection

	for rows.Next() {
		var col Collection
		if err := rows.Scan(&col.ID, &col.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
